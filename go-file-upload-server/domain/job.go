package domain

import (
    "errors"
    "time"

    "github.com/google/uuid"
)

type Job struct {
    Id          string    `json:"id"`
    OwnerID     string    `json:"ownerId"`
    Title       string    `json:"title"`
    Description string    `json:"description,omitempty"`
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
}

func NewJob(title, description, ownerID string) (*Job, error) {
    if title == "" {
        return nil, errors.New("title is required")
    }
    if ownerID == "" {
        return nil, errors.New("owner id is required")
    }

    now := time.Now()
    return &Job{
        Id:          uuid.NewString(),
        OwnerID:     ownerID,
        Title:       title,
        Description: description,
        CreatedAt:   now,
        UpdatedAt:   now,
    }, nil
}
