package domain

import (
    "errors"
    "time"

    "github.com/google/uuid"
)

type FileShare struct {
    Id           string    `json:"id"`
    ResourceType string    `json:"resourceType"` // file or folder
    ResourceID   string    `json:"resourceId"`
    ResourceName string    `json:"resourceName,omitempty"`
    OwnerID      string    `json:"ownerId"`
    OwnerEmail   string    `json:"ownerEmail,omitempty"`
    GranteeID    string    `json:"granteeId"`
    GranteeEmail string    `json:"granteeEmail,omitempty"`
    AccessLevel  string    `json:"accessLevel"` // e.g. "read", "write", "delete"
    CreatedAt    time.Time `json:"createdAt"`
}

func NewFileShare(resourceType, resourceID, ownerID, granteeID, accessLevel string) (*FileShare, error) {
    if resourceType != "file" && resourceType != "folder" {
        return nil, errors.New("resource type must be either 'file' or 'folder'")
    }
    if resourceID == "" {
        return nil, errors.New("resource ID cannot be empty")
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
        Id:           uuid.NewString(),
        ResourceType: resourceType,
        ResourceID:   resourceID,
        OwnerID:      ownerID,
        GranteeID:    granteeID,
        AccessLevel:  accessLevel,
        CreatedAt:    time.Now(),
    }, nil
}
