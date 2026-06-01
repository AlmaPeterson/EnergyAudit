package fileupload

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go-file-upload-server/domain"

	_ "github.com/lib/pq"
)

type PostgresFileUploadRepository struct {
	db      *sql.DB
	Config  PostgresFileUploadRepositoryConfig
	Secrets PostgresFileUploadRepositorySecrets
}

type PostgresFileUploadRepositoryConfig struct {
	UploadPath    string `json:"uploadPath"`
	ServerAddress string `json:"serverAddress"` // e.g. "localhost:5432"
	DatabaseName  string `json:"databaseName"`
	TableName     string `json:"tableName"`
}

type PostgresFileUploadRepositorySecrets struct {
	DbUsername string `json:"dbUsername"`
	DbPassword string `json:"dbPassword"`
}

type postgresConfigFile struct {
	RepositoryType string                             `json:"repositoryType"`
	Config         PostgresFileUploadRepositoryConfig `json:"config"`
}

type postgresSecretsFile struct {
	RepositoryType string                              `json:"repositoryType"`
	Secrets        PostgresFileUploadRepositorySecrets `json:"secrets"`
}

func LoadConfigFromJson(filePath string) (PostgresFileUploadRepositoryConfig, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return PostgresFileUploadRepositoryConfig{}, err
	}

	var configWrapper postgresConfigFile
	if err := json.Unmarshal(bytes, &configWrapper); err != nil {
		return PostgresFileUploadRepositoryConfig{}, err
	}

	return configWrapper.Config, nil
}

func LoadSecretsFromJson(filePath string) (PostgresFileUploadRepositorySecrets, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return PostgresFileUploadRepositorySecrets{}, err
	}

	var secretsWrapper postgresSecretsFile
	if err := json.Unmarshal(bytes, &secretsWrapper); err != nil {
		return PostgresFileUploadRepositorySecrets{}, err
	}

	return secretsWrapper.Secrets, nil
}

func NewPostgresFileUploadRepository(config PostgresFileUploadRepositoryConfig, secrets PostgresFileUploadRepositorySecrets) (PostgresFileUploadRepository, error) {
	if config.UploadPath == "" {
		return PostgresFileUploadRepository{}, errors.New("upload path is required")
	}
	if config.ServerAddress == "" {
		return PostgresFileUploadRepository{}, errors.New("server address is required")
	}
	if config.DatabaseName == "" {
		return PostgresFileUploadRepository{}, errors.New("database name is required")
	}
	if secrets.DbUsername == "" {
		return PostgresFileUploadRepository{}, errors.New("database username is required")
	}

	if err := os.MkdirAll(config.UploadPath, 0o755); err != nil {
		return PostgresFileUploadRepository{}, err
	}

	host := config.ServerAddress
	port := ""
	if strings.Contains(config.ServerAddress, ":") {
		var err error
		host, port, err = net.SplitHostPort(config.ServerAddress)
		if err != nil {
			return PostgresFileUploadRepository{}, fmt.Errorf("invalid server address: %w", err)
		}
	}

	dsnParts := []string{
		fmt.Sprintf("host=%s", host),
		fmt.Sprintf("dbname=%s", config.DatabaseName),
		fmt.Sprintf("user=%s", secrets.DbUsername),
		fmt.Sprintf("password=%s", secrets.DbPassword),
		"sslmode=disable",
	}
	if port != "" {
		dsnParts = append(dsnParts, fmt.Sprintf("port=%s", port))
	}

	dsn := strings.Join(dsnParts, " ")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return PostgresFileUploadRepository{}, err
	}

	if err := db.Ping(); err != nil {
		return PostgresFileUploadRepository{}, err
	}

	repo := PostgresFileUploadRepository{
		db:      db,
		Config:  config,
		Secrets: secrets,
	}

	if err := repo.ensureSchema(); err != nil {
		return PostgresFileUploadRepository{}, err
	}

	// ensure legacy columns exist in case this DB was created from an older migration
	if err := repo.ensureColumns(); err != nil {
		return PostgresFileUploadRepository{}, err
	}

	return repo, nil
}

