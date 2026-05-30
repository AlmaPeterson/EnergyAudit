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

	// Insert metadata first and get numeric id
	query := fmt.Sprintf(`INSERT INTO %s (original_name, file_extension, owner_id, folder_id, size_bytes, mime_type, uploaded_at) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, tn)
	var newId int64
	var folderID interface{}
	if fileUpload.FolderID != nil {
		folderID = *fileUpload.FolderID
	} else {
		folderID = nil
	}

	err = p.db.QueryRow(query, fileUpload.OriginalName, fileUpload.Extension, fileUpload.OwnerID, folderID, fileUpload.SizeBytes, fileUpload.MimeType, fileUpload.UploadedAt).Scan(&newId)
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
	query := fmt.Sprintf(`SELECT id, original_name, file_extension, owner_id, folder_id, size_bytes, mime_type, uploaded_at FROM %s WHERE id = $1`, tn)
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
	query := fmt.Sprintf(`SELECT id, original_name, file_extension, owner_id, folder_id, size_bytes, mime_type, uploaded_at FROM %s ORDER BY uploaded_at DESC`, tn)
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
	parsedID, err := strconv.ParseInt(id, 10, 64)
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

var _ domain.FileUploadRepository = PostgresFileUploadRepository{}
