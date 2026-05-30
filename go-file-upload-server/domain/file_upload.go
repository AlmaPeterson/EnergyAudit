package domain

import (
	"errors"
	"time"
)

type FileUpload struct {
	Id           string     `json:"id"`
	OriginalName string     `json:"originalName"`
	Extension    string     `json:"extension"`
	OwnerID      string     `json:"ownerId"`
	FolderID     *string    `json:"folderId,omitempty"`
	// StorageKey is derived from the DB-generated Id and extension; do not store user-controlled paths here.
	SizeBytes    int64      `json:"sizeBytes"`
	MimeType     string     `json:"mimeType"`
	UploadedAt   time.Time  `json:"uploadedAt"`
}

func NewFileUpload(originalName, extension, ownerID string, folderID *string, sizeBytes int64, mimeType string) (*FileUpload, error) {
	if originalName == "" {
		return nil, errors.New("original name cannot be empty")
	}
	if extension == "" {
		return nil, errors.New("extension cannot be empty")
	}
	if ownerID == "" {
		return nil, errors.New("owner ID cannot be empty")
	}
	if sizeBytes < 0 {
		return nil, errors.New("sizeBytes cannot be negative")
	}

	// Do not generate the DB id here — the repository should insert the record and return the assigned id.
	return &FileUpload{
		Id:           "",
		OriginalName: originalName,
		Extension:    extension,
		OwnerID:      ownerID,
		FolderID:     folderID,
		SizeBytes:    sizeBytes,
		MimeType:     mimeType,
		UploadedAt:   time.Now(),
	}, nil
}

// StorageKey returns the storage filename/key to use when saving the file contents.
// It uses the DB-assigned Id; if Id is empty this returns an empty string.
func (f *FileUpload) StorageKey() string {
	if f == nil || f.Id == "" {
		return ""
	}
	if f.Extension != "" {
		return f.Id + "." + f.Extension
	}
	return f.Id
}
