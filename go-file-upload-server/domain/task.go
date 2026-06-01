package domain

import (
    "errors"
    "time"

    "github.com/google/uuid"
)

type Task struct {
    Id          string    `json:"id"`
    JobID       string    `json:"jobId"`
    OwnerID     string    `json:"ownerId"`
    Title       string    `json:"title"`
    Description string    `json:"description,omitempty"`
    Status      string    `json:"status"`
    CreatedAt   time.Time `json:"createdAt"`
    UpdatedAt   time.Time `json:"updatedAt"`
}

func NewTask(jobID, title, description, ownerID string) (*Task, error) {
    if jobID == "" {
        return nil, errors.New("job id is required")
    }
    if title == "" {
        return nil, errors.New("title is required")
    }
    if ownerID == "" {
        return nil, errors.New("owner id is required")
    }

    now := time.Now()
    return &Task{
        Id:          uuid.NewString(),
        JobID:       jobID,
        OwnerID:     ownerID,
        Title:       title,
        Description: description,
        Status:      "open",
        CreatedAt:   now,
        UpdatedAt:   now,
    }, nil
}
