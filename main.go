package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"telemetry-demo/handlers"
	"telemetry-demo/store"
	"telemetry-demo/telemetry"
)

func main() {
	// Initialize tracing
	cleanup := telemetry.InitTracer()
	defer cleanup()

	// Create in-memory store
	memStore := store.NewMemoryStore()

	// Create handlers
	v0Handler := handlers.NewV0Handler(memStore)
	v1Handler := handlers.NewV1Handler(memStore)

	// Setup Gin router
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// V0 Routes - Basic Logging
	v0 := router.Group("/v0")
	{
		v0.POST("/subscribers", v0Handler.CreateSubscriber)
		v0.GET("/subscribers", v0Handler.GetSubscribers)
		v0.GET("/subscribers/:id", v0Handler.GetSubscriber)
	}

	// V1 Routes - Manual Tracing
	v1 := router.Group("/v1")
	{
		v1.POST("/subscribers", v1Handler.CreateSubscriber)
		v1.GET("/subscribers", v1Handler.GetSubscribers)
		v1.GET("/subscribers/:id", v1Handler.GetSubscriber)
	}

	// V2 Routes - Middleware Magic (isolated middleware!)
	v2Handler := handlers.NewV2Handler(memStore)
	
	// Create V2 group with isolated middleware
	v2 := router.Group("/v2")
	v2.Use(otelgin.Middleware("telemetry-demo"))  // Only applies to V2 routes!
	{
		v2.POST("/subscribers", v2Handler.CreateSubscriber)
		v2.GET("/subscribers", v2Handler.GetSubscribers) 
		v2.GET("/subscribers/:id", v2Handler.GetSubscriber)
	}

	log.Println("🚀 Starting Telemetry Demo Server on :8080")
	log.Println("📊 V0 endpoints available at /v0/subscribers (basic logging)")
	log.Println("🔍 V1 endpoints available at /v1/subscribers (manual tracing)")
	log.Println("✨ V2 endpoints available at /v2/subscribers (automatic middleware)")
	log.Fatal(router.Run(":8080"))
}
