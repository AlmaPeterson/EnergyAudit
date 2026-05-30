package domain

import (
    "errors"
    "time"

    "github.com/google/uuid"
)

type FileShare struct {
    Id          string    `json:"id"`
    FileID      string    `json:"fileId"`
    OwnerID     string    `json:"ownerId"`
    GranteeID   string    `json:"granteeId"`
    AccessLevel string    `json:"accessLevel"` // e.g. "read", "write", "delete"
    CreatedAt   time.Time `json:"createdAt"`
}

func NewFileShare(fileID, ownerID, granteeID, accessLevel string) (*FileShare, error) {
    if fileID == "" {
        return nil, errors.New("file ID cannot be empty")
    }
    if ownerID == "" {
        return nil, errors.New("owner ID cannot be empty")
    }
    if granteeID == "" {
        return nil, errors.New("grantee ID cannot be empty")
    }
    if accessLevel == "" {
        return nil, errors.New("access level cannot be empty")
    }

    return &FileShare{
        Id:          uuid.NewString(),
        FileID:      fileID,
        OwnerID:     ownerID,
        GranteeID:   granteeID,
        AccessLevel: accessLevel,
        CreatedAt:   time.Now(),
    }, nil
}
