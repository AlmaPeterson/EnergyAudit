package controllers

import (
	"errors"
	"go-file-upload-server/services/httpserver"
	"net/http"
)

type ApplicationController struct {
	errorHandler *httpserver.HttpErrorHandler
}

func NewApplicationController(errorHandler *httpserver.HttpErrorHandler) (ApplicationController, error) {
	if errorHandler == nil {
		return ApplicationController{}, errors.New("error handler is required")
	}

	return ApplicationController{
		errorHandler: errorHandler,
	}, nil
}

// BeforeAction implements httpserver.Controller.
func (a ApplicationController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}
}

// Routes implements httpserver.Controller.
func (a ApplicationController) Routes() []httpserver.Route {
	return []httpserver.Route{}
}

var _ httpserver.Controller = ApplicationController{}
