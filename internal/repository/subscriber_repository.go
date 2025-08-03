package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"telemetry-go/internal/models"
)

type SubscriberRepository interface {
	Create(ctx context.Context, subscriber *models.Subscriber) error
	GetByID(ctx context.Context, id string) (*models.Subscriber, error)
	GetAll(ctx context.Context) ([]*models.Subscriber, error)
	Update(ctx context.Context, subscriber *models.Subscriber) error
	Delete(ctx context.Context, id string) error
}

type InMemorySubscriberRepository struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]*models.Subscriber
	tracer      trace.Tracer
}

func NewInMemorySubscriberRepository() *InMemorySubscriberRepository {
	return &InMemorySubscriberRepository{
		subscribers: make(map[uuid.UUID]*models.Subscriber),
		tracer:      otel.Tracer("subscriber-repository"),
	}
}

func (r *InMemorySubscriberRepository) Create(ctx context.Context, subscriber *models.Subscriber) error {
	ctx, span := r.tracer.Start(ctx, "subscriber.repository.create",
		trace.WithAttributes(
			attribute.String("subscriber.id", subscriber.ID.String()),
			attribute.String("subscriber.email", subscriber.Email),
			attribute.String("operation", "database.write"),
		))
	defer span.End()

	time.Sleep(10 * time.Millisecond)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.subscribers[subscriber.ID]; exists {
		span.RecordError(fmt.Errorf("subscriber already exists"))
		return fmt.Errorf("subscriber with ID %s already exists", subscriber.ID)
	}

	r.subscribers[subscriber.ID] = subscriber
	span.SetAttributes(attribute.Bool("success", true))
	return nil
}

func (r *InMemorySubscriberRepository) GetByID(ctx context.Context, id string) (*models.Subscriber, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format: %w", err)
	}
	ctx, span := r.tracer.Start(ctx, "subscriber.repository.get_by_id",
		trace.WithAttributes(
			attribute.String("subscriber.id", id),
			attribute.String("operation", "database.read"),
		))
	defer span.End()

	time.Sleep(5 * time.Millisecond)

	r.mu.RLock()
	defer r.mu.RUnlock()

	subscriber, exists := r.subscribers[parsedID]
	if !exists {
		span.RecordError(fmt.Errorf("subscriber not found"))
		return nil, fmt.Errorf("subscriber with ID %s not found", id)
	}

	span.SetAttributes(attribute.Bool("success", true))
	return subscriber, nil
}

func (r *InMemorySubscriberRepository) GetAll(ctx context.Context) ([]*models.Subscriber, error) {
	ctx, span := r.tracer.Start(ctx, "subscriber.repository.get_all",
		trace.WithAttributes(
			attribute.String("operation", "database.read"),
		))
	defer span.End()

	time.Sleep(8 * time.Millisecond)

	r.mu.RLock()
	defer r.mu.RUnlock()

	subscribers := make([]*models.Subscriber, 0, len(r.subscribers))
	for _, subscriber := range r.subscribers {
		subscribers = append(subscribers, subscriber)
	}

	span.SetAttributes(
		attribute.Int("subscriber.count", len(subscribers)),
		attribute.Bool("success", true),
	)
	return subscribers, nil
}

func (r *InMemorySubscriberRepository) Update(ctx context.Context, subscriber *models.Subscriber) error {
	ctx, span := r.tracer.Start(ctx, "subscriber.repository.update",
		trace.WithAttributes(
			attribute.String("subscriber.id", subscriber.ID.String()),
			attribute.String("operation", "database.write"),
		))
	defer span.End()

	time.Sleep(12 * time.Millisecond)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.subscribers[subscriber.ID]; !exists {
		span.RecordError(fmt.Errorf("subscriber not found"))
		return fmt.Errorf("subscriber with ID %s not found", subscriber.ID)
	}

	subscriber.UpdatedAt = time.Now()
	r.subscribers[subscriber.ID] = subscriber
	span.SetAttributes(attribute.Bool("success", true))
	return nil
}

func (r *InMemorySubscriberRepository) Delete(ctx context.Context, id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}
	ctx, span := r.tracer.Start(ctx, "subscriber.repository.delete",
		trace.WithAttributes(
			attribute.String("subscriber.id", id),
			attribute.String("operation", "database.write"),
		))
	defer span.End()

	time.Sleep(7 * time.Millisecond)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.subscribers[parsedID]; !exists {
		span.RecordError(fmt.Errorf("subscriber not found"))
		return fmt.Errorf("subscriber with ID %s not found", id)
	}

	delete(r.subscribers, parsedID)
	span.SetAttributes(attribute.Bool("success", true))
	return nil
}