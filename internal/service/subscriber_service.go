package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"telemetry-go/internal/cache"
	"telemetry-go/internal/logging"
	"telemetry-go/internal/models"
	"telemetry-go/internal/repository"
)

type SubscriberService struct {
	repo   repository.SubscriberRepository
	cache  cache.Cache
	logger *logging.ContextLogger
	tracer trace.Tracer
}

func NewSubscriberService(repo repository.SubscriberRepository, cache cache.Cache, logger *logging.ContextLogger) *SubscriberService {
	return &SubscriberService{
		repo:   repo,
		cache:  cache,
		logger: logger,
		tracer: otel.Tracer("subscriber-service"),
	}
}

func (s *SubscriberService) CreateSubscriber(ctx context.Context, req *models.CreateSubscriberRequest) (*models.Subscriber, error) {
	ctx, span := s.tracer.Start(ctx, "subscriber.service.create",
		trace.WithAttributes(
			attribute.String("subscriber.email", req.Email),
			attribute.String("subscriber.name", req.Name),
		))
	defer span.End()

	s.logger.InfoWithTracing(ctx, "Creating new subscriber", logrus.Fields{
		"email": req.Email,
		"name":  req.Name,
	})

	subscriber := models.NewSubscriber(req.Email, req.Name)

	if err := s.repo.Create(ctx, subscriber); err != nil {
		s.logger.ErrorWithTracing(ctx, "Failed to create subscriber", err, logrus.Fields{
			"subscriber_id": subscriber.ID.String(),
			"email":         req.Email,
		})
		span.RecordError(err)
		return nil, err
	}

	cacheKey := cache.GenerateCacheKey(subscriber.ID)
	if err := s.cache.Set(ctx, cacheKey, subscriber, 5*time.Minute); err != nil {
		s.logger.WarnWithTracing(ctx, "Failed to cache subscriber", logrus.Fields{
			"subscriber_id": subscriber.ID.String(),
			"error":         err.Error(),
		})
	}

	s.logger.InfoWithTracing(ctx, "Successfully created subscriber", logrus.Fields{
		"subscriber_id": subscriber.ID.String(),
		"email":         subscriber.Email,
	})

	span.SetAttributes(
		attribute.String("subscriber.id", subscriber.ID.String()),
		attribute.Bool("success", true),
	)

	return subscriber, nil
}

func (s *SubscriberService) GetSubscriber(ctx context.Context, id uuid.UUID) (*models.Subscriber, error) {
	ctx, span := s.tracer.Start(ctx, "subscriber.service.get",
		trace.WithAttributes(
			attribute.String("subscriber.id", id.String()),
		))
	defer span.End()

	s.logger.InfoWithTracing(ctx, "Retrieving subscriber", logrus.Fields{
		"subscriber_id": id.String(),
	})

	cacheKey := cache.GenerateCacheKey(id)
	if subscriber, err := s.cache.Get(ctx, cacheKey); err == nil {
		s.logger.InfoWithTracing(ctx, "Subscriber found in cache", logrus.Fields{
			"subscriber_id": id.String(),
		})
		span.SetAttributes(
			attribute.Bool("cache.hit", true),
			attribute.Bool("success", true),
		)
		return subscriber, nil
	}

	s.logger.InfoWithTracing(ctx, "Subscriber not in cache, fetching from database", logrus.Fields{
		"subscriber_id": id.String(),
	})

	subscriber, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithTracing(ctx, "Failed to retrieve subscriber", err, logrus.Fields{
			"subscriber_id": id.String(),
		})
		span.RecordError(err)
		return nil, err
	}

	if err := s.cache.Set(ctx, cacheKey, subscriber, 5*time.Minute); err != nil {
		s.logger.WarnWithTracing(ctx, "Failed to cache subscriber", logrus.Fields{
			"subscriber_id": id.String(),
			"error":         err.Error(),
		})
	}

	s.logger.InfoWithTracing(ctx, "Successfully retrieved subscriber", logrus.Fields{
		"subscriber_id": subscriber.ID.String(),
		"email":         subscriber.Email,
	})

	span.SetAttributes(
		attribute.Bool("cache.hit", false),
		attribute.Bool("success", true),
	)

	return subscriber, nil
}

func (s *SubscriberService) GetAllSubscribers(ctx context.Context) ([]*models.Subscriber, error) {
	ctx, span := s.tracer.Start(ctx, "subscriber.service.get_all")
	defer span.End()

	s.logger.InfoWithTracing(ctx, "Retrieving all subscribers", nil)

	subscribers, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.ErrorWithTracing(ctx, "Failed to retrieve subscribers", err, nil)
		span.RecordError(err)
		return nil, err
	}

	s.logger.InfoWithTracing(ctx, "Successfully retrieved all subscribers", logrus.Fields{
		"count": len(subscribers),
	})

	span.SetAttributes(
		attribute.Int("subscriber.count", len(subscribers)),
		attribute.Bool("success", true),
	)

	return subscribers, nil
}

func (s *SubscriberService) UpdateSubscriber(ctx context.Context, id uuid.UUID, req *models.CreateSubscriberRequest) (*models.Subscriber, error) {
	ctx, span := s.tracer.Start(ctx, "subscriber.service.update",
		trace.WithAttributes(
			attribute.String("subscriber.id", id.String()),
		))
	defer span.End()

	s.logger.InfoWithTracing(ctx, "Updating subscriber", logrus.Fields{
		"subscriber_id": id.String(),
		"email":         req.Email,
		"name":          req.Name,
	})

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithTracing(ctx, "Failed to find subscriber for update", err, logrus.Fields{
			"subscriber_id": id.String(),
		})
		span.RecordError(err)
		return nil, err
	}

	existing.Email = req.Email
	existing.Name = req.Name
	existing.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.ErrorWithTracing(ctx, "Failed to update subscriber", err, logrus.Fields{
			"subscriber_id": id.String(),
		})
		span.RecordError(err)
		return nil, err
	}

	cacheKey := cache.GenerateCacheKey(id)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		s.logger.WarnWithTracing(ctx, "Failed to invalidate cache", logrus.Fields{
			"subscriber_id": id.String(),
			"error":         err.Error(),
		})
	}

	s.logger.InfoWithTracing(ctx, "Successfully updated subscriber", logrus.Fields{
		"subscriber_id": existing.ID.String(),
		"email":         existing.Email,
	})

	span.SetAttributes(attribute.Bool("success", true))
	return existing, nil
}

func (s *SubscriberService) DeleteSubscriber(ctx context.Context, id uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "subscriber.service.delete",
		trace.WithAttributes(
			attribute.String("subscriber.id", id.String()),
		))
	defer span.End()

	s.logger.InfoWithTracing(ctx, "Deleting subscriber", logrus.Fields{
		"subscriber_id": id.String(),
	})

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithTracing(ctx, "Failed to delete subscriber", err, logrus.Fields{
			"subscriber_id": id.String(),
		})
		span.RecordError(err)
		return err
	}

	cacheKey := cache.GenerateCacheKey(id)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		s.logger.WarnWithTracing(ctx, "Failed to remove from cache", logrus.Fields{
			"subscriber_id": id.String(),
			"error":         err.Error(),
		})
	}

	s.logger.InfoWithTracing(ctx, "Successfully deleted subscriber", logrus.Fields{
		"subscriber_id": id.String(),
	})

	span.SetAttributes(attribute.Bool("success", true))
	return nil
}