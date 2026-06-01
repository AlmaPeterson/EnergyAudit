package controllers

import (
    "fmt"
    "io"
    "net/http"
    "strings"

    "go-file-upload-server/domain"
    "go-file-upload-server/services/httpserver"
)

type ImagesController struct {
    repo          domain.JobRepository
    errorHandler  *httpserver.HttpErrorHandler
    authMiddleware func(http.HandlerFunc) http.HandlerFunc
}

func NewImagesController(repo domain.JobRepository, errorHandler *httpserver.HttpErrorHandler, authMiddleware func(http.HandlerFunc) http.HandlerFunc) ImagesController {
    return ImagesController{repo: repo, errorHandler: errorHandler, authMiddleware: authMiddleware}
}

func (c ImagesController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
    h := handler
    if c.authMiddleware != nil {
        h = c.authMiddleware(h)
    }
    return func(w http.ResponseWriter, r *http.Request) {
        h(w, r)
    }
}

func (c ImagesController) Routes() []httpserver.Route {
    return []httpserver.Route{
        {Pattern: "/api/images", Method: http.MethodPost, Handler: c.HandleUploadImage},
        {Pattern: "/api/images", Method: http.MethodGet, Handler: c.HandleListTaskImages},
        {Pattern: "/api/images/", Method: http.MethodGet, Handler: c.HandleGetImage},
    }
}

func (c ImagesController) HandleUploadImage(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        c.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    if err := r.ParseMultipartForm(16 << 20); err != nil {
        c.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    taskID := r.FormValue("taskId")
    if taskID == "" {
        c.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("taskId form field is required"))
        return
    }

    file, header, err := r.FormFile("image")
    if err != nil {
        c.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }
    defer file.Close()

    data, err := io.ReadAll(file)
    if err != nil {
        c.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }

    photoType := r.FormValue("photoType")
    jobIDValue := r.FormValue("jobId")
    var jobID *string
    if jobIDValue != "" {
        jobID = &jobIDValue
    }

    extension := ""
    filenameParts := strings.Split(header.Filename, ".")
    if len(filenameParts) > 1 {
        extension = filenameParts[len(filenameParts)-1]
    }

    upload, err := domain.NewImageUpload(current.Id, &taskID, jobID, header.Filename, extension, photoType, int64(len(data)), header.Header.Get("Content-Type"))
    if err != nil {
        c.errorHandler.HandleError(http.StatusBadRequest, w, err)
        return
    }

    saved, err := c.repo.UploadImage(*upload, data)
    if err != nil {
        c.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusCreated, saved)
}

func (c ImagesController) HandleListTaskImages(w http.ResponseWriter, r *http.Request) {
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
    uploads, err := c.repo.ListImageUploadsByTask(taskID)
    if err != nil {
        c.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }
    writeJSON(w, http.StatusOK, uploads)
}

func (c ImagesController) HandleGetImage(w http.ResponseWriter, r *http.Request) {
    current := httpserver.GetCurrentUser(r)
    if current == nil {
        c.errorHandler.HandleError(http.StatusUnauthorized, w, fmt.Errorf("authentication required"))
        return
    }

    id := strings.TrimPrefix(r.URL.Path, "/api/images/")
    id = strings.TrimSuffix(id, "/")
    if id == "" {
        c.errorHandler.HandleError(http.StatusBadRequest, w, fmt.Errorf("image id is required"))
        return
    }

    upload, err := c.repo.GetImageUploadByID(id)
    if err != nil {
        c.errorHandler.HandleError(http.StatusNotFound, w, err)
        return
    }
    data, err := c.repo.GetImageUploadDataByID(id)
    if err != nil {
        c.errorHandler.HandleError(http.StatusInternalServerError, w, err)
        return
    }

    w.Header().Set("Content-Type", upload.MimeType)
    w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", upload.OriginalName))
    w.WriteHeader(http.StatusOK)
    w.Write(data)
}

var _ httpserver.Controller = ImagesController{}
