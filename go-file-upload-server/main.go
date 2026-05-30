package main

import (
	"flag"
	"go-file-upload-server/controllers"
	"go-file-upload-server/logging"
	"go-file-upload-server/repositories/fileupload"
	"go-file-upload-server/routes"
	"go-file-upload-server/service"
	"go-file-upload-server/services/httpserver"
	"go-file-upload-server/terminal"
)

func main() {
	httpServerConfigPath := flag.String("http-server-config", "configs/http-server-config.json", "The path to the http server config file")
	fileUploadConfigPath := flag.String("file-upload-config", "configs/file-upload-config.json", "The path to the file upload config file")
	fileUploadSecretsPath := flag.String("file-upload-secrets", "secrets/file-upload-secrets.json", "The path to the file upload secrets file")
	flag.Parse()

	httpServerConfig, err := httpserver.ConfigFromJson(*httpServerConfigPath)
	if err != nil {
		panic(err)
	}

	fileUploadConfig, err := fileupload.LoadConfigFromJson(*fileUploadConfigPath)
	if err != nil {
		panic(err)
	}

	fileUploadSecrets, err := fileupload.LoadSecretsFromJson(*fileUploadSecretsPath)
	if err != nil {
		panic(err)
	}

	fileUploadRepo, err := fileupload.NewPostgresFileUploadRepository(fileUploadConfig, fileUploadSecrets)
	if err != nil {
		panic(err)
	}

	logger := logging.NewLogger()
	logger.ServiceName = "Main"
	errorHandler, err := httpserver.NewHttpErrorHandler(&logger)
	if err != nil {
		panic(err)
	}

	applicationController, controllerList, err := controllers.SetupControllers(&errorHandler)
	if err != nil {
		panic(err)
	}

	fileUploadsController := controllers.NewFileUploadsController(fileUploadRepo, &errorHandler)
	controllerList = append(controllerList, fileUploadsController)

	routes, err := routes.Routes(controllerList, applicationController)
	if err != nil {
		panic(err)
	}

	httpServer, err := httpserver.NewHttpServer(httpServerConfig, routes, &logger)
	if err != nil {
		panic(err)
	}

	terminal := terminal.NewTerminal([]service.Service{httpServer})
	terminal.Start()
}
