package controllers

import (
    "encoding/json"
    "fmt"
    "net/http"

    "go-file-upload-server/domain"
    "go-file-upload-server/services/httpserver"
)

type EnergyAuditsController struct {
    repo          domain.JobRepository
    errorHandler  *httpserver.HttpErrorHandler
    authMiddleware func(http.HandlerFunc) http.HandlerFunc
}

func NewEnergyAuditsController(repo domain.JobRepository, errorHandler *httpserver.HttpErrorHandler, authMiddleware func(http.HandlerFunc) http.HandlerFunc) EnergyAuditsController {
    return EnergyAuditsController{repo: repo, errorHandler: errorHandler, authMiddleware: authMiddleware}
}

func (c EnergyAuditsController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
    h := handler
    if c.authMiddleware != nil {
        h = c.authMiddleware(h)
    }
    return func(w http.ResponseWriter, r *http.Request) {
        h(w, r)
    }
}

func (c EnergyAuditsController) Routes() []httpserver.Route {
    return []httpserver.Route{
        {Pattern: "/api/audits", Method: http.MethodPost, Handler: c.HandleCreateEnergyAudit},
        {Pattern: "/api/audits", Method: http.MethodGet, Handler: c.HandleListEnergyAudits},
    }
}

type createEnergyAuditRequest struct {
    JobID            string `json:"jobId"`
    TaskID           string `json:"taskId"`
    Easy             bool   `json:"easy"`
    Hard             bool   `json:"hard"`
    Fun              bool   `json:"fun"`
    NotFun           bool   `json:"notFun"`
    EfficiencyRating int    `json:"efficiencyRating"`
    Notes            string `json:"notes,omitempty"`
}

func (c EnergyAuditsController) HandleCreateEnergyAudit(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        c.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    var req createEnergyAuditRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        c.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }
    if req.JobID == "" {
        c.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("jobId is required"))
        return
    }
    if req.TaskID == "" {
        c.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("taskId is required"))
        return
    }

    audit, err := domain.NewEnergyAudit(req.TaskID, req.JobID, current.Id, req.Easy, req.Hard, req.Fun, req.NotFun, req.EfficiencyRating, req.Notes)
    if err != nil {
        c.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    created, err := c.repo.CreateEnergyAudit(*audit)
    if err != nil {
        c.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusCreated, created)
}

func (c EnergyAuditsController) HandleListEnergyAudits(w http.ResponseWriter, r *http.Request) {
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

    audits, err := c.repo.ListEnergyAuditsByTask(taskID)
    if err != nil {
        c.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusOK, audits)
}

var _ httpserver.Controller = EnergyAuditsController{}
