package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"telemetry-go/internal/logging"
	"telemetry-go/internal/models"
	"telemetry-go/internal/service"
)

type SubscriberHandler struct {
	service *service.SubscriberService
	logger  *logging.ContextLogger
	tracer  trace.Tracer
}

func NewSubscriberHandler(service *service.SubscriberService, logger *logging.ContextLogger) *SubscriberHandler {
	return &SubscriberHandler{
		service: service,
		logger:  logger,
		tracer:  otel.Tracer("subscriber-handler"),
	}
}

func (h *SubscriberHandler) CreateSubscriber(c *gin.Context) {
	// This will start a span manually using the request context.
	ctx, span := h.tracer.Start(c.Request.Context(), "subscriber.handler.create")
	// defer here will automatically close the span
	defer span.End()

	var req models.CreateSubscriberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.ErrorWithTracing(ctx, "Invalid request payload", err, logrus.Fields{
			"endpoint": "POST /subscribers",
		})
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.InfoWithTracing(ctx, "Received create subscriber request", logrus.Fields{
		"email":    req.Email,
		"name":     req.Name,
		"endpoint": "POST /subscribers",
	})

	subscriber, err := h.service.CreateSubscriber(ctx, &req)
	if err != nil {
		h.logger.ErrorWithTracing(ctx, "Failed to create subscriber", err, logrus.Fields{
			"email":    req.Email,
			"endpoint": "POST /subscribers",
		})
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscriber"})
		return
	}

	h.logger.InfoWithTracing(ctx, "Successfully created subscriber", logrus.Fields{
		"subscriber_id": subscriber.ID.String(),
		"email":         subscriber.Email,
		"endpoint":      "POST /subscribers",
	})

	span.SetAttributes(
		attribute.String("subscriber.id", subscriber.ID.String()),
		attribute.String("subscriber.email", subscriber.Email),
		attribute.Bool("success", true),
	)

	c.JSON(http.StatusCreated, subscriber)
}

func (h *SubscriberHandler) GetSubscriber(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "subscriber.handler.get")
	defer span.End()

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.logger.ErrorWithTracing(ctx, "Invalid subscriber ID", err, logrus.Fields{
			"id":       idParam,
			"endpoint": "GET /subscribers/:id",
		})
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscriber ID"})
		return
	}

	h.logger.InfoWithTracing(ctx, "Received get subscriber request", logrus.Fields{
		"subscriber_id": id.String(),
		"endpoint":      "GET /subscribers/:id",
	})

	subscriber, err := h.service.GetSubscriber(ctx, id.String())
	if err != nil {
		h.logger.ErrorWithTracing(ctx, "Failed to get subscriber", err, logrus.Fields{
			"subscriber_id": id.String(),
			"endpoint":      "GET /subscribers/:id",
		})
		span.RecordError(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
		return
	}

	h.logger.InfoWithTracing(ctx, "Successfully retrieved subscriber", logrus.Fields{
		"subscriber_id": subscriber.ID.String(),
		"email":         subscriber.Email,
		"endpoint":      "GET /subscribers/:id",
	})

	span.SetAttributes(
		attribute.String("subscriber.id", subscriber.ID.String()),
		attribute.String("subscriber.email", subscriber.Email),
		attribute.Bool("success", true),
	)

	c.JSON(http.StatusOK, subscriber)
}

func (h *SubscriberHandler) GetAllSubscribers(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "subscriber.handler.get_all")
	defer span.End()

	h.logger.InfoWithTracing(ctx, "Received get all subscribers request", logrus.Fields{
		"endpoint": "GET /subscribers",
	})

	subscribers, err := h.service.GetAllSubscribers(ctx)
	if err != nil {
		h.logger.ErrorWithTracing(ctx, "Failed to get subscribers", err, logrus.Fields{
			"endpoint": "GET /subscribers",
		})
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve subscribers"})
		return
	}

	h.logger.InfoWithTracing(ctx, "Successfully retrieved all subscribers", logrus.Fields{
		"count":    len(subscribers),
		"endpoint": "GET /subscribers",
	})

	span.SetAttributes(
		attribute.Int("subscriber.count", len(subscribers)),
		attribute.Bool("success", true),
	)

	c.JSON(http.StatusOK, subscribers)
}

func (h *SubscriberHandler) UpdateSubscriber(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "subscriber.handler.update")
	defer span.End()

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.logger.ErrorWithTracing(ctx, "Invalid subscriber ID", err, logrus.Fields{
			"id":       idParam,
			"endpoint": "PUT /subscribers/:id",
		})
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscriber ID"})
		return
	}

	var req models.CreateSubscriberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.ErrorWithTracing(ctx, "Invalid request payload", err, logrus.Fields{
			"subscriber_id": id.String(),
			"endpoint":      "PUT /subscribers/:id",
		})
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.InfoWithTracing(ctx, "Received update subscriber request", logrus.Fields{
		"subscriber_id": id.String(),
		"email":         req.Email,
		"name":          req.Name,
		"endpoint":      "PUT /subscribers/:id",
	})

	subscriber, err := h.service.UpdateSubscriber(ctx, id.String(), &req)
	if err != nil {
		h.logger.ErrorWithTracing(ctx, "Failed to update subscriber", err, logrus.Fields{
			"subscriber_id": id.String(),
			"endpoint":      "PUT /subscribers/:id",
		})
		span.RecordError(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
		return
	}

	h.logger.InfoWithTracing(ctx, "Successfully updated subscriber", logrus.Fields{
		"subscriber_id": subscriber.ID.String(),
		"email":         subscriber.Email,
		"endpoint":      "PUT /subscribers/:id",
	})

	span.SetAttributes(
		attribute.String("subscriber.id", subscriber.ID.String()),
		attribute.String("subscriber.email", subscriber.Email),
		attribute.Bool("success", true),
	)

	c.JSON(http.StatusOK, subscriber)
}

func (h *SubscriberHandler) DeleteSubscriber(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "subscriber.handler.delete")
	defer span.End()

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.logger.ErrorWithTracing(ctx, "Invalid subscriber ID", err, logrus.Fields{
			"id":       idParam,
			"endpoint": "DELETE /subscribers/:id",
		})
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscriber ID"})
		return
	}

	h.logger.InfoWithTracing(ctx, "Received delete subscriber request", logrus.Fields{
		"subscriber_id": id.String(),
		"endpoint":      "DELETE /subscribers/:id",
	})

	err = h.service.DeleteSubscriber(ctx, id.String())
	if err != nil {
		h.logger.ErrorWithTracing(ctx, "Failed to delete subscriber", err, logrus.Fields{
			"subscriber_id": id.String(),
			"endpoint":      "DELETE /subscribers/:id",
		})
		span.RecordError(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscriber not found"})
		return
	}

	h.logger.InfoWithTracing(ctx, "Successfully deleted subscriber", logrus.Fields{
		"subscriber_id": id.String(),
		"endpoint":      "DELETE /subscribers/:id",
	})

	span.SetAttributes(
		attribute.String("subscriber.id", id.String()),
		attribute.Bool("success", true),
	)

	c.JSON(http.StatusNoContent, nil)
}
