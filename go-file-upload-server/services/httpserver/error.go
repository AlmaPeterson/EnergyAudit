package httpserver

import (
	"errors"
	"go-file-upload-server/logging"
	"net/http"
)

type HttpErrorHandler struct {
	logger *logging.Logger
}

func NewHttpErrorHandler(logger *logging.Logger) (HttpErrorHandler, error) {
	if logger == nil {
		return HttpErrorHandler{}, errors.New("logger is required")
	}

	return HttpErrorHandler{
		logger: logger,
	}, nil
}

func (h *HttpErrorHandler) HandleError(code int, rw http.ResponseWriter, err error) {
	h.logger.Errorf("Error %v: %v", code, err)
	http.Error(rw, err.Error(), code)
}
