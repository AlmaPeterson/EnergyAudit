package httpserver

import (
	"context"
	"errors"
	"fmt"
	"go-file-upload-server/logging"
	"go-file-upload-server/service"
	"net"
	"net/http"
	"time"
)

type HttpServer struct {
	config Config
	routes []Route
	server *http.Server
	logger *logging.Logger
	status service.ServiceStatus
	mux    http.ServeMux
}

func NewHttpServer(config Config, routes []Route, logger *logging.Logger) (*HttpServer, error) {
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	err := config.Verify()
	if err != nil {
		return nil, err
	}

	if config.Name == "" {
		return nil, errors.New("name is required")
	}

	srv := HttpServer{
		config: config,
		routes: routes,
		status: service.ServiceStatusStopped,
		logger: logger,
	}

	srv.logger.ServiceName = config.Name

	return &srv, nil
}

func (h *HttpServer) setStatus(status service.ServiceStatus) {
	h.logger.Infof("%v status changed to %v", h.Name(), status)
	h.status = status
}

// Name implements service.Service.
func (h *HttpServer) Name() string {
	return h.config.Name
}

func (h *HttpServer) canRestart() bool {
	return h.status == service.ServiceStatusRunning
}

func (h *HttpServer) canStart() bool {
	return h.status == service.ServiceStatusStopped || h.status == service.ServiceStatusFailed
}

func (h *HttpServer) canStop() bool {
	return h.status == service.ServiceStatusRunning
}

// Restart implements service.Service.
func (h *HttpServer) Restart() {
	if !h.canRestart() {
		h.logger.Warningf("%v has stopped or failed", h.Name())
		return
	}

	h.setStatus(service.ServiceStatusRestarting)

	for {
		if h.status == service.ServiceStatusRunning || h.status == service.ServiceStatusFailed {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	h.logger.Infof("%v %v", h.Name(), h.status)
}

// Start implements service.Service.
func (h *HttpServer) Start() {
	if !h.canStart() {
		h.logger.Warningf("%v is already running", h.Name())
		return
	}

	h.setStatus(service.ServiceStatusStarting)

	go h.run()

	for {
		if h.status == service.ServiceStatusRunning || h.status == service.ServiceStatusFailed {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	h.logger.Infof("%v %v", h.Name(), h.status)
}

// Status implements service.Service.
func (h *HttpServer) Status() service.ServiceStatus {
	return h.status
}

// Stop implements service.Service.
func (h *HttpServer) Stop() {
	if !h.canStop() {
		h.logger.Warningf("%v has already stopped or failed", h.Name())
		return
	}

	h.setStatus(service.ServiceStatusStopping)

	for {
		if h.status == service.ServiceStatusStopped || h.status == service.ServiceStatusFailed {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	h.logger.Infof("%v %v", h.Name(), h.status)
}

func (h *HttpServer) run() {
	h.preRun()

	for {
		if h.status == service.ServiceStatusFailed {
			return
		}

		if h.status == service.ServiceStatusStopping || h.status == service.ServiceStatusRestarting {
			break
		}

		time.Sleep(1 * time.Second)
	}

	h.postRun()
}

func (h *HttpServer) preRun() {
	h.logger.Debug("Building server mux")
	h.mux = *http.NewServeMux()
	h.logger.Debug("Registering routes")
	h.RegisterRoutes()
	h.logger.Infof("Registered %v routes", len(h.routes))
	h.logger.Debug("Building server")
	h.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", h.config.Port),
		Handler: &h.mux,
	}

	h.logger.Debug("Starting listener")
	l, err := net.Listen("tcp", h.server.Addr)
	if err != nil {
		h.logger.Errorf("Failed to listen on %v: %v", h.server.Addr, err)
		h.setStatus(service.ServiceStatusFailed)
		return
	}

	h.setStatus(service.ServiceStatusRunning)

	go func() {
		var err error
		if h.config.SSLEnabled() {
			h.logger.Info("Serving over HTTPS")
			err = h.server.ServeTLS(l, h.config.SSL.CertFile, h.config.SSL.KeyFile)
		} else {
			h.logger.Info("Serving over HTTP")
			err = h.server.Serve(l)
		}

		if err != nil {
			if err != http.ErrServerClosed {
				h.logger.Errorf("Failed to serve: %v", err)
				h.setStatus(service.ServiceStatusFailed)
			} else {
				h.logger.Infof("Server closed: %v", err)
			}
		} else {
			h.logger.Infof("Server closed: %v", err)
		}
	}()

	h.logger.Infof("Listening on %v", h.server.Addr)
}

func (h *HttpServer) postRun() {
	context, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	err := h.server.Shutdown(context)
	if err != nil {
		h.logger.Errorf("There was an error while shutting down the server: %v", err)
		h.setStatus(service.ServiceStatusFailed)
		cancel()
		return
	}

	cancel()

	if h.status == service.ServiceStatusRestarting {
		go h.run()
	} else {
		h.setStatus(service.ServiceStatusStopped)
		h.logger.Infof("Server shutdown")
	}
}

var _ service.Service = &HttpServer{}