func (p PostgresFileUploadRepository) ensureSchema() error {
	tn, err := p.tableName()
	if err != nil {
		return err
	}

	// Create users table
	usersQuery := `CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	)`
	if _, err := p.db.Exec(usersQuery); err != nil {
		return err
	}

	// Create folders table
	foldersQuery := `CREATE TABLE IF NOT EXISTS folders (
		id TEXT PRIMARY KEY,
		owner_id TEXT NOT NULL,
		parent_id TEXT NULL,
		name TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL
	)`
	if _, err := p.db.Exec(foldersQuery); err != nil {
		return err
	}

	// Create file_uploads table (existing)
	uploadsQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id BIGSERIAL PRIMARY KEY,
		original_name TEXT NOT NULL,
		file_extension TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		folder_id TEXT NULL,
		upload_uuid TEXT NOT NULL UNIQUE,
		size_bytes BIGINT NOT NULL DEFAULT 0,
		mime_type TEXT,
		uploaded_at TIMESTAMPTZ NOT NULL
	)`, tn)
	if _, err := p.db.Exec(uploadsQuery); err != nil {
		return err
	}

	// Create file_shares table
	sharesQuery := `CREATE TABLE IF NOT EXISTS file_shares (
		id BIGSERIAL PRIMARY KEY,
		resource_type TEXT NOT NULL DEFAULT 'file',
		resource_id TEXT NOT NULL,
		owner_id TEXT NOT NULL,
		grantee_id TEXT NOT NULL,
		access_level TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL,
		file_id BIGINT
	)`
	if _, err := p.db.Exec(sharesQuery); err != nil {
		return err
	}

    // Create jobs table
    jobsQuery := `CREATE TABLE IF NOT EXISTS jobs (
        id TEXT PRIMARY KEY,
        owner_id TEXT NOT NULL,
        title TEXT NOT NULL,
        description TEXT,
        created_at TIMESTAMPTZ NOT NULL,
        updated_at TIMESTAMPTZ NOT NULL
    )`
    if _, err := p.db.Exec(jobsQuery); err != nil {
        return err
    }

    // Create tasks table
    tasksQuery := `CREATE TABLE IF NOT EXISTS tasks (
        id TEXT PRIMARY KEY,
        job_id TEXT NOT NULL,
        owner_id TEXT NOT NULL,
        title TEXT NOT NULL,
        description TEXT,
        status TEXT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL,
        updated_at TIMESTAMPTZ NOT NULL
    )`
    if _, err := p.db.Exec(tasksQuery); err != nil {
        return err
    }

    // Create time_entries table
    timeEntriesQuery := `CREATE TABLE IF NOT EXISTS time_entries (
        id TEXT PRIMARY KEY,
        task_id TEXT NOT NULL,
        user_id TEXT NOT NULL,
        start_time TIMESTAMPTZ NOT NULL,
        end_time TIMESTAMPTZ NULL,
        duration_minutes INT NOT NULL,
        note TEXT,
        created_at TIMESTAMPTZ NOT NULL
    )`
    if _, err := p.db.Exec(timeEntriesQuery); err != nil {
        return err
    }

    // Create energy_audits table
    energyAuditsQuery := `CREATE TABLE IF NOT EXISTS energy_audits (
        id TEXT PRIMARY KEY,
        task_id TEXT NOT NULL,
        job_id TEXT NOT NULL,
        user_id TEXT NOT NULL,
        easy BOOLEAN NOT NULL,
        hard BOOLEAN NOT NULL,
        fun BOOLEAN NOT NULL,
        not_fun BOOLEAN NOT NULL,
        efficiency_rating INT NOT NULL,
        notes TEXT,
        created_at TIMESTAMPTZ NOT NULL
    )`
    if _, err := p.db.Exec(energyAuditsQuery); err != nil {
        return err
    }

    // Create image_uploads table
    imageUploadsQuery := `CREATE TABLE IF NOT EXISTS image_uploads (
        id TEXT PRIMARY KEY,
        job_id TEXT NULL,
        task_id TEXT NULL,
        owner_id TEXT NOT NULL,
        original_name TEXT NOT NULL,
        extension TEXT NOT NULL,
        upload_uuid TEXT NOT NULL UNIQUE,
        photo_type TEXT NOT NULL,
        size_bytes BIGINT NOT NULL,
        mime_type TEXT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL,
        data BYTEA NOT NULL
    )`
    if _, err := p.db.Exec(imageUploadsQuery); err != nil {
        return err
    }

    return nil
}

func (p PostgresFileUploadRepository) ensureColumns() error {
    tn, err := p.tableName()
    if err != nil {
        return err
    }

    stmts := []string{
        fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS file_name TEXT", tn),
        fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS file_extension TEXT", tn),
        fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS owner_id TEXT", tn),
        fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS folder_id TEXT", tn),
        fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS upload_uuid TEXT", tn),
        fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS size_bytes BIGINT DEFAULT 0", tn),
        fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS mime_type TEXT", tn),
        fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS uploaded_at TIMESTAMPTZ", tn),
    }

    for _, s := range stmts {
        if _, err := p.db.Exec(s); err != nil {
            return err
        }
    }

    sharesStmts := []string{
        "ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS resource_type TEXT DEFAULT 'file'",
        "ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS resource_id TEXT",
        "ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS owner_id TEXT",
        "ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS grantee_id TEXT",
        "ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS access_level TEXT",
        "ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ",
        "ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS file_id BIGINT",
    }

    for _, s := range sharesStmts {
        if _, err := p.db.Exec(s); err != nil {
            return err
        }
    }

    if _, err := p.db.Exec(fmt.Sprintf("UPDATE %s SET file_name = original_name WHERE file_name IS NULL AND original_name IS NOT NULL", tn)); err != nil {
        return err
    }

    if _, err := p.db.Exec("UPDATE file_shares SET resource_id = file_id::text, resource_type = 'file' WHERE resource_id IS NULL AND file_id IS NOT NULL"); err != nil {
        return err
    }

    return nil
}

func (p PostgresFileUploadRepository) filePathForID(id int64, extension string) string {
	extension = strings.TrimPrefix(extension, ".")
	if extension != "" {
		return filepath.Join(p.Config.UploadPath, fmt.Sprintf("%d.%s", id, extension))
	}
	return filepath.Join(p.Config.UploadPath, fmt.Sprintf("%d", id))
}

func (p PostgresFileUploadRepository) tableName() (string, error) {
	name := p.Config.TableName
	if name == "" {
		name = "file_uploads"
	}
	// allow only alphanumeric and underscore
	valid := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !valid.MatchString(name) {
		return "", fmt.Errorf("invalid table name: %s", name)
	}
	return name, nil
}

func (p PostgresFileUploadRepository) UploadFile(fileUpload domain.FileUpload, fileData []byte) (domain.FileUpload, error) {
	if fileUpload.OriginalName == "" {
		return domain.FileUpload{}, errors.New("original name is required")
	}
	if fileUpload.Extension == "" {
		return domain.FileUpload{}, errors.New("file extension is required")
	}
	if len(fileData) == 0 {
		return domain.FileUpload{}, errors.New("file data cannot be empty")
	}

	fileUpload.Extension = strings.TrimPrefix(fileUpload.Extension, ".")

	tn, err := p.tableName()
	if err != nil {
		return domain.FileUpload{}, err
	}

	// Generate UUID for upload_uuid column
	uploadUUID := uuid.New().String()

	// Insert metadata first and get numeric id.
	// Populate both `original_name` and `file_name` to be compatible with databases
	// that may have one or the other as the canonical column.
	query := fmt.Sprintf(`INSERT INTO %s (original_name, file_name, file_extension, owner_id, folder_id, upload_uuid, size_bytes, mime_type, uploaded_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, tn)
	var newId int64
	var folderID interface{}
	if fileUpload.FolderID != nil {
		folderID = *fileUpload.FolderID
	} else {
		folderID = nil
	}

	err = p.db.QueryRow(query, fileUpload.OriginalName, fileUpload.OriginalName, fileUpload.Extension, fileUpload.OwnerID, folderID, uploadUUID, fileUpload.SizeBytes, fileUpload.MimeType, fileUpload.UploadedAt).Scan(&newId)
	if err != nil {
		return domain.FileUpload{}, err
	}

	// Now write file using id as storage key. If write fails, remove DB row.
	filePath := p.filePathForID(newId, fileUpload.Extension)
	if err := os.WriteFile(filePath, fileData, 0o644); err != nil {
		// cleanup DB row
		delQuery := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, tn)
		_, _ = p.db.Exec(delQuery, newId)
		return domain.FileUpload{}, err
	}

	fileUpload.Id = fmt.Sprintf("%d", newId)
	return fileUpload, nil
}

