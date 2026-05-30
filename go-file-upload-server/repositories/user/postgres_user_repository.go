package user

import (
    "database/sql"
    "errors"
    "fmt"
    "net"
    "strings"

    "go-file-upload-server/domain"
    fileupload "go-file-upload-server/repositories/fileupload"

    _ "github.com/lib/pq"
)

type PostgresUserRepository struct {
    db *sql.DB
}

func NewPostgresUserRepository(config fileupload.PostgresFileUploadRepositoryConfig, secrets fileupload.PostgresFileUploadRepositorySecrets) (PostgresUserRepository, error) {
    host := config.ServerAddress
    port := ""
    if strings.Contains(config.ServerAddress, ":") {
        var err error
        host, port, err = net.SplitHostPort(config.ServerAddress)
        if err != nil {
            return PostgresUserRepository{}, fmt.Errorf("invalid server address: %w", err)
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
        return PostgresUserRepository{}, err
    }
    if err := db.Ping(); err != nil {
        return PostgresUserRepository{}, err
    }

    return PostgresUserRepository{db: db}, nil
}

func (r PostgresUserRepository) CreateUser(u *domain.User) error {
    if u == nil {
        return errors.New("user is required")
    }
    query := `INSERT INTO users (id, first_name, last_name, email, password_hash, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`
    _, err := r.db.Exec(query, u.Id, u.FirstName, u.LastName, u.Email, u.PasswordHash, u.CreatedAt, u.UpdatedAt)
    return err
}

func (r PostgresUserRepository) GetByEmail(email string) (*domain.User, error) {
    if email == "" {
        return nil, errors.New("email is required")
    }
    query := `SELECT id, first_name, last_name, email, password_hash, created_at, updated_at FROM users WHERE email = $1`
    row := r.db.QueryRow(query, email)
    var u domain.User
    if err := row.Scan(&u.Id, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return &u, nil
}

func (r PostgresUserRepository) GetByID(id string) (*domain.User, error) {
    if id == "" {
        return nil, errors.New("id is required")
    }
    query := `SELECT id, first_name, last_name, email, password_hash, created_at, updated_at FROM users WHERE id = $1`
    row := r.db.QueryRow(query, id)
    var u domain.User
    if err := row.Scan(&u.Id, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return &u, nil
}

var _ interface{} = PostgresUserRepository{}
