package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"telemetry-go/internal/cache"
	"telemetry-go/internal/handlers"
	"telemetry-go/internal/logging"
	"telemetry-go/internal/repository"
	"telemetry-go/internal/service"
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

	repo := repository.NewInMemorySubscriberRepository()
	cacheInstance := cache.NewInMemoryCache()
	subscriberService := service.NewSubscriberService(repo, cacheInstance, logger)
	subscriberHandler := handlers.NewSubscriberHandler(subscriberService, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("subscriber-api"))

	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.WithTracing(c.Request.Context()).WithFields(map[string]interface{}{
			"method":     method,
			"path":       path,
			"status":     status,
			"latency_ms": latency.Milliseconds(),
			"user_agent": c.Request.UserAgent(),
		}).Info("HTTP request completed")
	})

	api := r.Group("/api/v1")
	{
		subscribers := api.Group("/subscribers")
		{
			subscribers.POST("", subscriberHandler.CreateSubscriber)
			subscribers.GET("", subscriberHandler.GetAllSubscribers)
			subscribers.GET("/:id", subscriberHandler.GetSubscriber)
			subscribers.PUT("/:id", subscriberHandler.UpdateSubscriber)
			subscribers.DELETE("/:id", subscriberHandler.DeleteSubscriber)
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   "subscriber-api",
		})
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		logger.Info("Starting server on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}