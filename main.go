package main

import (
	"log"
	"net/http"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/api/rest/v1/assets"
	"github.com/ArtemVoronov/clearway-task-assets-service/internal/api/rest/v1/auth"
	"github.com/ArtemVoronov/clearway-task-assets-service/internal/api/rest/v1/users"
	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app"
	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

func main() {
	readAppConfig()
	initAppServices()

	httpServerConfig, err := app.NewHttpServerConfig()
	if err != nil {
		log.Fatalf("error during server config creating: %s", err)
	}

	routes := http.NewServeMux()
	// TODO: add tech endpoints
	routes.HandleFunc("/api/auth", auth.ProcessAuthRoute)
	routes.HandleFunc("/api/users", users.ProcessUsersRoute)
	routes.HandleFunc("/api/assets", assets.ProcessAssetsRoute)
	routes.HandleFunc("/api/assets/:name", assets.ProcessAssetsRoute)
	routes.HandleFunc("/api/upload-assets/:name", assets.ProcessAssetsRoute)

	app.StartHttpServer(httpServerConfig, routes, onShutdown)
}

func readAppConfig() {
	err := app.SetUpEnvVarsFromConfig()
	if err != nil {
		log.Fatalf("error during app config reading: %s", err)
	}
}

func initAppServices() {
	services.Instance()
}

func onShutdown() {
	err := services.Instance().Shutdown()
	log.Fatalf("error during services shutdown: %s", err)
}
