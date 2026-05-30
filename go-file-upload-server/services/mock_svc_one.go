package services

import (
	"go-file-upload-server/logging"
	"go-file-upload-server/service"
	"reflect"
	"time"
)

type MockSvcOne struct {
	status service.ServiceStatus
	logger logging.Logger
}

func NewMockSvcOne() MockSvcOne {
	svc := MockSvcOne{
		status: service.ServiceStatusStopped,
	}
	svc.logger = logging.NewLogger()
	svc.logger.ServiceName = svc.Name()

	return svc
}

// Name implements service.Service.
func (m *MockSvcOne) Name() string {
	svc := *m

	return reflect.TypeOf(svc).Name()
}

func (m *MockSvcOne) canRestart() bool {
	return m.status == service.ServiceStatusRunning
}

// Restart implements service.Service.
func (m *MockSvcOne) Restart() {
	if !m.canRestart() {
		return
	}

	m.logger.Info("MockSvcOne is restarting")

	m.status = service.ServiceStatusRestarting

	for {
		if m.status == service.ServiceStatusRunning || m.status == service.ServiceStatusFailed {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	m.logger.Info("MockSvcOne restarted")
}

func (m *MockSvcOne) canStart() bool {
	return m.status == service.ServiceStatusStopped || m.status == service.ServiceStatusFailed
}

// Start implements service.Service.
func (m *MockSvcOne) Start() {
	if !m.canStart() {
		return
	}

	m.logger.Info("MockSvcOne is starting")

	m.status = service.ServiceStatusStarting

	go m.run()

	for {
		if m.status == service.ServiceStatusRunning || m.status == service.ServiceStatusFailed {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	m.logger.Info("MockSvcOne started")
}

// Status implements service.Service.
func (m *MockSvcOne) Status() service.ServiceStatus {
	return m.status
}

func (m *MockSvcOne) canStop() bool {
	return m.status == service.ServiceStatusRunning
}

// Stop implements service.Service.
func (m *MockSvcOne) Stop() {
	if !m.canStop() {
		return
	}

	m.logger.Info("MockSvcOne is stopping")

	m.status = service.ServiceStatusStopping

	for {
		if m.status == service.ServiceStatusStopped || m.status == service.ServiceStatusFailed {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	m.logger.Info("MockSvcOne stopped")
}

func (m *MockSvcOne) preRun() {
	m.status = service.ServiceStatusRunning
}

func (m *MockSvcOne) run() {
	m.preRun()

	for {
		if m.status == service.ServiceStatusStopping || m.status == service.ServiceStatusRestarting {
			break
		}

		time.Sleep(1 * time.Second)
	}

	m.postRun()
}

func (m *MockSvcOne) postRun() {
	if m.status == service.ServiceStatusRestarting {
		go m.run()
	} else {
		m.status = service.ServiceStatusStopped
	}
}

var _ service.Service = &MockSvcOne{}
