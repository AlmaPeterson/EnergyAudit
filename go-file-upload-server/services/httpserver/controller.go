package httpserver

import (
	"net/http"
)

type Controller interface {
	BeforeAction(handler http.HandlerFunc) http.HandlerFunc
	Routes() []Route
}
