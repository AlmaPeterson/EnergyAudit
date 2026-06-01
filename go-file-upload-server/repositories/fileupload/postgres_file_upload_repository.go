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
		file_id BIGINT NOT NULL,
		owner_id TEXT NOT NULL,
		grantee_id TEXT NOT NULL,
		access_level TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL
	)`
	if _, err := p.db.Exec(sharesQuery); err != nil {
		return err
	}

	return nil
}

// ensureColumns makes sure expected columns exist in legacy tables (safe to run repeatedly).
func (p PostgresFileUploadRepository) ensureColumns() error {
	tn, err := p.tableName()
	if err != nil {
		return err
	}

	// Add commonly-missing columns if they don't exist. Use IF NOT EXISTS where possible.
	// Note: Postgres supports ADD COLUMN IF NOT EXISTS.
	stmts := []string{
		// original_name and file_extension may be missing in older schemas
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS original_name TEXT", tn),
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

	// Ensure file_shares columns exist as well
	sharesStmts := []string{
		"ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS file_id BIGINT",
		"ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS owner_id TEXT",
		"ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS grantee_id TEXT",
		"ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS access_level TEXT",
		"ALTER TABLE IF EXISTS file_shares ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ",
	}
	for _, s := range sharesStmts {
		if _, err := p.db.Exec(s); err != nil {
			return err
		}
	}

	// Backfill legacy column `file_name` from `original_name` if present
	if _, err := p.db.Exec(fmt.Sprintf("UPDATE %s SET file_name = original_name WHERE file_name IS NULL AND original_name IS NOT NULL", tn)); err != nil {
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

var _ domain.FileUploadRepository = PostgresFileUploadRepository{}
