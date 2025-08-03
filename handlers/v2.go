package handlers

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"telemetry-demo/models"
	"telemetry-demo/store"
)

type V2Handler struct {
	store  *store.MemoryStore
	logger *logrus.Logger
}

func NewV2Handler(store *store.MemoryStore) *V2Handler {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "15:04:05",
		FullTimestamp:   true,
		ForceColors:     true,
	})
	
	return &V2Handler{
		store:  store,
		logger: logger,
	}
}

func (h *V2Handler) CreateSubscriber(c *gin.Context) {
	start := time.Now()
	
	// Get current span from middleware (automatically created!)
	span := trace.SpanFromContext(c.Request.Context())
	
	// Read and preserve raw body for logging  
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	
	var req models.Subscriber
	if err := c.ShouldBindJSON(&req); err != nil {
		// Add business context to the automatic span
		span.SetAttributes(
			attribute.String("error.type", "validation_error"),
			attribute.String("request.body", string(body)),
		)
		
		h.logger.WithFields(logrus.Fields{
			"method":    "POST",
			"endpoint":  "/v2/subscribers",
			"error":     err.Error(),
			"raw_body":  string(body),
			"duration":  time.Since(start),
			"trace_id":  span.SpanContext().TraceID().String(),
		}).Error("Invalid request body")
		
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Add business context to span (HTTP context already handled by middleware!)
	span.SetAttributes(
		attribute.String("user.name", req.Name),
		attribute.String("user.email", req.Email),
	)
	
	// Pure business logic - no span management needed!
	h.validateSubscriberData(c, req.Name, req.Email)
	subscriber := h.storeSubscriber(c, req.Name, req.Email)
	
	// Add result to span
	span.SetAttributes(attribute.Int("subscriber.id", subscriber.ID))
	
	h.logger.WithFields(logrus.Fields{
		"method":         "POST",
		"endpoint":      "/v2/subscribers",
		"subscriber_id": subscriber.ID,
		"name":          subscriber.Name,
		"email":         subscriber.Email,
		"duration":      time.Since(start),
		"trace_id":      span.SpanContext().TraceID().String(),
		"span_id":       span.SpanContext().SpanID().String(),
	}).Info("Subscriber created successfully")
	
	c.JSON(http.StatusCreated, subscriber)
}

func (h *V2Handler) GetSubscribers(c *gin.Context) {
	start := time.Now()
	span := trace.SpanFromContext(c.Request.Context())
	
	// Pure business logic
	subscribers := h.queryAllSubscribers(c)
	
	// Add business context to automatic span  
	span.SetAttributes(attribute.Int("subscribers.count", len(subscribers)))
	
	h.logger.WithFields(logrus.Fields{
		"method":    "GET", 
		"endpoint":  "/v2/subscribers",
		"count":     len(subscribers),
		"duration":  time.Since(start),
		"trace_id":  span.SpanContext().TraceID().String(),
		"span_id":   span.SpanContext().SpanID().String(),
	}).Info("Retrieved all subscribers")
	
	c.JSON(http.StatusOK, gin.H{
		"subscribers": subscribers,
		"count":       len(subscribers),
	})
}

func (h *V2Handler) GetSubscriber(c *gin.Context) {
	start := time.Now()
	span := trace.SpanFromContext(c.Request.Context())
	idStr := c.Param("id")
	
	// Add parameter to span
	span.SetAttributes(attribute.String("subscriber.id_param", idStr))
	
	id, err := strconv.Atoi(idStr)
	if err != nil {
		span.SetAttributes(attribute.String("error.type", "parsing_error"))
		
		h.logger.WithFields(logrus.Fields{
			"method":    "GET",
			"endpoint":  "/v2/subscribers/:id", 
			"id":        idStr,
			"error":     "Invalid ID format",
			"duration":  time.Since(start),
			"trace_id":  span.SpanContext().TraceID().String(),
		}).Error("Invalid subscriber ID")
		
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscriber ID"})
		return
	}
	
	// Pure business logic
	subscriber, exists := h.lookupSubscriber(c, id)
	if !exists {
		span.SetAttributes(attribute.Int("subscriber.id", id))
		
		h.logger.WithFields(logrus.Fields{
			"method":        "GET",
			"endpoint":      "/v2/subscribers/:id",
			"subscriber_id": id,
			"duration":      time.Since(start),
			"trace_id":      span.SpanContext().TraceID().String(),
		}).Warn("Subscriber not found")
		
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
		return
	}
	
	// Add business context to span
	span.SetAttributes(
		attribute.Int("subscriber.id", subscriber.ID),
		attribute.String("subscriber.name", subscriber.Name),
		attribute.String("subscriber.email", subscriber.Email),
	)
	
	h.logger.WithFields(logrus.Fields{
		"method":        "GET",
		"endpoint":      "/v2/subscribers/:id",
		"subscriber_id": subscriber.ID,
		"name":          subscriber.Name,
		"email":         subscriber.Email,
		"duration":      time.Since(start),
		"trace_id":      span.SpanContext().TraceID().String(),
		"span_id":       span.SpanContext().SpanID().String(),
	}).Info("Retrieved subscriber")
	
	c.JSON(http.StatusOK, subscriber)
}

// Business logic methods with automatic tracing
func (h *V2Handler) validateSubscriberData(c *gin.Context, name, email string) {
	// Get tracer for custom spans (when needed)
	tracer := otel.Tracer("telemetry-demo/business-logic")
	
	_, span := tracer.Start(c.Request.Context(), "validate_subscriber_data")
	defer span.End()
	
	span.SetAttributes(
		attribute.String("validation.name", name),
		attribute.String("validation.email", email),
	)
	
	// Simulate validation work
	time.Sleep(20 * time.Millisecond)
}

func (h *V2Handler) storeSubscriber(c *gin.Context, name, email string) *models.Subscriber {
	tracer := otel.Tracer("telemetry-demo/business-logic")
	
	_, span := tracer.Start(c.Request.Context(), "store_subscriber")
	defer span.End()
	
	span.SetAttributes(
		attribute.String("operation", "create"),
		attribute.String("store.type", "memory"),
	)
	
	// Simulate database work
	time.Sleep(50 * time.Millisecond)
	subscriber := h.store.CreateSubscriber(name, email)
	
	span.SetAttributes(
		attribute.Int("subscriber.id", subscriber.ID),
		attribute.String("subscriber.name", subscriber.Name),
		attribute.String("subscriber.email", subscriber.Email),
	)
	
	return subscriber
}

func (h *V2Handler) queryAllSubscribers(c *gin.Context) []*models.Subscriber {
	tracer := otel.Tracer("telemetry-demo/business-logic")
	
	_, span := tracer.Start(c.Request.Context(), "query_all_subscribers")
	defer span.End()
	
	span.SetAttributes(
		attribute.String("operation", "read_all"),
		attribute.String("store.type", "memory"),
	)
	
	// Simulate database query time
	time.Sleep(30 * time.Millisecond)
	subscribers := h.store.GetAllSubscribers()
	
	span.SetAttributes(attribute.Int("result.count", len(subscribers)))
	
	return subscribers
}

func (h *V2Handler) lookupSubscriber(c *gin.Context, id int) (*models.Subscriber, bool) {
	tracer := otel.Tracer("telemetry-demo/business-logic")
	
	_, span := tracer.Start(c.Request.Context(), "lookup_subscriber")
	defer span.End()
	
	span.SetAttributes(
		attribute.String("operation", "read_by_id"),
		attribute.String("store.type", "memory"),
		attribute.Int("subscriber.id", id),
	)
	
	// Simulate database lookup time
	time.Sleep(20 * time.Millisecond)
	subscriber, exists := h.store.GetSubscriber(id)
	
	if exists {
		span.SetAttributes(
			attribute.String("subscriber.name", subscriber.Name),
			attribute.String("subscriber.email", subscriber.Email),
		)
	}
	
	return subscriber, exists
}