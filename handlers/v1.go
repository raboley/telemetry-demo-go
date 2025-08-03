package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"telemetry-demo/models"
	"telemetry-demo/store"
)

type V1Handler struct {
	store  *store.MemoryStore
	logger *logrus.Logger
	tracer trace.Tracer
}

func NewV1Handler(store *store.MemoryStore) *V1Handler {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "15:04:05",
		FullTimestamp:   true,
		ForceColors:     true,
	})
	
	return &V1Handler{
		store:  store,
		logger: logger,
		tracer: otel.Tracer("telemetry-demo/v1"),
	}
}

func (h *V1Handler) CreateSubscriber(c *gin.Context) {
	// Start root span for the entire request
	ctx, span := h.tracer.Start(c.Request.Context(), "create_subscriber_request")
	defer span.End()
	
	start := time.Now()
	
	// Add basic request attributes to the span
	span.SetAttributes(
		attribute.String("http.method", "POST"),
		attribute.String("http.route", "/v1/subscribers"),
		attribute.String("component", "http_handler"),
	)
	
	var req models.Subscriber
	if err := c.ShouldBindJSON(&req); err != nil {
		// Mark span as error and add error details
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		span.SetAttributes(attribute.String("error.type", "validation_error"))
		
		h.logger.WithFields(logrus.Fields{
			"method":    "POST",
			"endpoint":  "/v1/subscribers",
			"error":     err.Error(),
			"duration":  time.Since(start),
			"trace_id":  span.SpanContext().TraceID().String(),
		}).Error("Invalid request body")
		
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Add user context to span
	span.SetAttributes(
		attribute.String("user.name", req.Name),
		attribute.String("user.email", req.Email),
	)
	
	// Create child span for validation
	ctx, validationSpan := h.tracer.Start(ctx, "validate_subscriber_data")
	validationSpan.SetAttributes(
		attribute.String("validation.name", req.Name),
		attribute.String("validation.email", req.Email),
	)
	
	// Simulate validation work
	time.Sleep(20 * time.Millisecond)
	validationSpan.SetStatus(codes.Ok, "Validation successful")
	validationSpan.End()
	
	// Create child span for database operation
	ctx, dbSpan := h.tracer.Start(ctx, "store_subscriber")
	dbSpan.SetAttributes(
		attribute.String("operation", "create"),
		attribute.String("store.type", "memory"),
	)
	
	// Simulate database work
	time.Sleep(50 * time.Millisecond)
	subscriber := h.store.CreateSubscriber(req.Name, req.Email)
	
	// Add result to database span
	dbSpan.SetAttributes(
		attribute.Int("subscriber.id", subscriber.ID),
		attribute.String("subscriber.name", subscriber.Name),
		attribute.String("subscriber.email", subscriber.Email),
	)
	dbSpan.SetStatus(codes.Ok, "Subscriber created successfully")
	dbSpan.End()
	
	// Add final attributes to root span
	span.SetAttributes(
		attribute.Int("subscriber.id", subscriber.ID),
		attribute.Int("http.status_code", http.StatusCreated),
	)
	span.SetStatus(codes.Ok, "Request completed successfully")
	
	h.logger.WithFields(logrus.Fields{
		"method":         "POST",
		"endpoint":      "/v1/subscribers",
		"subscriber_id": subscriber.ID,
		"name":          subscriber.Name,
		"email":         subscriber.Email,
		"duration":      time.Since(start),
		"trace_id":      span.SpanContext().TraceID().String(),
		"span_id":       span.SpanContext().SpanID().String(),
	}).Info("Subscriber created successfully")
	
	c.JSON(http.StatusCreated, subscriber)
}

func (h *V1Handler) GetSubscribers(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "get_subscribers_request")
	defer span.End()
	
	start := time.Now()
	
	span.SetAttributes(
		attribute.String("http.method", "GET"),
		attribute.String("http.route", "/v1/subscribers"),
		attribute.String("component", "http_handler"),
	)
	
	// Create child span for database query
	ctx, dbSpan := h.tracer.Start(ctx, "query_all_subscribers")
	dbSpan.SetAttributes(
		attribute.String("operation", "read_all"),
		attribute.String("store.type", "memory"),
	)
	
	// Simulate database query time
	time.Sleep(30 * time.Millisecond)
	subscribers := h.store.GetAllSubscribers()
	
	dbSpan.SetAttributes(attribute.Int("result.count", len(subscribers)))
	dbSpan.SetStatus(codes.Ok, fmt.Sprintf("Retrieved %d subscribers", len(subscribers)))
	dbSpan.End()
	
	span.SetAttributes(
		attribute.Int("subscribers.count", len(subscribers)),
		attribute.Int("http.status_code", http.StatusOK),
	)
	span.SetStatus(codes.Ok, "Request completed successfully")
	
	h.logger.WithFields(logrus.Fields{
		"method":    "GET",
		"endpoint":  "/v1/subscribers",
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

func (h *V1Handler) GetSubscriber(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "get_subscriber_request")
	defer span.End()
	
	start := time.Now()
	idStr := c.Param("id")
	
	span.SetAttributes(
		attribute.String("http.method", "GET"),
		attribute.String("http.route", "/v1/subscribers/:id"),
		attribute.String("component", "http_handler"),
		attribute.String("subscriber.id_param", idStr),
	)
	
	// Create child span for ID parsing
	ctx, parseSpan := h.tracer.Start(ctx, "parse_subscriber_id")
	parseSpan.SetAttributes(attribute.String("id_string", idStr))
	
	id, err := strconv.Atoi(idStr)
	if err != nil {
		parseSpan.RecordError(err)
		parseSpan.SetStatus(codes.Error, "Invalid ID format")
		parseSpan.End()
		
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid subscriber ID")
		span.SetAttributes(attribute.String("error.type", "parsing_error"))
		
		h.logger.WithFields(logrus.Fields{
			"method":    "GET",
			"endpoint":  "/v1/subscribers/:id",
			"id":        idStr,
			"error":     "Invalid ID format",
			"duration":  time.Since(start),
			"trace_id":  span.SpanContext().TraceID().String(),
		}).Error("Invalid subscriber ID")
		
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscriber ID"})
		return
	}
	
	parseSpan.SetAttributes(attribute.Int("parsed_id", id))
	parseSpan.SetStatus(codes.Ok, "ID parsed successfully")
	parseSpan.End()
	
	// Create child span for database lookup
	ctx, dbSpan := h.tracer.Start(ctx, "lookup_subscriber")
	dbSpan.SetAttributes(
		attribute.String("operation", "read_by_id"),
		attribute.String("store.type", "memory"),
		attribute.Int("subscriber.id", id),
	)
	
	// Simulate database lookup time
	time.Sleep(20 * time.Millisecond)
	subscriber, exists := h.store.GetSubscriber(id)
	
	if !exists {
		dbSpan.SetStatus(codes.Error, "Subscriber not found")
		dbSpan.End()
		
		span.SetAttributes(
			attribute.Int("subscriber.id", id),
			attribute.Int("http.status_code", http.StatusNotFound),
		)
		span.SetStatus(codes.Error, "Subscriber not found")
		
		h.logger.WithFields(logrus.Fields{
			"method":        "GET",
			"endpoint":      "/v1/subscribers/:id",
			"subscriber_id": id,
			"duration":      time.Since(start),
			"trace_id":      span.SpanContext().TraceID().String(),
		}).Warn("Subscriber not found")
		
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
		return
	}
	
	dbSpan.SetAttributes(
		attribute.String("subscriber.name", subscriber.Name),
		attribute.String("subscriber.email", subscriber.Email),
	)
	dbSpan.SetStatus(codes.Ok, "Subscriber found")
	dbSpan.End()
	
	span.SetAttributes(
		attribute.Int("subscriber.id", subscriber.ID),
		attribute.String("subscriber.name", subscriber.Name),
		attribute.String("subscriber.email", subscriber.Email),
		attribute.Int("http.status_code", http.StatusOK),
	)
	span.SetStatus(codes.Ok, "Request completed successfully")
	
	h.logger.WithFields(logrus.Fields{
		"method":        "GET",
		"endpoint":      "/v1/subscribers/:id",
		"subscriber_id": subscriber.ID,
		"name":          subscriber.Name,
		"email":         subscriber.Email,
		"duration":      time.Since(start),
		"trace_id":      span.SpanContext().TraceID().String(),
		"span_id":       span.SpanContext().SpanID().String(),
	}).Info("Retrieved subscriber")
	
	c.JSON(http.StatusOK, subscriber)
}