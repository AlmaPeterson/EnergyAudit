-- Migration: 001_create_tables.sql
-- Creates users, folders, file_uploads and file_shares tables

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS folders (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL,
    parent_id TEXT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS file_uploads (
    id BIGSERIAL PRIMARY KEY,
    original_name TEXT NOT NULL,
    file_extension TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    folder_id TEXT NULL,
    upload_uuid TEXT NOT NULL UNIQUE,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    mime_type TEXT,
    uploaded_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_file_uploads_owner ON file_uploads(owner_id);
CREATE INDEX IF NOT EXISTS idx_file_uploads_folder ON file_uploads(folder_id);

CREATE TABLE IF NOT EXISTS file_shares (
    id BIGSERIAL PRIMARY KEY,
    file_id BIGINT NOT NULL,
    owner_id TEXT NOT NULL,
    grantee_id TEXT NOT NULL,
    access_level TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_file_shares_file_id ON file_shares(file_id);
CREATE INDEX IF NOT EXISTS idx_file_shares_grantee ON file_shares(grantee_id);
