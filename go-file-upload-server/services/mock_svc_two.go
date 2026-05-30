package services

import (
	"go-file-upload-server/logging"
	"go-file-upload-server/service"
	"reflect"
	"time"
)

type MockSvcTwo struct {
	status service.ServiceStatus
	logger logging.Logger
}

func NewMockSvcTwo() MockSvcTwo {
	svc := MockSvcTwo{
		status: service.ServiceStatusStopped,
	}
	svc.logger = logging.NewLogger()
	svc.logger.ServiceName = svc.Name()

	return svc
}

// Name implements service.Service.
func (m *MockSvcTwo) Name() string {
	svc := *m

	return reflect.TypeOf(svc).Name()
}

func (m *MockSvcTwo) canRestart() bool {
	return m.status == service.ServiceStatusRunning
}

// Restart implements service.Service.
func (m *MockSvcTwo) Restart() {
	if !m.canRestart() {
		return
	}

	m.logger.Info("MockSvcTwo is restarting")

	m.status = service.ServiceStatusRestarting

	for {
		if m.status == service.ServiceStatusRunning || m.status == service.ServiceStatusFailed {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	m.logger.Info("MockSvcTwo restarted")
}

func (m *MockSvcTwo) canStart() bool {
	return m.status == service.ServiceStatusStopped || m.status == service.ServiceStatusFailed
}

// Start implements service.Service.
func (m *MockSvcTwo) Start() {
	if !m.canStart() {
		return
	}

	m.logger.Info("MockSvcTwo is starting")

	m.status = service.ServiceStatusStarting

	go m.run()

	for {
		if m.status == service.ServiceStatusRunning || m.status == service.ServiceStatusFailed {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	m.logger.Info("MockSvcTwo started")
}

// Status implements service.Service.
func (m *MockSvcTwo) Status() service.ServiceStatus {
	return m.status
}

func (m *MockSvcTwo) canStop() bool {
	return m.status == service.ServiceStatusRunning
}

// Stop implements service.Service.
func (m *MockSvcTwo) Stop() {
	if !m.canStop() {
		return
	}

	m.logger.Info("MockSvcTwo is stopping")

	m.status = service.ServiceStatusStopping

	for {
		if m.status == service.ServiceStatusStopped || m.status == service.ServiceStatusFailed {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	m.logger.Info("MockSvcTwo stopped")
}

func (m *MockSvcTwo) preRun() {
	m.status = service.ServiceStatusRunning
}

func (m *MockSvcTwo) run() {
	m.preRun()

	for {
		if m.status == service.ServiceStatusStopping || m.status == service.ServiceStatusRestarting {
			break
		}

		time.Sleep(1 * time.Second)
	}

	m.postRun()
}

func (m *MockSvcTwo) postRun() {
	if m.status == service.ServiceStatusRestarting {
		go m.run()
	} else {
		m.status = service.ServiceStatusStopped
	}
}

var _ service.Service = &MockSvcTwo{}
