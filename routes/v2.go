package routes

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"telemetry-demo/handlers"
	"telemetry-demo/store"
)

// CreateV2Router creates an isolated router for V2 with OpenTelemetry middleware
func CreateV2Router(store *store.MemoryStore) *gin.Engine {
	// Create a fresh router instance (isolated from main router)
	router := gin.New()
	
	// Add standard Gin middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	// Add OpenTelemetry middleware - this is the magic!
	router.Use(otelgin.Middleware("telemetry-demo"))
	
	// Create V2 handler
	v2Handler := handlers.NewV2Handler(store)
	
	// V2 Routes with automatic instrumentation
	v2 := router.Group("/v2") 
	{
		v2.POST("/subscribers", v2Handler.CreateSubscriber)
		v2.GET("/subscribers", v2Handler.GetSubscribers)
		v2.GET("/subscribers/:id", v2Handler.GetSubscriber)
	}
	
	return router
}