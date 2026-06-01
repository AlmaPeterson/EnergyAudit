package main

import (
	"flag"
	"os"
	"go-file-upload-server/controllers"
	"go-file-upload-server/logging"
	"go-file-upload-server/repositories/fileupload"
	userrepo "go-file-upload-server/repositories/user"
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

	userRepo, err := userrepo.NewPostgresUserRepository(fileUploadConfig, fileUploadSecrets)
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

	// auth middleware using JWT secret from env (set JWT_SECRET)
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	authMw := httpserver.AuthMiddleware(&userRepo, jwtSecret)

	// register auth controller
	authController := controllers.NewAuthController(userRepo, &errorHandler)
	controllerList = append(controllerList, authController)

	foldersController := controllers.NewFoldersController(fileUploadRepo, &errorHandler, authMw)
	controllerList = append(controllerList, foldersController)

	fileUploadsController := controllers.NewFileUploadsController(fileUploadRepo, &errorHandler, authMw)
	controllerList = append(controllerList, fileUploadsController)

	sharesController := controllers.NewSharesController(fileUploadRepo, &userRepo, &errorHandler, authMw)
	controllerList = append(controllerList, sharesController)

	jobsController := controllers.NewJobsController(fileUploadRepo, &errorHandler, authMw)
	controllerList = append(controllerList, jobsController)

	tasksController := controllers.NewTasksController(fileUploadRepo, &errorHandler, authMw)
	controllerList = append(controllerList, tasksController)

	timeEntriesController := controllers.NewTimeEntriesController(fileUploadRepo, &errorHandler, authMw)
	controllerList = append(controllerList, timeEntriesController)

	energyAuditsController := controllers.NewEnergyAuditsController(fileUploadRepo, &errorHandler, authMw)
	controllerList = append(controllerList, energyAuditsController)

	imagesController := controllers.NewImagesController(fileUploadRepo, &errorHandler, authMw)
	controllerList = append(controllerList, imagesController)

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
