package app

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/trace"

	"telemetry-go/internal/cache"
	"telemetry-go/internal/handlers"
	"telemetry-go/internal/logging"
	"telemetry-go/internal/repository"
	"telemetry-go/internal/service"
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	Port           string
	Logger         *logging.ContextLogger
	TracerProvider trace.TracerProvider
	GinMode        string
	Repository     repository.SubscriberRepository // Allow injecting any repository implementation
}

type Application struct {
	server  *http.Server
	config  *Config
	router  *gin.Engine
	repo    repository.SubscriberRepository
	cache   *cache.InMemoryCache
	service *service.SubscriberService
	handler *handlers.SubscriberHandler
}

func Build(config *Config) *Application {
	if config.GinMode != "" {
		gin.SetMode(config.GinMode)
	}

	// Use injected repository or fall back to in-memory
	var repo repository.SubscriberRepository
	if config.Repository != nil {
		repo = config.Repository
	} else {
		repo = repository.NewInMemorySubscriberRepository()
	}
	
	cacheInstance := cache.NewInMemoryCache()
	subscriberService := service.NewSubscriberService(repo, cacheInstance, config.Logger)
	subscriberHandler := handlers.NewSubscriberHandler(subscriberService, config.Logger)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware(config.ServiceName))

	router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		config.Logger.WithTracing(c.Request.Context()).WithFields(map[string]interface{}{
			"method":     method,
			"path":       path,
			"status":     status,
			"latency_ms": latency.Milliseconds(),
			"user_agent": c.Request.UserAgent(),
		}).Info("HTTP request completed")
	})

	api := router.Group("/api/v1")
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

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   config.ServiceName,
		})
	})

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	return &Application{
		server:  server,
		config:  config,
		router:  router,
		repo:    repo,
		cache:   cacheInstance,
		service: subscriberService,
		handler: subscriberHandler,
	}
}

func (app *Application) Run() error {
	app.config.Logger.Info("Starting server on :" + app.config.Port)
	if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (app *Application) Shutdown(ctx context.Context) error {
	app.config.Logger.Info("Shutting down server...")
	return app.server.Shutdown(ctx)
}

func (app *Application) GetRepo() repository.SubscriberRepository {
	return app.repo
}

func (app *Application) GetCache() *cache.InMemoryCache {
	return app.cache
}

func (app *Application) GetService() *service.SubscriberService {
	return app.service
}

func (app *Application) GetHandler() *handlers.SubscriberHandler {
	return app.handler
}

func (app *Application) GetRouter() *gin.Engine {
	return app.router
}
