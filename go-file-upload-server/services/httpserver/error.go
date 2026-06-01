package httpserver

import (
	"encoding/json"
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
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)
	_ = json.NewEncoder(rw).Encode(map[string]string{"error": err.Error()})
}
