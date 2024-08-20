package main

import (
	"log"
	"net/http"
	"os"
	"time"

	v1 "github.com/ArtemVoronov/clearway-task-assets-service/internal/api/rest/v1"
	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app"
	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app/utils"
	"github.com/ArtemVoronov/clearway-task-assets-service/internal/services"
)

func main() {
	readAppConfig()
	initAppServices()
	initAppMonitoring()

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

func initRestApiRoutes() http.Handler {
	routes := http.NewServeMux()
	routes.Handle("GET /api/assets", v1.AuthRequired(v1.LoadAssetsList))
	routes.Handle("POST /api/upload-asset/{name}", v1.AuthRequired(v1.StoreAsset))
	routes.Handle("GET /api/asset/{name}", v1.AuthRequired(v1.LoadAsset))
	routes.Handle("DELETE /api/asset/{name}", v1.AuthRequired(v1.DeleteAsset))
	routes.Handle("POST /api/auth", v1.ErrorHandleRequired(v1.Authenicate))
	routes.Handle("POST /api/users", v1.ErrorHandleRequired(v1.CreateUser))

	// CORS
	processOptionsRequestsFunc := v1.NewProcessOptionsRequestsFunc()
	routes.HandleFunc("OPTIONS /api/assets", processOptionsRequestsFunc)
	routes.HandleFunc("OPTIONS /api/upload-asset/{name}", processOptionsRequestsFunc)
	routes.HandleFunc("OPTIONS /api/asset/{name}", processOptionsRequestsFunc)
	routes.HandleFunc("OPTIONS /api/auth", processOptionsRequestsFunc)
	routes.HandleFunc("OPTIONS /api/users", processOptionsRequestsFunc)

	return v1.NewLoggerHandler(v1.NewBodySizeLimitHandler(routes))
}

func onShutdown() {
	err := services.Instance().Shutdown()
	log.Fatalf("error during services shutdown: %s", err)
}

func initAppMonitoring() {
	enableRuntimeMonitoring, ok := os.LookupEnv("APP_ENABLE_RUNTIME_MONITORING")
	if !ok {
		enableRuntimeMonitoring = "false"
	}

	if enableRuntimeMonitoring == "true" {
		go func() {
			for {
				time.Sleep(5 * time.Second)
				utils.PrintMemUsage()
			}
		}()
	}
}
