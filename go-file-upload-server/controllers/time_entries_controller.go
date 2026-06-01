package controllers

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"

    "go-file-upload-server/domain"
    "go-file-upload-server/services/httpserver"
)

type TimeEntriesController struct {
    repo          domain.JobRepository
    errorHandler  *httpserver.HttpErrorHandler
    authMiddleware func(http.HandlerFunc) http.HandlerFunc
}

func NewTimeEntriesController(repo domain.JobRepository, errorHandler *httpserver.HttpErrorHandler, authMiddleware func(http.HandlerFunc) http.HandlerFunc) TimeEntriesController {
    return TimeEntriesController{repo: repo, errorHandler: errorHandler, authMiddleware: authMiddleware}
}

func (c TimeEntriesController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
    h := handler
    if c.authMiddleware != nil {
        h = c.authMiddleware(h)
    }
    return func(w http.ResponseWriter, r *http.Request) {
        h(w, r)
    }
}

func (c TimeEntriesController) Routes() []httpserver.Route {
    return []httpserver.Route{
        {Pattern: "/api/time-entries", Method: http.MethodPost, Handler: c.HandleCreateTimeEntry},
        {Pattern: "/api/time-entries", Method: http.MethodGet, Handler: c.HandleListTimeEntries},
        {Pattern: "/api/time-entries/", Method: http.MethodPut, Handler: c.HandleStopTimer},
    }
}

type createTimeEntryRequest struct {
    StartTime string `json:"startTime"`
    EndTime   string `json:"endTime,omitempty"`
    Note      string `json:"note,omitempty"`
}

func (c TimeEntriesController) HandleCreateTimeEntry(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        c.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    var req struct {
        TaskID    string `json:"taskId"`
        StartTime string `json:"startTime"`
        EndTime   string `json:"endTime,omitempty"`
        Note      string `json:"note,omitempty"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        c.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }
    if req.TaskID == "" {
        c.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("taskId is required"))
        return
    }

    taskID := req.TaskID

    startTime, err := time.Parse(time.RFC3339, req.StartTime)
    if err != nil {
        c.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("invalid startTime: %w", err))
        return
    }

    var endTime *time.Time
    if req.EndTime != "" {
        parsed, err := time.Parse(time.RFC3339, req.EndTime)
        if err != nil {
            c.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("invalid endTime: %w", err))
            return
        }
        endTime = &parsed
    }

    entry, err := domain.NewTimeEntry(taskID, current.Id, startTime, endTime, req.Note)
    if err != nil {
        c.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    created, err := c.repo.CreateTimeEntry(*entry)
    if err != nil {
        c.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusCreated, created)
}

func (c TimeEntriesController) HandleListTimeEntries(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        c.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    taskID := r.URL.Query().Get("taskId")
    if taskID == "" {
        c.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("taskId query parameter is required"))
        return
    }
    entries, err := c.repo.ListTimeEntriesByTask(taskID)
    if err != nil {
        c.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusOK, entries)
}

func (c TimeEntriesController) HandleStopTimer(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        c.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    path := strings.TrimPrefix(r.URL.Path, "/api/time-entries/")
    path = strings.TrimSuffix(path, "/")
    parts := strings.Split(path, "/")
    if len(parts) != 2 || parts[1] != "stop" {
        c.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("path must be /api/time-entries/{id}/stop"))
        return
    }

    id := parts[0]
    activeEntry, err := c.repo.GetTimeEntryByID(id)
    if err != nil {
        c.errorHandler.HandleError(http.StatusNotFound, w, err)
        return
    }
    if activeEntry.UserID != current.Id {
        c.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("permission denied"))
        return
    }
    if activeEntry.EndTime != nil {
        c.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("time entry is already stopped"))
        return
    }

    now := time.Now()
    activeEntry.EndTime = &now
    activeEntry.DurationMinutes = int(now.Sub(activeEntry.StartTime).Minutes())

    updated, err := c.repo.UpdateTimeEntry(activeEntry)
    if err != nil {
        c.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusOK, updated)
}

var _ httpserver.Controller = TimeEntriesController{}
