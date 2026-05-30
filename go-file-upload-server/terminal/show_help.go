package terminal

func (t *Terminal) showHelp() {
	t.logger.Info("Available commands:")
	t.logger.Info("start <service> - Start a service")
	t.logger.Info("restart <service> - Restart a service")
	t.logger.Info("stop <service> - Stop a service")
	t.logger.Info("status <service> - Show the status of a service")
	t.logger.Info("status - Show the status of all services")
	t.logger.Info("clear - Clear the terminal")
	t.logger.Info("shutdown - Shutdown the system")
	t.logger.Info("exit - Exit the terminal")
	t.logger.Info("help - Show this help message")
}
