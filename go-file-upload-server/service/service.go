package service

type ServiceStatus string

const (
	ServiceStatusRunning    ServiceStatus = "running"
	ServiceStatusStopped    ServiceStatus = "stopped"
	ServiceStatusStarting   ServiceStatus = "starting"
	ServiceStatusStopping   ServiceStatus = "stopping"
	ServiceStatusRestarting ServiceStatus = "restarting"
	ServiceStatusFailed     ServiceStatus = "failed"
)

type Service interface {
	Start()
	Stop()
	Restart()
	Status() ServiceStatus
	Name() string
}
