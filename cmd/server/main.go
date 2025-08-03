package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"

	"telemetry-go/internal/app"
	"telemetry-go/internal/logging"
	"telemetry-go/internal/telemetry"
)

func main() {
	logger := logging.NewLogger()

	tp, err := telemetry.InitTracing("subscriber-api", "1.0.0")
	if err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer func() {
		if err := telemetry.ShutdownTracing(context.Background(), tp); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	config := &app.Config{
		ServiceName:    "subscriber-api",
		ServiceVersion: "1.0.0",
		Port:           "8080",
		Logger:         logger,
		TracerProvider: otel.GetTracerProvider(),
		GinMode:        "", // Use default (debug mode)
	}

	application := app.Build(config)

	go func() {
		if err := application.Run(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}