func (p PostgresFileUploadRepository) GetFileUploadByID(id string) (domain.FileUpload, error) {
	var fileUpload domain.FileUpload
	tn, err := p.tableName()
	if err != nil {
		return domain.FileUpload{}, err
	}
	// Convert string ID to int64 for database query
	parsedID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return domain.FileUpload{}, fmt.Errorf("invalid ID format")
	}

	var dbID int64
	var folderID sql.NullString
	// Read original name using COALESCE to support legacy schemas that may use `file_name`.
	query := fmt.Sprintf(`SELECT id, COALESCE(original_name, file_name) AS original_name, file_extension, owner_id, folder_id, size_bytes, mime_type, uploaded_at FROM %s WHERE id = $1`, tn)
	row := p.db.QueryRow(query, parsedID)
	if err := row.Scan(&dbID, &fileUpload.OriginalName, &fileUpload.Extension, &fileUpload.OwnerID, &folderID, &fileUpload.SizeBytes, &fileUpload.MimeType, &fileUpload.UploadedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.FileUpload{}, fmt.Errorf("file upload not found")
		}
		return domain.FileUpload{}, err
	}

	if folderID.Valid {
		v := folderID.String
		fileUpload.FolderID = &v
	} else {
		fileUpload.FolderID = nil
	}

	// Convert int64 ID back to string for domain model
	fileUpload.Id = fmt.Sprintf("%d", dbID)
	return fileUpload, nil
}

