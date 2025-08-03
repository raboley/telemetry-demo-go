package main

import (
	"log"
	
	"github.com/gin-gonic/gin"
	"telemetry-demo/handlers"
	"telemetry-demo/store"
)

func main() {
	// Create in-memory store
	memStore := store.NewMemoryStore()
	
	// Create handlers
	v0Handler := handlers.NewV0Handler(memStore)
	
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
	
	log.Println("🚀 Starting Telemetry Demo Server on :8080")
	log.Println("📊 V0 endpoints available at /v0/subscribers")
	log.Fatal(router.Run(":8080"))
}