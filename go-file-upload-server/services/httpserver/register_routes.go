package httpserver

import (
	"net/http"
	"strings"
)

// RegisterRoutes registers handlers grouped by path. Multiple methods on the same
// path are dispatched by a single mux handler which checks the request method.
func (h *HttpServer) RegisterRoutes() {
	routesByPattern := map[string]map[string]Route{}

	for _, route := range h.routes {
		if _, ok := routesByPattern[route.Pattern]; !ok {
			routesByPattern[route.Pattern] = map[string]Route{}
		}
		// last registration wins for same method+pattern
		routesByPattern[route.Pattern][route.Method] = route
	}

	for pattern, methodMap := range routesByPattern {
		mm := methodMap // capture
		h.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				handleOptions(w, r)
				return
			}

			if rt, ok := mm[r.Method]; ok {
				h.handleRequest(rt)(w, r)
				return
			}

			// method not allowed
			allowed := []string{}
			for m := range mm {
				allowed = append(allowed, m)
			}
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		})
	}
}
