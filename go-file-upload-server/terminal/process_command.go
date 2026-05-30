package terminal

import (
	"strings"
)

func (t *Terminal) processCommand(command string) {
	command = strings.TrimSpace(command)

	if command == "" {
		return
	}

	if command == "shutdown" || command == "exit" {
		t.handleShutdown()
		return
	}

	if command == "start" {
		t.logger.Warning("Please specify a service to start")
		return
	}

	if command == "restart" {
		t.logger.Warning("Please specify a service to restart")
		return
	}

	if command == "stop" {
		t.logger.Warning("Please specify a service to stop. To shutdown the system, use 'shutdown' or 'exit'.")
		return
	}

	if command == "status" {
		t.handleStatuses()
		return
	}

	if command == "help" {
		t.showHelp()
		return
	}

	if command == "clear" {
		t.handleClear()
		return
	}

	if after, ok := strings.CutPrefix(command, "start "); ok {
		t.handleStartService(after)
		return
	}

	if after, ok := strings.CutPrefix(command, "stop "); ok {
		t.handleStopService(after)
		return
	}

	if after, ok := strings.CutPrefix(command, "restart "); ok {
		t.handleRestartService(after)
		return
	}

	if after, ok := strings.CutPrefix(command, "status "); ok {
		t.handleStatus(after)
		return
	}

	t.logger.Warningf("Unknown command: %v", command)
}
