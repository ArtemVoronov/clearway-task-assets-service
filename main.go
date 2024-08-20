// Assets Service:
//
//	Schemes: https
//	BasePath: /
//	Version: 1.0
//	Host: https://localhost:3005
//
//  SecurityDefinitions:
//    Bearer:
//      type: apiKey
//      name: Authorization
//      in: header
// swagger:meta
package main

import (
	"encoding/json"
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

	routes, err := initRestApiRoutes()
	if err != nil {
		log.Fatalf("error during routes creating: %s", err)
	}

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

func initRestApiRoutes() (http.Handler, error) {
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

	// API Spec
	fs := http.FileServer(http.Dir("./api/swagger"))
	routes.Handle("GET /api/doc/", http.StripPrefix("/api/doc/", fs))
	routes.Handle("GET /api/", v1.ErrorHandleRequired(apiSpec))
	routes.Handle("GET /health", v1.ErrorHandleRequired(health))

	bodyMaxSize, err := app.ParseBodyMaxSize()
	if err != nil {
		return nil, err
	}

	return v1.NewLoggerHandler(v1.NewBodySizeLimitHandler(routes, bodyMaxSize)), nil
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

func apiSpec(w http.ResponseWriter, r *http.Request) error {
	spec, err := os.ReadFile("./api/swagger/swagger.yaml")
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(spec)
	return nil
}

// Used rendering cumulative information about the readiness and performance of the service
// swagger:model
type AppInfo struct {
	// Version
	// in: string
	Version string `json:"version"`
	// State
	// in: string
	State string `json:"state"`
}

func health(w http.ResponseWriter, r *http.Request) error {
	appInfo := AppInfo{"1.0", "running"}
	result, err := json.Marshal(appInfo)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(result)
	return nil
}
