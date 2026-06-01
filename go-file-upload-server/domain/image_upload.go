package domain

import (
    "errors"
    "time"

    "github.com/google/uuid"
)

type ImageUpload struct {
    Id           string     `json:"id"`
    JobID        *string    `json:"jobId,omitempty"`
    TaskID       *string    `json:"taskId,omitempty"`
    OwnerID      string     `json:"ownerId"`
    OriginalName string     `json:"originalName"`
    Extension    string     `json:"extension"`
    UploadUUID   string     `json:"uploadUuid"`
    PhotoType    string     `json:"photoType"`
    SizeBytes    int64      `json:"sizeBytes"`
    MimeType     string     `json:"mimeType"`
    CreatedAt    time.Time  `json:"createdAt"`
}

func NewImageUpload(ownerID string, taskID *string, jobID *string, originalName, extension, photoType string, sizeBytes int64, mimeType string) (*ImageUpload, error) {
    if ownerID == "" {
        return nil, errors.New("owner id is required")
    }
    if taskID == nil && jobID == nil {
        return nil, errors.New("either taskId or jobId is required")
    }
    if originalName == "" {
        return nil, errors.New("original name is required")
    }
    if extension == "" {
        return nil, errors.New("extension is required")
    }
    if photoType == "" {
        return nil, errors.New("photo type is required")
    }
    if sizeBytes < 0 {
        return nil, errors.New("size bytes must be non-negative")
    }

    return &ImageUpload{
        Id:           uuid.NewString(),
        JobID:        jobID,
        TaskID:       taskID,
        OwnerID:      ownerID,
        OriginalName: originalName,
        Extension:    extension,
        UploadUUID:   uuid.NewString(),
        PhotoType:    photoType,
        SizeBytes:    sizeBytes,
        MimeType:     mimeType,
        CreatedAt:    time.Now(),
    }, nil
}
