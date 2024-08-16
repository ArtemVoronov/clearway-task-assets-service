package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app"
)

func main() {
	appConfig, err := app.InitAppConfig()
	if err != nil {
		log.Fatal(err)
	}

	routes := http.NewServeMux()
	// TODO: init routes for auth and assets
	routes.HandleFunc("/", processCommon)

	app.Start(appConfig, routes, setup, shutdown)
}

func setup() {
	// TODO: clean and setup services
	fmt.Println("setup")
}

func shutdown() {
	// TODO: clean and shutdown services
	fmt.Println("shutdown")
	time.Sleep(2 * time.Second)
}

// TODO: remove, add appropriate handlers for each REST endpoint
func processCommon(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("request RemoteAddr: %v\n", r.RemoteAddr)
	contentLength, ok := r.Header["Content-Length"]
	if ok {
		fmt.Printf("header Content-Length: %v\n", contentLength)
	}
	authHeader, ok := r.Header["Authorization"]
	if ok {
		fmt.Printf("header Content-Length: %v\n", authHeader)
	}
	w.Write([]byte(fmt.Sprintf("got: %s\n", r.URL)))
	w.WriteHeader(200)
}
