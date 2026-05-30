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

	"github.com/google/uuid"

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

	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id BIGSERIAL PRIMARY KEY,
		file_name TEXT NOT NULL,
		file_extension TEXT NOT NULL,
		upload_uuid TEXT NOT NULL UNIQUE,
		uploaded_at TIMESTAMPTZ NOT NULL
	)`, tn)

	_, err = p.db.Exec(query)
	return err
}

func (p PostgresFileUploadRepository) filePath(uploadUuid, extension string) string {
	extension = strings.TrimPrefix(extension, ".")
	return filepath.Join(p.Config.UploadPath, fmt.Sprintf("%s.%s", uploadUuid, extension))
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
	if fileUpload.FileName == "" {
		return domain.FileUpload{}, errors.New("file name is required")
	}
	if fileUpload.FileExtension == "" {
		return domain.FileUpload{}, errors.New("file extension is required")
	}
	if len(fileData) == 0 {
		return domain.FileUpload{}, errors.New("file data cannot be empty")
	}

	fileUpload.FileExtension = strings.TrimPrefix(fileUpload.FileExtension, ".")
	// ID will be assigned by the database (BIGSERIAL)
	if fileUpload.UploadUuid == "" {
		fileUpload.UploadUuid = uuid.NewString()
	}

	filePath := p.filePath(fileUpload.UploadUuid, fileUpload.FileExtension)
	if err := os.WriteFile(filePath, fileData, 0o644); err != nil {
		return domain.FileUpload{}, err
	}

	tn, err := p.tableName()
	if err != nil {
		_ = os.Remove(filePath)
		return domain.FileUpload{}, err
	}

	// Insert and return generated id (as int64), then convert to string for domain
	query := fmt.Sprintf(`INSERT INTO %s (file_name, file_extension, upload_uuid, uploaded_at) VALUES ($1, $2, $3, $4) RETURNING id`, tn)
	var newId int64
	err = p.db.QueryRow(query, fileUpload.FileName, fileUpload.FileExtension, fileUpload.UploadUuid, fileUpload.UploadedAt).Scan(&newId)
	if err != nil {
		_ = os.Remove(filePath)
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
	query := fmt.Sprintf(`SELECT id, file_name, file_extension, upload_uuid, uploaded_at FROM %s WHERE id = $1`, tn)
	row := p.db.QueryRow(query, parsedID)
	if err := row.Scan(&dbID, &fileUpload.FileName, &fileUpload.FileExtension, &fileUpload.UploadUuid, &fileUpload.UploadedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.FileUpload{}, fmt.Errorf("file upload not found")
		}
		return domain.FileUpload{}, err
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

	filePath := p.filePath(fileUpload.UploadUuid, fileUpload.FileExtension)
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
	query := fmt.Sprintf(`SELECT id, file_name, file_extension, upload_uuid, uploaded_at FROM %s ORDER BY uploaded_at DESC`, tn)
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	uploads := []domain.FileUpload{}
	for rows.Next() {
		var upload domain.FileUpload
		var dbID int64
		if err := rows.Scan(&dbID, &upload.FileName, &upload.FileExtension, &upload.UploadUuid, &upload.UploadedAt); err != nil {
			return nil, err
		}
		// Convert int64 ID back to string for domain model
		upload.Id = fmt.Sprintf("%d", dbID)
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

	filePath := p.filePath(fileUpload.UploadUuid, fileUpload.FileExtension)
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
