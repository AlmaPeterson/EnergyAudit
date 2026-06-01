package domain

import (
    "errors"
    "time"

    "github.com/google/uuid"
)

type TimeEntry struct {
    Id              string     `json:"id"`
    TaskID          string     `json:"taskId"`
    UserID          string     `json:"userId"`
    StartTime       time.Time  `json:"startTime"`
    EndTime         *time.Time `json:"endTime,omitempty"`
    DurationMinutes int        `json:"durationMinutes"`
    Note            string     `json:"note,omitempty"`
    CreatedAt       time.Time  `json:"createdAt"`
}

func NewTimeEntry(taskID, userID string, startTime time.Time, endTime *time.Time, note string) (*TimeEntry, error) {
    if taskID == "" {
        return nil, errors.New("task id is required")
    }
    if userID == "" {
        return nil, errors.New("user id is required")
    }
    if startTime.IsZero() {
        return nil, errors.New("start time is required")
    }

    duration := 0
    if endTime != nil {
        if endTime.Before(startTime) {
            return nil, errors.New("end time cannot be before start time")
        }
        duration = int(endTime.Sub(startTime).Minutes())
    }

    now := time.Now()
    return &TimeEntry{
        Id:              uuid.NewString(),
        TaskID:          taskID,
        UserID:          userID,
        StartTime:       startTime,
        EndTime:         endTime,
        DurationMinutes: duration,
        Note:            note,
        CreatedAt:       now,
    }, nil
}
