package domain

import (
    "errors"
    "time"

    "github.com/google/uuid"
)

type EnergyAudit struct {
    Id               string    `json:"id"`
    TaskID           string    `json:"taskId"`
    JobID            string    `json:"jobId"`
    UserID           string    `json:"userId"`
    Easy             bool      `json:"easy"`
    Hard             bool      `json:"hard"`
    Fun              bool      `json:"fun"`
    NotFun           bool      `json:"notFun"`
    EfficiencyRating int       `json:"efficiencyRating"`
    Notes            string    `json:"notes,omitempty"`
    CreatedAt        time.Time `json:"createdAt"`
}

func NewEnergyAudit(taskID, jobID, userID string, easy, hard, fun, notFun bool, efficiencyRating int, notes string) (*EnergyAudit, error) {
    if taskID == "" {
        return nil, errors.New("task id is required")
    }
    if jobID == "" {
        return nil, errors.New("job id is required")
    }
    if userID == "" {
        return nil, errors.New("user id is required")
    }
    if efficiencyRating < 1 || efficiencyRating > 10 {
        return nil, errors.New("efficiency rating must be between 1 and 10")
    }

    now := time.Now()
    return &EnergyAudit{
        Id:               uuid.NewString(),
        TaskID:           taskID,
        JobID:            jobID,
        UserID:           userID,
        Easy:             easy,
        Hard:             hard,
        Fun:              fun,
        NotFun:           notFun,
        EfficiencyRating: efficiencyRating,
        Notes:            notes,
        CreatedAt:        now,
    }, nil
}
