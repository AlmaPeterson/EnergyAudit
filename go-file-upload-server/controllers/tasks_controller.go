package controllers

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "go-file-upload-server/domain"
    "go-file-upload-server/services/httpserver"
)

type TasksController struct {
    repo          domain.JobRepository
    errorHandler  *httpserver.HttpErrorHandler
    authMiddleware func(http.HandlerFunc) http.HandlerFunc
}

func NewTasksController(repo domain.JobRepository, errorHandler *httpserver.HttpErrorHandler, authMiddleware func(http.HandlerFunc) http.HandlerFunc) TasksController {
    return TasksController{repo: repo, errorHandler: errorHandler, authMiddleware: authMiddleware}
}

func (t TasksController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
    h := handler
    if t.authMiddleware != nil {
        h = t.authMiddleware(h)
    }
    return func(w http.ResponseWriter, r *http.Request) {
        h(w, r)
    }
}

func (t TasksController) Routes() []httpserver.Route {
    return []httpserver.Route{
        {Pattern: "/api/tasks", Method: http.MethodPost, Handler: t.HandleCreateTask},
        {Pattern: "/api/tasks", Method: http.MethodGet, Handler: t.HandleListTasksByJob},
        {Pattern: "/api/tasks/", Method: http.MethodGet, Handler: t.HandleGetTask},
        {Pattern: "/api/tasks/", Method: http.MethodPut, Handler: t.HandleUpdateTask},
    }
}

type createTaskRequest struct {
    Title       string `json:"title"`
    Description string `json:"description"`
}

func (t TasksController) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        t.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    var req struct {
        JobID       string `json:"jobId"`
        Title       string `json:"title"`
        Description string `json:"description"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        t.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }
    if req.JobID == "" {
        t.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("jobId is required"))
        return
    }

    _, err := t.repo.GetJobByID(req.JobID)
    if err != nil {
        t.errorHandler.HandleError(http.StatusNotFound, w, err)
        return
    }

    task, err := domain.NewTask(req.JobID, req.Title, req.Description, current.Id)
    if err != nil {
        t.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    created, err := t.repo.CreateTask(*task)
    if err != nil {
        t.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusCreated, created)
}

func (t TasksController) HandleListTasksByJob(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        t.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    jobID := r.URL.Query().Get("jobId")
    if jobID == "" {
        t.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("jobId query parameter is required"))
        return
    }

    tasks, err := t.repo.ListTasksByJob(jobID)
    if err != nil {
        t.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusOK, tasks)
}

type updateTaskRequest struct {
    Title       string `json:"title,omitempty"`
    Description string `json:"description,omitempty"`
    Status      string `json:"status,omitempty"`
}

func (t TasksController) HandleGetTask(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        t.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    id := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
    id = strings.TrimSuffix(id, "/")
    if id == "" {
        t.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("task id is required"))
        return
    }

    task, err := t.repo.GetTaskByID(id)
    if err != nil {
        t.errorHandler.HandleError(http.StatusNotFound, w, err)
        return
    }
    writeJSON(w, http.StatusOK, task)
}

func (t TasksController) HandleUpdateTask(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        t.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    id := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
    id = strings.TrimSuffix(id, "/")
    if id == "" {
        t.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("task id is required"))
        return
    }

    existing, err := t.repo.GetTaskByID(id)
    if err != nil {
        t.errorHandler.HandleError(http.StatusNotFound, w, err)
        return
    }
    if existing.OwnerID != current.Id {
        t.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("only owner can update task"))
        return
    }

    var req updateTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        t.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    if req.Title != "" {
        existing.Title = req.Title
    }
    if req.Description != "" {
        existing.Description = req.Description
    }
    if req.Status != "" {
        existing.Status = req.Status
    }

    updated, err := t.repo.UpdateTask(existing)
    if err != nil {
        t.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusOK, updated)
}

var _ httpserver.Controller = TasksController{}
