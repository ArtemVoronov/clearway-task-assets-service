package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type HttpServerConfig struct {
	Host                    string
	CertificateFilePath     string
	KeyFilePath             string
	ReadTimeout             time.Duration
	WriteTimeout            time.Duration
	GracefulShutdownTimeout time.Duration
}

func StartHttpServer(config *HttpServerConfig, routes *http.ServeMux, shutdown func()) {
	if shutdown != nil {
		defer shutdown()
	}

	server := &http.Server{
		Handler:      routes,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		Addr:         config.Host,
	}

	idleCloseConnections := make(chan struct{})
	go func() {
		interruptSignals := make(chan os.Signal, 1)
		signal.Notify(interruptSignals, syscall.SIGINT, syscall.SIGTERM)
		<-interruptSignals
		log.Println("server shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), config.GracefulShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("server forced to shutdown: %v", err)
		}
		close(idleCloseConnections)
	}()

	if err := server.ListenAndServeTLS(config.CertificateFilePath, config.KeyFilePath); err != http.ErrServerClosed {
		log.Fatal(err)
	}

	<-idleCloseConnections

	log.Println("server has been shutdown")
}
