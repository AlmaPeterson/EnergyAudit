package controllers

import (
	"errors"
	"go-file-upload-server/services/httpserver"
)

func SetupControllers(errorHandler *httpserver.HttpErrorHandler) (httpserver.Controller, []httpserver.Controller, error) {
	if errorHandler == nil {
		return nil, nil, errors.New("error handler is required")
	}

	applicationController, err := NewApplicationController(errorHandler)
	if err != nil {
		return nil, nil, err
	}

	htmlController := NewHTMLController()

	return applicationController, []httpserver.Controller{htmlController}, nil
}
