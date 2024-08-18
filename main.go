package main

import (
	"log"
	"net/http"

	v1 "github.com/ArtemVoronov/clearway-task-assets-service/internal/api/rest/v1"
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

	routes := initRestApiRoutes()

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

func initRestApiRoutes() *http.ServeMux {
	routes := http.NewServeMux()
	// TODO: add tech endpoints
	routes.HandleFunc("/", v1.HandleRoute)
	return routes
}

func onShutdown() {
	err := services.Instance().Shutdown()
	log.Fatalf("error during services shutdown: %s", err)
}
