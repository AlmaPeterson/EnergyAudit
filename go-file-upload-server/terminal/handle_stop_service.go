package terminal

func (t *Terminal) handleStopService(serviceName string) {
	index := t.getServiceIndex(serviceName)
	if index == -1 {
		t.logger.Warningf("Service %v not found", serviceName)
		return
	}

	t.services[index].Stop()
}
