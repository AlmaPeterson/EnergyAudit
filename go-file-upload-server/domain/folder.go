package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	OwnerID   string     `json:"ownerId"`
	ParentID  *string    `json:"parentId,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

func NewFolder(name, ownerID string, parentID *string) (*Folder, error) {
	if name == "" {
		return nil, errors.New("folder name cannot be empty")
	}
	if ownerID == "" {
		return nil, errors.New("owner ID cannot be empty")
	}

	now := time.Now()
	return &Folder{
		Id:        uuid.NewString(),
		Name:      name,
		OwnerID:   ownerID,
		ParentID:  parentID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
