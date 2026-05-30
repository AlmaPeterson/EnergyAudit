package controllers

import (
	"go-file-upload-server/services/httpserver"
	"net/http"
)

type HTMLController struct {
	errorHandler *httpserver.HttpErrorHandler
}

func NewHTMLController() HTMLController {
	return HTMLController{}
}

// BeforeAction implements httpserver.Controller.
func (h HTMLController) BeforeAction(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}
}

// Routes implements httpserver.Controller.
func (h HTMLController) Routes() []httpserver.Route {
	return []httpserver.Route{
		{
			Pattern: "/",
			Method:  http.MethodGet,
			Handler: h.HandleServeHTML,
		},
	}
}

func (h HTMLController) HandleServeHTML(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.Dir("html")).ServeHTTP(w, r)
}

var _ httpserver.Controller = HTMLController{}