func (p PostgresFileUploadRepository) GetFileUploadDataByID(id string) ([]byte, error) {
	fileUpload, err := p.GetFileUploadByID(id)
	if err != nil {
		return nil, err
	}
	// storage path uses DB id
	parsedID, err := strconv.ParseInt(fileUpload.Id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid id")
	}
	filePath := p.filePathForID(parsedID, fileUpload.Extension)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (p PostgresFileUploadRepository) ListFileUploads() ([]domain.FileUpload, error) {
	tn, err := p.tableName()
	if err != nil {
		return nil, err
	}
	// Use COALESCE to read from either `original_name` or legacy `file_name`.
	query := fmt.Sprintf(`SELECT id, COALESCE(original_name, file_name) AS original_name, file_extension, owner_id, folder_id, size_bytes, mime_type, uploaded_at FROM %s ORDER BY uploaded_at DESC`, tn)
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	uploads := []domain.FileUpload{}
	for rows.Next() {
		var upload domain.FileUpload
		var dbID int64
		var folderID sql.NullString
		if err := rows.Scan(&dbID, &upload.OriginalName, &upload.Extension, &upload.OwnerID, &folderID, &upload.SizeBytes, &upload.MimeType, &upload.UploadedAt); err != nil {
			return nil, err
		}
		// Convert int64 ID back to string for domain model
		upload.Id = fmt.Sprintf("%d", dbID)
		if folderID.Valid {
			v := folderID.String
			upload.FolderID = &v
		}
		uploads = append(uploads, upload)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return uploads, nil
}

func (p PostgresFileUploadRepository) DeleteFileUpload(id string) error {
	fileUpload, err := p.GetFileUploadByID(id)
	if err != nil {
		return err
	}
	// storage path uses DB id
	parsedID, err := strconv.ParseInt(fileUpload.Id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id")
	}
	filePath := p.filePathForID(parsedID, fileUpload.Extension)
	if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	tn, err := p.tableName()
	if err != nil {
		return err
	}

	// Convert string ID to int64 for database query
	parsedID, err = strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID format")
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, tn)
	_, err = p.db.Exec(query, parsedID)
	if err != nil {
		return err
	}

	return nil
}

func (p PostgresFileUploadRepository) CreateFolder(folder domain.Folder) (domain.Folder, error) {
	query := `INSERT INTO folders (id, owner_id, parent_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
	var parentID interface{}
	if folder.ParentID != nil {
		parentID = *folder.ParentID
	}

	_, err := p.db.Exec(query, folder.Id, folder.OwnerID, parentID, folder.Name, folder.CreatedAt, folder.UpdatedAt)
	if err != nil {
		return domain.Folder{}, err
	}

	return folder, nil
}

func (p PostgresFileUploadRepository) ListFoldersForUser(userID string, parentID *string) ([]domain.Folder, error) {
	query := `SELECT id, owner_id, parent_id, name, created_at, updated_at FROM folders WHERE owner_id = $1`
	args := []interface{}{userID}

	if parentID == nil {
		query += ` AND parent_id IS NULL`
	} else {
		query += ` AND parent_id = $2`
		args = append(args, *parentID)
	}

	query += ` ORDER BY name`

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	folders := []domain.Folder{}
	for rows.Next() {
		var folder domain.Folder
		var parentID sql.NullString
		if err := rows.Scan(&folder.Id, &folder.OwnerID, &parentID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			v := parentID.String
			folder.ParentID = &v
		}
		folders = append(folders, folder)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return folders, nil
}

func (p PostgresFileUploadRepository) GetFolderByID(id string) (domain.Folder, error) {
	var folder domain.Folder
	row := p.db.QueryRow(`SELECT id, owner_id, parent_id, name, created_at, updated_at FROM folders WHERE id = $1`, id)

	var parentID sql.NullString
	if err := row.Scan(&folder.Id, &folder.OwnerID, &parentID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Folder{}, fmt.Errorf("folder not found")
		}
		return domain.Folder{}, err
	}

	if parentID.Valid {
		v := parentID.String
		folder.ParentID = &v
	}

	return folder, nil
}

func (p PostgresFileUploadRepository) DeleteFolder(id string) error {
	var count int
	if err := p.db.QueryRow(`SELECT COUNT(1) FROM folders WHERE parent_id = $1`, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("folder is not empty")
	}

	tn, err := p.tableName()
	if err != nil {
		return err
	}

	if err := p.db.QueryRow(fmt.Sprintf(`SELECT COUNT(1) FROM %s WHERE folder_id = $1`, tn), id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("folder is not empty")
	}

	_, err = p.db.Exec(`DELETE FROM folders WHERE id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}

func (p PostgresFileUploadRepository) ListFileUploadsByFolder(ownerID string, folderID *string) ([]domain.FileUpload, error) {
	tn, err := p.tableName()
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT id, COALESCE(original_name, file_name) AS original_name, file_extension, owner_id, folder_id, size_bytes, mime_type, uploaded_at FROM %s WHERE owner_id = $1`, tn)
	args := []interface{}{ownerID}

	if folderID == nil {
		query += ` AND folder_id IS NULL`
	} else {
		query += ` AND folder_id = $2`
		args = append(args, *folderID)
	}

	query += ` ORDER BY uploaded_at DESC`

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	uploads := []domain.FileUpload{}
	for rows.Next() {
		var upload domain.FileUpload
		var dbID int64
		var folderID sql.NullString
		if err := rows.Scan(&dbID, &upload.OriginalName, &upload.Extension, &upload.OwnerID, &folderID, &upload.SizeBytes, &upload.MimeType, &upload.UploadedAt); err != nil {
			return nil, err
		}
		upload.Id = fmt.Sprintf("%d", dbID)
		if folderID.Valid {
			v := folderID.String
			upload.FolderID = &v
		}
		uploads = append(uploads, upload)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return uploads, nil
}

func (p PostgresFileUploadRepository) CreateJob(job domain.Job) (domain.Job, error) {
    query := `INSERT INTO jobs (id, owner_id, title, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
    _, err := p.db.Exec(query, job.Id, job.OwnerID, job.Title, job.Description, job.CreatedAt, job.UpdatedAt)
    if err != nil {
        return domain.Job{}, err
    }
    return job, nil
}

func (p PostgresFileUploadRepository) ListJobs() ([]domain.Job, error) {
    rows, err := p.db.Query(`SELECT id, owner_id, title, description, created_at, updated_at FROM jobs ORDER BY created_at DESC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    jobs := []domain.Job{}
    for rows.Next() {
        var job domain.Job
        if err := rows.Scan(&job.Id, &job.OwnerID, &job.Title, &job.Description, &job.CreatedAt, &job.UpdatedAt); err != nil {
            return nil, err
        }
        jobs = append(jobs, job)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return jobs, nil
}

func (p PostgresFileUploadRepository) GetJobByID(id string) (domain.Job, error) {
    var job domain.Job
    row := p.db.QueryRow(`SELECT id, owner_id, title, description, created_at, updated_at FROM jobs WHERE id = $1`, id)
    if err := row.Scan(&job.Id, &job.OwnerID, &job.Title, &job.Description, &job.CreatedAt, &job.UpdatedAt); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return domain.Job{}, fmt.Errorf("job not found")
        }
        return domain.Job{}, err
    }
    return job, nil
}

func (p PostgresFileUploadRepository) UpdateJob(job domain.Job) (domain.Job, error) {
    query := `UPDATE jobs SET title = $1, description = $2, updated_at = $3 WHERE id = $4`
    _, err := p.db.Exec(query, job.Title, job.Description, job.UpdatedAt, job.Id)
    if err != nil {
        return domain.Job{}, err
    }
    return job, nil
}

func (p PostgresFileUploadRepository) CreateTask(task domain.Task) (domain.Task, error) {
    query := `INSERT INTO tasks (id, job_id, owner_id, title, description, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
    _, err := p.db.Exec(query, task.Id, task.JobID, task.OwnerID, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt)
    if err != nil {
        return domain.Task{}, err
    }
    return task, nil
}

func (p PostgresFileUploadRepository) ListTasksByJob(jobID string) ([]domain.Task, error) {
    rows, err := p.db.Query(`SELECT id, job_id, owner_id, title, description, status, created_at, updated_at FROM tasks WHERE job_id = $1 ORDER BY created_at DESC`, jobID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    tasks := []domain.Task{}
    for rows.Next() {
        var task domain.Task
        if err := rows.Scan(&task.Id, &task.JobID, &task.OwnerID, &task.Title, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt); err != nil {
            return nil, err
        }
        tasks = append(tasks, task)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return tasks, nil
}

func (p PostgresFileUploadRepository) GetTaskByID(id string) (domain.Task, error) {
    var task domain.Task
    row := p.db.QueryRow(`SELECT id, job_id, owner_id, title, description, status, created_at, updated_at FROM tasks WHERE id = $1`, id)
    if err := row.Scan(&task.Id, &task.JobID, &task.OwnerID, &task.Title, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return domain.Task{}, fmt.Errorf("task not found")
        }
        return domain.Task{}, err
    }
    return task, nil
}

func (p PostgresFileUploadRepository) UpdateTask(task domain.Task) (domain.Task, error) {
    query := `UPDATE tasks SET title = $1, description = $2, status = $3, updated_at = $4 WHERE id = $5`
    _, err := p.db.Exec(query, task.Title, task.Description, task.Status, task.UpdatedAt, task.Id)
    if err != nil {
        return domain.Task{}, err
    }
    return task, nil
}

func (p PostgresFileUploadRepository) CreateTimeEntry(entry domain.TimeEntry) (domain.TimeEntry, error) {
    query := `INSERT INTO time_entries (id, task_id, user_id, start_time, end_time, duration_minutes, note, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
    _, err := p.db.Exec(query, entry.Id, entry.TaskID, entry.UserID, entry.StartTime, entry.EndTime, entry.DurationMinutes, entry.Note, entry.CreatedAt)
    if err != nil {
        return domain.TimeEntry{}, err
    }
    return entry, nil
}

func (p PostgresFileUploadRepository) GetTimeEntryByID(id string) (domain.TimeEntry, error) {
    var entry domain.TimeEntry
    row := p.db.QueryRow(`SELECT te.id, te.task_id, te.user_id, u.first_name, te.start_time, te.end_time, te.duration_minutes, te.note, te.created_at FROM time_entries te JOIN users u ON te.user_id = u.id WHERE te.id = $1`, id)
    if err := row.Scan(&entry.Id, &entry.TaskID, &entry.UserID, &entry.UserFirstName, &entry.StartTime, &entry.EndTime, &entry.DurationMinutes, &entry.Note, &entry.CreatedAt); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return domain.TimeEntry{}, fmt.Errorf("time entry not found")
        }
        return domain.TimeEntry{}, err
    }
    return entry, nil
}

func (p PostgresFileUploadRepository) GetActiveTimeEntry(taskID, userID string) (domain.TimeEntry, error) {
    var entry domain.TimeEntry
    row := p.db.QueryRow(`SELECT te.id, te.task_id, te.user_id, u.first_name, te.start_time, te.end_time, te.duration_minutes, te.note, te.created_at FROM time_entries te JOIN users u ON te.user_id = u.id WHERE te.task_id = $1 AND te.user_id = $2 AND te.end_time IS NULL ORDER BY te.start_time DESC LIMIT 1`, taskID, userID)
    if err := row.Scan(&entry.Id, &entry.TaskID, &entry.UserID, &entry.UserFirstName, &entry.StartTime, &entry.EndTime, &entry.DurationMinutes, &entry.Note, &entry.CreatedAt); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return domain.TimeEntry{}, fmt.Errorf("active time entry not found")
        }
        return domain.TimeEntry{}, err
    }
    return entry, nil
}

func (p PostgresFileUploadRepository) UpdateTimeEntry(entry domain.TimeEntry) (domain.TimeEntry, error) {
    query := `UPDATE time_entries SET start_time = $1, end_time = $2, duration_minutes = $3, note = $4 WHERE id = $5`
    _, err := p.db.Exec(query, entry.StartTime, entry.EndTime, entry.DurationMinutes, entry.Note, entry.Id)
    if err != nil {
        return domain.TimeEntry{}, err
    }
    return entry, nil
}

func (p PostgresFileUploadRepository) ListTimeEntriesByTask(taskID string) ([]domain.TimeEntry, error) {
    rows, err := p.db.Query(`SELECT te.id, te.task_id, te.user_id, u.first_name, te.start_time, te.end_time, te.duration_minutes, te.note, te.created_at FROM time_entries te JOIN users u ON te.user_id = u.id WHERE te.task_id = $1 ORDER BY te.start_time DESC`, taskID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    entries := []domain.TimeEntry{}
    for rows.Next() {
        var entry domain.TimeEntry
        if err := rows.Scan(&entry.Id, &entry.TaskID, &entry.UserID, &entry.UserFirstName, &entry.StartTime, &entry.EndTime, &entry.DurationMinutes, &entry.Note, &entry.CreatedAt); err != nil {
            return nil, err
        }
        entries = append(entries, entry)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return entries, nil
}

func (p PostgresFileUploadRepository) ListTimeEntriesByUser(userID string) ([]domain.TimeEntry, error) {
    rows, err := p.db.Query(`SELECT te.id, te.task_id, te.user_id, u.first_name, te.start_time, te.end_time, te.duration_minutes, te.note, te.created_at FROM time_entries te JOIN users u ON te.user_id = u.id WHERE te.user_id = $1 ORDER BY te.start_time DESC`, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    entries := []domain.TimeEntry{}
    for rows.Next() {
        var entry domain.TimeEntry
        if err := rows.Scan(&entry.Id, &entry.TaskID, &entry.UserID, &entry.UserFirstName, &entry.StartTime, &entry.EndTime, &entry.DurationMinutes, &entry.Note, &entry.CreatedAt); err != nil {
            return nil, err
        }
        entries = append(entries, entry)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return entries, nil
}

func (p PostgresFileUploadRepository) CreateEnergyAudit(audit domain.EnergyAudit) (domain.EnergyAudit, error) {
    query := `INSERT INTO energy_audits (id, task_id, job_id, user_id, easy, hard, fun, not_fun, efficiency_rating, notes, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
    _, err := p.db.Exec(query, audit.Id, audit.TaskID, audit.JobID, audit.UserID, audit.Easy, audit.Hard, audit.Fun, audit.NotFun, audit.EfficiencyRating, audit.Notes, audit.CreatedAt)
    if err != nil {
        return domain.EnergyAudit{}, err
    }
    return audit, nil
}

func (p PostgresFileUploadRepository) ListEnergyAuditsByTask(taskID string) ([]domain.EnergyAudit, error) {
    rows, err := p.db.Query(`SELECT ea.id, ea.task_id, ea.job_id, ea.user_id, u.first_name, ea.easy, ea.hard, ea.fun, ea.not_fun, ea.efficiency_rating, ea.notes, ea.created_at FROM energy_audits ea JOIN users u ON ea.user_id = u.id WHERE ea.task_id = $1 ORDER BY ea.created_at DESC`, taskID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    audits := []domain.EnergyAudit{}
    for rows.Next() {
        var audit domain.EnergyAudit
        if err := rows.Scan(&audit.Id, &audit.TaskID, &audit.JobID, &audit.UserID, &audit.UserFirstName, &audit.Easy, &audit.Hard, &audit.Fun, &audit.NotFun, &audit.EfficiencyRating, &audit.Notes, &audit.CreatedAt); err != nil {
            return nil, err
        }
        audits = append(audits, audit)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return audits, nil
}

func (p PostgresFileUploadRepository) UploadImage(image domain.ImageUpload, imageData []byte) (domain.ImageUpload, error) {
    query := `INSERT INTO image_uploads (id, job_id, task_id, owner_id, original_name, extension, upload_uuid, photo_type, size_bytes, mime_type, created_at, data) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
    _, err := p.db.Exec(query, image.Id, image.JobID, image.TaskID, image.OwnerID, image.OriginalName, image.Extension, image.UploadUUID, image.PhotoType, image.SizeBytes, image.MimeType, image.CreatedAt, imageData)
    if err != nil {
        return domain.ImageUpload{}, err
    }
    return image, nil
}

func (p PostgresFileUploadRepository) ListImageUploadsByTask(taskID string) ([]domain.ImageUpload, error) {
    rows, err := p.db.Query(`SELECT id, job_id, task_id, owner_id, original_name, extension, upload_uuid, photo_type, size_bytes, mime_type, created_at FROM image_uploads WHERE task_id = $1 ORDER BY created_at DESC`, taskID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    uploads := []domain.ImageUpload{}
    for rows.Next() {
        var upload domain.ImageUpload
        if err := rows.Scan(&upload.Id, &upload.JobID, &upload.TaskID, &upload.OwnerID, &upload.OriginalName, &upload.Extension, &upload.UploadUUID, &upload.PhotoType, &upload.SizeBytes, &upload.MimeType, &upload.CreatedAt); err != nil {
            return nil, err
        }
        uploads = append(uploads, upload)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return uploads, nil
}

func (p PostgresFileUploadRepository) GetImageUploadByID(id string) (domain.ImageUpload, error) {
    var upload domain.ImageUpload
    row := p.db.QueryRow(`SELECT id, job_id, task_id, owner_id, original_name, extension, upload_uuid, photo_type, size_bytes, mime_type, created_at FROM image_uploads WHERE id = $1`, id)
    if err := row.Scan(&upload.Id, &upload.JobID, &upload.TaskID, &upload.OwnerID, &upload.OriginalName, &upload.Extension, &upload.UploadUUID, &upload.PhotoType, &upload.SizeBytes, &upload.MimeType, &upload.CreatedAt); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return domain.ImageUpload{}, fmt.Errorf("image upload not found")
        }
        return domain.ImageUpload{}, err
    }
    return upload, nil
}

func (p PostgresFileUploadRepository) GetImageUploadDataByID(id string) ([]byte, error) {
    var data []byte
    row := p.db.QueryRow(`SELECT data FROM image_uploads WHERE id = $1`, id)
    if err := row.Scan(&data); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fmt.Errorf("image upload not found")
        }
        return nil, err
    }
    return data, nil
}

var _ domain.FileUploadRepository = PostgresFileUploadRepository{}
