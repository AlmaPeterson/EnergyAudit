package terminal

import "fmt"

func (t *Terminal) handleStatuses() {
	for _, s := range t.services {
		fmt.Printf("%v: %v\n", s.Name(), s.Status())
	}
}

func (t *Terminal) handleStatus(serviceName string) {
	index := t.getServiceIndex(serviceName)
	if index == -1 {
		t.logger.Warningf("Service %v not found", serviceName)
		return
	}

	fmt.Println(t.services[index].Status())
}
