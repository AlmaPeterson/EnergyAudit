package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type paramsKeyType string

const paramsKey paramsKeyType = "params"

func handleOptions(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Methods", "*")
	rw.Header().Set("Access-Control-Allow-Headers", "*")
	rw.WriteHeader(http.StatusOK)
}

func (h *HttpServer) handleRequest(route Route) http.HandlerFunc {
	if route.Method == http.MethodOptions {
		return handleOptions
	}

	return func(rw http.ResponseWriter, req *http.Request) {
		if route.Method != req.Method {
			rw.Header().Set("Allow", route.Method)
			h.logger.Warningf("Method not allowed: %v %v", req.Method, req.URL.Path)
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		requestId := uuid.NewString()
		req.Header.Set("X-Request-ID", requestId)

		remoteAddr := req.Header.Get("X-Forwarded-For")
		if remoteAddr == "" {
			remoteAddr = req.Header.Get("X-Real-IP")
		}
		if remoteAddr == "" {
			remoteAddr = req.RemoteAddr
		}
		h.logger.Infof("Started %v %v for %v", req.Method, req.URL.Path, remoteAddr)

		params, err := h.getParams(req)
		if err == nil {
			h.logger.Debugf("Params: %+v", params)
		}
		if params == nil {
			params = map[string]any{}
		}

		reqWithParams := req.WithContext(context.WithValue(req.Context(), paramsKey, params))
		route.Handler(rw, reqWithParams)

		h.logger.Infof("Completed %v %v for %v", req.Method, req.URL.Path, remoteAddr)
	}
}

func (h *HttpServer) getParams(req *http.Request, maxSize ...int64) (map[string]any, error) {
	size := int64(100)
	if len(maxSize) > 0 {
		size = maxSize[0]
	}

	if req.Method == http.MethodGet {
		queryValues := req.URL.Query()
		params := map[string]any{}
		for key, values := range queryValues {
			params[key] = values[0]
		}
		return params, nil
	}

	bodyBytes, err := readRequestBody(req)
	if err != nil {
		return nil, err
	}

	// If the request is multipart/form-data prefer parsing the form first.
	// We do not restore req.Body after this because multipart parsing stores the form data
	// and the handler can use r.FormFile / r.ParseMultipartForm without re-reading the body.
	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/") || strings.Contains(contentType, "multipart/form-data") {
		params, err := decodeFormDataToMap(req, size)
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	// Try to decode JSON body first, fall back to form data (urlencoded or multipart)
	params, err := decodeJSONBodyToMap(bodyBytes)
	if err != nil {
		params, err = decodeFormDataToMap(req, size)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return params, nil
	}

	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return params, nil
}

func readRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return []byte{}, nil
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return bodyBytes, nil
}

func decodeJSONBodyToMap(body []byte) (map[string]any, error) {
	if len(body) == 0 {
		return map[string]any{}, nil
	}

	var decoded map[string]any
	err := json.Unmarshal(body, &decoded)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func decodeFormDataToMap(req *http.Request, maxSize ...int64) (map[string]any, error) {
	size := int64(100)
	if len(maxSize) > 0 {
		size = maxSize[0]
	}

	err := req.ParseMultipartForm(size << 20)
	if err != nil {
		return nil, err
	}

	decoded := map[string]any{}
	for key, value := range req.MultipartForm.Value {
		decoded[key] = value[0]
	}

	for key, value := range req.MultipartForm.File {
		decoded[key] = value[0]
	}

	return decoded, nil
}
