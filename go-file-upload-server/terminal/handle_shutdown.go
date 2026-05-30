package terminal

import (
	"sync"
	"time"
)

func (t *Terminal) handleShutdown() {
	t.logger.Info("Shutting down terminal, stopping services")

	wg := sync.WaitGroup{}

	for i := range t.services {
		wg.Add(1)
		go func() {
			t.services[i].Stop()
			wg.Done()
		}()
	}

	wg.Wait()

	t.shutdown = true

	go func() {
		time.Sleep(1 * time.Second)
		t.forceQuit()
	}()
}
