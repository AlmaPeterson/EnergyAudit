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

type JobsController struct {
    repo          domain.JobRepository
    errorHandler  *httpserver.HttpErrorHandler
    authMiddleware func(http.HandlerFunc) http.HandlerFunc
}

func NewJobsController(repo domain.JobRepository, errorHandler *httpserver.HttpErrorHandler, authMiddleware func(http.HandlerFunc) http.HandlerFunc) JobsController {
    return JobsController{repo: repo, errorHandler: errorHandler, authMiddleware: authMiddleware}
}

func (j JobsController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
    h := handler
    if j.authMiddleware != nil {
        h = j.authMiddleware(h)
    }
    return func(w http.ResponseWriter, r *http.Request) {
        h(w, r)
    }
}

func (j JobsController) Routes() []httpserver.Route {
    return []httpserver.Route{
        {Pattern: "/api/jobs", Method: http.MethodGet, Handler: j.HandleListJobs},
        {Pattern: "/api/jobs", Method: http.MethodPost, Handler: j.HandleCreateJob},
        {Pattern: "/api/jobs/", Method: http.MethodGet, Handler: j.HandleGetJob},
        {Pattern: "/api/jobs/", Method: http.MethodPut, Handler: j.HandleUpdateJob},
    }
}

type createJobRequest struct {
    Title       string `json:"title"`
    Description string `json:"description"`
}

func (j JobsController) HandleListJobs(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        j.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    jobs, err := j.repo.ListJobs()
    if err != nil {
        j.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusOK, jobs)
}

func (j JobsController) HandleCreateJob(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        j.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    var req createJobRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        j.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    job, err := domain.NewJob(req.Title, req.Description, current.Id)
    if err != nil {
        j.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    saved, err := j.repo.CreateJob(*job)
    if err != nil {
        j.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusCreated, saved)
}

func (j JobsController) HandleGetJob(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        j.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    id := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
    id = strings.TrimSuffix(id, "/")
    if id == "" {
        j.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("job id is required"))
        return
    }

    job, err := j.repo.GetJobByID(id)
    if err != nil {
        j.errorHandler.HandleError(http.StatusNotFound, w, err)
        return
    }
    writeJSON(w, http.StatusOK, job)
}

type updateJobRequest struct {
    Title       string `json:"title,omitempty"`
    Description string `json:"description,omitempty"`
}

func (j JobsController) HandleUpdateJob(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        j.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    id := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
    id = strings.TrimSuffix(id, "/")
    if id == "" {
        j.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("job id is required"))
        return
    }

    existing, err := j.repo.GetJobByID(id)
    if err != nil {
        j.errorHandler.HandleError(http.StatusNotFound, w, err)
        return
    }
    if existing.OwnerID != current.Id {
        j.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("only owner can update job"))
        return
    }

    var req updateJobRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        j.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    if req.Title != "" {
        existing.Title = req.Title
    }
    if req.Description != "" {
        existing.Description = req.Description
    }
    existing.UpdatedAt = time.Now()

    updated, err := j.repo.UpdateJob(existing)
    if err != nil {
        j.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusOK, updated)
}

var _ httpserver.Controller = JobsController{}
