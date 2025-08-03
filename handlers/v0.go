package handlers

import (
	"bytes"
	"io"
	"net/http"
	"strconv" 
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"telemetry-demo/models"
	"telemetry-demo/store"
)

type V0Handler struct {
	store  *store.MemoryStore
	logger *logrus.Logger
}

func NewV0Handler(store *store.MemoryStore) *V0Handler {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "15:04:05",
		FullTimestamp:   true,
		ForceColors:     true,
	})
	
	return &V0Handler{
		store:  store,
		logger: logger,
	}
}

func (h *V0Handler) CreateSubscriber(c *gin.Context) {
	start := time.Now()
	
	// Read and preserve raw body for logging
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	
	var req models.Subscriber
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"method":      "POST",
			"endpoint":    "/v0/subscribers",
			"error":       err.Error(),
			"raw_body":    string(body),
			"duration":    time.Since(start),
		}).Error("Invalid request body")
		
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Simulate some processing time
	time.Sleep(50 * time.Millisecond)
	
	subscriber := h.store.CreateSubscriber(req.Name, req.Email)
	
	h.logger.WithFields(logrus.Fields{
		"method":         "POST",
		"endpoint":      "/v0/subscribers",
		"subscriber_id": subscriber.ID,
		"name":          subscriber.Name,
		"email":         subscriber.Email,
		"duration":      time.Since(start),
	}).Info("Subscriber created successfully")
	
	c.JSON(http.StatusCreated, subscriber)
}

func (h *V0Handler) GetSubscribers(c *gin.Context) {
	start := time.Now()
	
	// Simulate database query time
	time.Sleep(30 * time.Millisecond)
	
	subscribers := h.store.GetAllSubscribers()
	
	h.logger.WithFields(logrus.Fields{
		"method":    "GET",
		"endpoint":  "/v0/subscribers",
		"count":     len(subscribers),
		"duration":  time.Since(start),
	}).Info("Retrieved all subscribers")
	
	c.JSON(http.StatusOK, gin.H{
		"subscribers": subscribers,
		"count":       len(subscribers),
	})
}

func (h *V0Handler) GetSubscriber(c *gin.Context) {
	start := time.Now()
	
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"method":    "GET",
			"endpoint":  "/v0/subscribers/:id",
			"id":        idStr,
			"error":     "Invalid ID format",
			"duration":  time.Since(start),
		}).Error("Invalid subscriber ID")
		
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscriber ID"})
		return
	}
	
	// Simulate database lookup time
	time.Sleep(20 * time.Millisecond)
	
	subscriber, exists := h.store.GetSubscriber(id)
	if !exists {
		h.logger.WithFields(logrus.Fields{
			"method":        "GET",
			"endpoint":      "/v0/subscribers/:id",
			"subscriber_id": id,
			"duration":      time.Since(start),
		}).Warn("Subscriber not found")
		
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
		return
	}
	
	h.logger.WithFields(logrus.Fields{
		"method":        "GET",
		"endpoint":      "/v0/subscribers/:id",
		"subscriber_id": subscriber.ID,
		"name":          subscriber.Name,
		"email":         subscriber.Email,
		"duration":      time.Since(start),
	}).Info("Retrieved subscriber")
	
	c.JSON(http.StatusOK, subscriber)
}