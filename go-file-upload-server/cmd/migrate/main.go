package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"

	"go-file-upload-server/repositories/fileupload"

	_ "github.com/lib/pq"
)

func main() {
	cfgPath := flag.String("config", "configs/file-upload-config.json", "path to file upload config")
	secretsPath := flag.String("secrets", "secrets/file-upload-secrets.json", "path to file upload secrets")
	flag.Parse()

	cfg, err := fileupload.LoadConfigFromJson(*cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	secrets, err := fileupload.LoadSecretsFromJson(*secretsPath)
	if err != nil {
		log.Fatalf("failed to load secrets: %v", err)
	}

	if err := ensureDatabase(cfg, secrets); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// instantiate repository which will create schema and upload path
	_, err = fileupload.NewPostgresFileUploadRepository(cfg, secrets)
	if err != nil {
		log.Fatalf("failed to initialize repository: %v", err)
	}

	fmt.Println("migration completed")
}

func ensureDatabase(cfg fileupload.PostgresFileUploadRepositoryConfig, secrets fileupload.PostgresFileUploadRepositorySecrets) error {
	// basic validation of database name to avoid SQL injection in CREATE DATABASE
	validName := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validName.MatchString(cfg.DatabaseName) {
		return errors.New("invalid database name")
	}

	host := cfg.ServerAddress
	port := ""
	if strings.Contains(cfg.ServerAddress, ":") {
		var err error
		host, port, err = net.SplitHostPort(cfg.ServerAddress)
		if err != nil {
			return fmt.Errorf("invalid server address: %w", err)
		}
	}

	dsnParts := []string{
		fmt.Sprintf("host=%s", host),
		fmt.Sprintf("user=%s", secrets.DbUsername),
		fmt.Sprintf("password=%s", secrets.DbPassword),
		"sslmode=disable",
		"dbname=postgres",
	}
	if port != "" {
		dsnParts = append(dsnParts, fmt.Sprintf("port=%s", port))
	}
	dsn := strings.Join(dsnParts, " ")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	var exists bool
	row := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DatabaseName)
	if err := row.Scan(&exists); err != nil {
		return err
	}
	if exists {
		fmt.Printf("database %s already exists\n", cfg.DatabaseName)
		return nil
	}

	createSQL := fmt.Sprintf("CREATE DATABASE %s", cfg.DatabaseName)
	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	fmt.Printf("created database %s\n", cfg.DatabaseName)
	return nil
}
