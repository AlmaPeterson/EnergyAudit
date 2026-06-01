package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go-file-upload-server/domain"
	userrepo "go-file-upload-server/repositories/user"
	"go-file-upload-server/services/httpserver"
)

// SharesController manages creating/listing/deleting file shares.
type SharesController struct {
	repo          domain.FileUploadRepository // we'll use the concrete fileupload Postgres repo which implements share methods
	userRepo      *userrepo.PostgresUserRepository // for looking up users by email
	errorHandler  *httpserver.HttpErrorHandler
	authMiddleware func(http.HandlerFunc) http.HandlerFunc
}

func NewSharesController(repo domain.FileUploadRepository, userRepo *userrepo.PostgresUserRepository, errorHandler *httpserver.HttpErrorHandler, authMiddleware func(http.HandlerFunc) http.HandlerFunc) SharesController {
	return SharesController{repo: repo, userRepo: userRepo, errorHandler: errorHandler, authMiddleware: authMiddleware}
}

func (s SharesController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
	h := handler
	if s.authMiddleware != nil {
		h = s.authMiddleware(h)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		h(w, r)
	}
}

func (s SharesController) Routes() []httpserver.Route {
	return []httpserver.Route{
		{Pattern: "/api/shares", Method: http.MethodPost, Handler: s.HandleCreateShare},
		{Pattern: "/api/shares", Method: http.MethodGet, Handler: s.HandleListShares},
		{Pattern: "/api/shares/", Method: http.MethodDelete, Handler: s.HandleDeleteShare},
	}
}

type createShareRequest struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	GranteeEmail string `json:"granteeEmail"`
	AccessLevel  string `json:"accessLevel"`
}

func (s SharesController) HandleCreateShare(w http.ResponseWriter, r *http.Request) {
	current := httpserver.GetCurrentUser(r)
	if current == nil {
		s.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	var req createShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorHandler.HandleError(http.StatusBadRequest, w, err)
		return
	}
	if req.ResourceType == "" || req.ResourceID == "" || req.GranteeEmail == "" || req.AccessLevel == "" {
		s.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("resourceType, resourceId, granteeEmail and accessLevel are required"))
		return
	}

	if req.ResourceType != "file" && req.ResourceType != "folder" {
		s.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("resourceType must be 'file' or 'folder'"))
		return
	}

	// Look up the grantee user by email
	granteeUser, err := s.userRepo.GetByEmail(req.GranteeEmail)
	if err != nil {
		s.errorHandler.HandleError(http.StatusInternalServerError, w, fmt.Errorf("failed to lookup user"))
		return
	}
	if granteeUser == nil {
		s.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("user with email %s not found", req.GranteeEmail))
		return
	}

	var ownerResourceID string
	if req.ResourceType == "file" {
		file, err := s.repo.GetFileUploadByID(req.ResourceID)
		if err != nil {
			s.errorHandler.HandleError(http.StatusNotFound, w, err)
			return
		}
		if file.OwnerID != current.Id {
			s.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("only owner can share"))
			return
		}
		ownerResourceID = file.Id
	} else {
		folder, err := s.repo.GetFolderByID(req.ResourceID)
		if err != nil {
			s.errorHandler.HandleError(http.StatusNotFound, w, err)
			return
		}
		if folder.OwnerID != current.Id {
			s.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("only owner can share"))
			return
		}
		ownerResourceID = folder.Id
	}

	// Prevent sharing with self
	if granteeUser.Id == current.Id {
		s.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("cannot share resource with yourself"))
		return
	}

	fs, err := domain.NewFileShare(req.ResourceType, ownerResourceID, current.Id, granteeUser.Id, strings.ToLower(req.AccessLevel))
	if err != nil {
		s.errorHandler.HandleError(http.StatusBadRequest, w, err)
		return
	}

	// we expect the concrete repository to implement CreateFileShare via type assertion
	if concrete, ok := s.repo.(interface{ CreateFileShare(domain.FileShare) (domain.FileShare, error) }); ok {
		saved, err := concrete.CreateFileShare(*fs)
		if err != nil {
			s.errorHandler.HandleError(http.StatusInternalServerError, w, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
		return
	}

	s.errorHandler.HandleError(http.StatusInternalServerError, w, fmt.Errorf("repository does not support file shares"))
}

func (s SharesController) HandleListShares(w http.ResponseWriter, r *http.Request) {
	current := httpserver.GetCurrentUser(r)
	if current == nil {
		s.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	// expect concrete repo to implement ListFileSharesForUser
	if concrete, ok := s.repo.(interface{ ListFileSharesForUser(string) ([]domain.FileShare, error) }); ok {
		shares, err := concrete.ListFileSharesForUser(current.Id)
		if err != nil {
			s.errorHandler.HandleError(http.StatusInternalServerError, w, err)
			return
		}
		writeJSON(w, http.StatusOK, shares)
		return
	}

	s.errorHandler.HandleError(http.StatusInternalServerError, w, fmt.Errorf("repository does not support file shares"))
}

func (s SharesController) HandleDeleteShare(w http.ResponseWriter, r *http.Request) {
	current := httpserver.GetCurrentUser(r)
	if current == nil {
		s.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/shares/")
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		s.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("share ID is required"))
		return
	}

	// get share to confirm owner
	if concreteGet, ok := s.repo.(interface{ GetFileShareByID(string) (domain.FileShare, error) }); ok {
		fs, err := concreteGet.GetFileShareByID(id)
		if err != nil {
			s.errorHandler.HandleError(http.StatusNotFound, w, err)
			return
		}
		if fs.OwnerID != current.Id {
			s.errorHandler.HandleError(http.StatusForbidden, w, fmt.Errorf("only owner can delete share"))
			return
		}
		if concreteDel, ok := s.repo.(interface{ DeleteFileShare(string) error }); ok {
			if err := concreteDel.DeleteFileShare(id); err != nil {
				s.errorHandler.HandleError(http.StatusInternalServerError, w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
			s.errorHandler.HandleError(http.StatusInternalServerError, w, fmt.Errorf("repository does not support file shares"))
			return
	}

	s.errorHandler.HandleError(http.StatusInternalServerError, w, fmt.Errorf("repository does not support file shares"))
}

var _ httpserver.Controller = SharesController{}
