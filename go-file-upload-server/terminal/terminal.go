package terminal

import (
	"bufio"
	"fmt"
	"go-file-upload-server/logging"
	"go-file-upload-server/service"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"
)

type Terminal struct {
	services []service.Service
	shutdown bool
	logger   logging.Logger
}

func NewTerminal(services []service.Service) *Terminal {
	logger := logging.NewLogger()
	logger.ServiceName = "Terminal"

	return &Terminal{
		services: services,
		shutdown: false,
		logger:   logger,
	}
}

func (t *Terminal) Start() {
	t.checkServices()
	t.startServices()
	t.catchSignals()
	t.startTerminal()
}

func (t *Terminal) checkServices() {
	names := []string{}
	for _, s := range t.services {
		if slices.Contains(names, s.Name()) {
			t.logger.Errorf("Found more than one service with the same name: %v", s.Name())
			os.Exit(1)
		}

		names = append(names, s.Name())
	}
}

func (t *Terminal) startServices() {
	wg := sync.WaitGroup{}

	for _, s := range t.services {
		wg.Add(1)
		go func() {
			s.Start()
			wg.Done()
		}()
	}

	wg.Wait()
}

func (t *Terminal) catchSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGSEGV, syscall.SIGABRT, syscall.SIGTERM)

	go func() {
		<-signals
		t.logger.Info("Received shutdown signal, shutting down")
		t.handleShutdown()
	}()
}

func (t *Terminal) startTerminal() {
	reader := bufio.NewReader(os.Stdin)

	var terminalError error

	for {
		if t.shutdown {
			break
		}

		fmt.Print("> ")
		var command string
		command, terminalError = reader.ReadString('\n')
		if terminalError != nil {
			t.logger.Warningf("Error creating command reader: %v, falling back to zero-input system", terminalError)
			break
		}
		t.processCommand(command)
	}

	if terminalError != nil {
		for {
			if t.shutdown {
				break
			}
			time.Sleep(1 * time.Second)
		}
	}

	t.logger.Info("Terminal shutdown")
}

func (t *Terminal) getServiceIndex(serviceName string) int {
	for i, s := range t.services {
		if s.Name() == serviceName {
			return i
		}
	}
	return -1
}

func (t *Terminal) forceQuit() {
	os.Exit(0)
}
