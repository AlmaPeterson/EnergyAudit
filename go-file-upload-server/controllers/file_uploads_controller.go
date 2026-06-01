package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"go-file-upload-server/domain"
	"go-file-upload-server/services/httpserver"
)

type FileUploadsController struct {
	repository     domain.FileUploadRepository
	errorHandler   *httpserver.HttpErrorHandler
	authMiddleware func(http.HandlerFunc) http.HandlerFunc
}

func NewFileUploadsController(repository domain.FileUploadRepository, errorHandler *httpserver.HttpErrorHandler, authMiddleware func(http.HandlerFunc) http.HandlerFunc) FileUploadsController {
	return FileUploadsController{
		repository:     repository,
		errorHandler:   errorHandler,
		authMiddleware: authMiddleware,
	}
}

// BeforeAction implements httpserver.Controller.
func (f FileUploadsController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
	h := handler
	if f.authMiddleware != nil {
		h = f.authMiddleware(h)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r)
	}
}

// Routes implements httpserver.Controller.
func (f FileUploadsController) Routes() []httpserver.Route {
	return []httpserver.Route{
		{
			Pattern: "/api/uploads",
			Method:  http.MethodGet,
			Handler: f.HandleListUploads,
		},
		{
			Pattern: "/api/uploads",
			Method:  http.MethodPost,
			Handler: f.HandleUploadFile,
		},
		{
			Pattern: "/api/uploads/",
			Method:  http.MethodDelete,
			Handler: f.HandleDeleteUpload,
		},
		{
			Pattern: "/uploads/",
			Method:  http.MethodGet,
			Handler: f.HandleDownloadUpload,
		},
	}
}

func (f FileUploadsController) HandleListUploads(w http.ResponseWriter, r *http.Request) {
	// require authenticated user
	current := httpserver.GetCurrentUser(r)
	if current == nil {
		f.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	var folderID *string
	if v := r.URL.Query().Get("folderId"); v != "" {
		folderID = &v
	}

	uploads, err := f.repository.ListFileUploadsByFolder(current.Id, folderID)
	if err != nil {
		f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
		return
	}

	writeJSON(w, http.StatusOK, uploads)
}

func (f FileUploadsController) HandleUploadFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		f.errorHandler.HandleError(http.StatusBadRequest, w, err)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		f.errorHandler.HandleError(http.StatusBadRequest, w, err)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
		return
	}

	fileName, fileExtension := splitFileName(header.Filename)
	if fileExtension == "" {
		f.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("file extension is required"))
		return
	}

	// require authenticated user
	current := httpserver.GetCurrentUser(r)
	if current == nil {
		f.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	var folderID *string
	// optionally read folder id from form field "folderId"
	if v := r.FormValue("folderId"); v != "" {
		folderID = &v
	}

	if folderID != nil {
		folder, err := f.repository.GetFolderByID(*folderID)
		if err != nil {
			f.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("invalid folder id"))
			return
		}
		if folder.OwnerID != current.Id {
			f.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("forbidden folder access"))
			return
		}
	}

	fileUpload, err := domain.NewFileUpload(fileName, fileExtension, current.Id, folderID, int64(len(data)), header.Header.Get("Content-Type"))
	if err != nil {
		f.errorHandler.HandleError(http.StatusBadRequest, w, err)
		return
	}

	saved, err := f.repository.UploadFile(*fileUpload, data)
	if err != nil {
		f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
		return
	}

	writeJSON(w, http.StatusCreated, saved)
}

func (f FileUploadsController) HandleDeleteUpload(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/uploads/")
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		f.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("upload ID is required"))
		return
	}

	current := httpserver.GetCurrentUser(r)
	if current == nil {
		f.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	upload, err := f.repository.GetFileUploadByID(id)
	if err != nil {
		f.errorHandler.HandleError(http.StatusNotFound, w, err)
		return
	}

	if upload.OwnerID != current.Id {
		if concrete, ok := f.repository.(interface{ UserHasAccess(string, string, string) (bool, error) }); ok {
			allowed, err := concrete.UserHasAccess(id, current.Id, "delete")
			if err != nil {
				f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
				return
			}
			if !allowed {
				f.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("forbidden"))
				return
			}
		} else {
			f.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("forbidden"))
			return
		}
	}

	if err := f.repository.DeleteFileUpload(id); err != nil {
		f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (f FileUploadsController) HandleDownloadUpload(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/uploads/")
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		f.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("upload ID is required"))
		return
	}

	current := httpserver.GetCurrentUser(r)
	if current == nil {
		f.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	upload, err := f.repository.GetFileUploadByID(id)
	if err != nil {
		f.errorHandler.HandleError(http.StatusNotFound, w, err)
		return
	}

	if upload.OwnerID != current.Id {
		if concrete, ok := f.repository.(interface{ UserHasAccess(string, string, string) (bool, error) }); ok {
			allowed, err := concrete.UserHasAccess(id, current.Id, "read")
			if err != nil {
				f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
				return
			}
			if !allowed {
				f.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("forbidden"))
				return
			}
		} else {
			f.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("forbidden"))
			return
		}
	}

	data, err := f.repository.GetFileUploadDataByID(id)
	if err != nil {
		f.errorHandler.HandleError(http.StatusInternalServerError, w, err)
		return
	}

	fileName := fmt.Sprintf("%s.%s", upload.OriginalName, upload.Extension)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func splitFileName(name string) (string, string) {
	extension := strings.TrimPrefix(filepath.Ext(name), ".")
	baseName := strings.TrimSuffix(name, filepath.Ext(name))
	if baseName == "" {
		baseName = name
	}
	return baseName, extension
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

var _ httpserver.Controller = FileUploadsController{}
