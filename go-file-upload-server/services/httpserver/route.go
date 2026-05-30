package httpserver

import "net/http"

type Route struct {
	Pattern string
	Method  string
	Handler http.HandlerFunc
}
