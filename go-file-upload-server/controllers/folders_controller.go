package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go-file-upload-server/domain"
	"go-file-upload-server/services/httpserver"
)

type FoldersController struct {
	repo          domain.FileUploadRepository
	errorHandler *httpserver.HttpErrorHandler
	authMiddleware func(http.HandlerFunc) http.HandlerFunc
}

func NewFoldersController(repo domain.FileUploadRepository, errorHandler *httpserver.HttpErrorHandler, authMiddleware func(http.HandlerFunc) http.HandlerFunc) FoldersController {
	return FoldersController{
		repo:           repo,
		errorHandler:   errorHandler,
		authMiddleware: authMiddleware,
	}
}

func (f FoldersController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
	h := handler
	if f.authMiddleware != nil {
		h = f.authMiddleware(h)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r)
	}
}

func (f FoldersController) Routes() []httpserver.Route {
	return []httpserver.Route{
		{Pattern: "/api/folders", Method: http.MethodGet, Handler: f.HandleListFolders},
		{Pattern: "/api/folders", Method: http.MethodPost, Handler: f.HandleCreateFolder},
		{Pattern: "/api/folders/", Method: http.MethodGet, Handler: f.HandleFolderDetailRequest},
		{Pattern: "/api/folders/", Method: http.MethodDelete, Handler: f.HandleDeleteFolder},
	}
}

type createFolderRequest struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parentId,omitempty"`
}

func (f FoldersController) HandleListFolders(w http.ResponseWriter, r *http.Request) {
	current := httpserver.GetCurrentUser(r)
	if current == nil {
		f.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	var parentID *string
	if v := r.URL.Query().Get("parentId"); v != "" {
		parentID = &v
	}

	folders, err := f.repo.ListFoldersForUser(current.Id, parentID)
	if err != nil {
		f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
		return
	}

	writeJSON(w, http.StatusOK, folders)
}

func (f FoldersController) HandleFolderDetailRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/folders/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		f.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("folder ID is required"))
		return
	}

	if strings.HasSuffix(r.URL.Path, "/contents") {
		f.HandleGetFolderContents(w, r)
		return
	}

	f.HandleGetFolder(w, r)
}

func (f FoldersController) HandleGetFolder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/folders/")
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		f.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("folder ID is required"))
		return
	}

	current := httpserver.GetCurrentUser(r)
	if current == nil {
		f.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	folder, err := f.repo.GetFolderByID(id)
	if err != nil {
		f.errorHandler.HandleError(http.StatusNotFound, w, err)
		return
	}

	if folder.OwnerID != current.Id {
		allowed, err := f.repo.UserHasFolderAccess(id, current.Id)
		if err != nil {
			f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
			return
		}
		if !allowed {
			f.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("forbidden"))
			return
		}
	}

	writeJSON(w, http.StatusOK, folder)
}

func (f FoldersController) HandleGetFolderContents(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/folders/")
	trimmed = strings.TrimSuffix(trimmed, "/contents")
	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed == "" {
		f.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("folder ID is required"))
		return
	}

	current := httpserver.GetCurrentUser(r)
	if current == nil {
		f.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	folders, uploads, err := f.repo.ListFolderContents(current.Id, trimmed)
	if err != nil {
		if strings.Contains(err.Error(), "forbidden") {
			f.errorHandler.HandleError(http.StatusForbidden, w, err)
			return
		}
		f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"folders": folders,
		"uploads": uploads,
	})
}

func (f FoldersController) HandleCreateFolder(w http.ResponseWriter, r *http.Request) {
	current := httpserver.GetCurrentUser(r)
	if current == nil {
		f.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	var req createFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.errorHandler.HandleError(http.StatusBadRequest, w, err)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		f.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("folder name is required"))
		return
	}

	if req.ParentID != nil {
		parent, err := f.repo.GetFolderByID(*req.ParentID)
		if err != nil {
			f.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("invalid parent folder"))
			return
		}
		if parent.OwnerID != current.Id {
			f.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("forbidden parent folder"))
			return
		}
	}

	folder, err := domain.NewFolder(strings.TrimSpace(req.Name), current.Id, req.ParentID)
	if err != nil {
		f.errorHandler.HandleError(http.StatusBadRequest, w, err)
		return
	}

	saved, err := f.repo.CreateFolder(*folder)
	if err != nil {
		f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
		return
	}

	writeJSON(w, http.StatusCreated, saved)
}

func (f FoldersController) HandleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/folders/")
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		f.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("folder ID is required"))
		return
	}

	current := httpserver.GetCurrentUser(r)
	if current == nil {
		f.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	folder, err := f.repo.GetFolderByID(id)
	if err != nil {
		f.errorHandler.HandleError(http.StatusNotFound, w, err)
		return
	}

	if folder.OwnerID != current.Id {
		f.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("forbidden"))
		return
	}

	if err := f.repo.DeleteFolder(id); err != nil {
		f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

var _ httpserver.Controller = FoldersController{}
