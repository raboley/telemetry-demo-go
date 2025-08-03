package repository

import (
	"context"
	"encoding/json"
	"fmt"

	dapr "github.com/dapr/go-sdk/client"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"telemetry-go/internal/models"
)

type DaprSubscriberRepository struct {
	client    dapr.Client
	tracer    trace.Tracer
	storeName string
}

func NewDaprSubscriberRepository(client dapr.Client, storeName string) *DaprSubscriberRepository {
	return &DaprSubscriberRepository{
		client:    client,
		tracer:    otel.Tracer("dapr.repository"),
		storeName: storeName,
	}
}

func (r *DaprSubscriberRepository) Create(ctx context.Context, subscriber *models.Subscriber) error {
	ctx, span := r.tracer.Start(ctx, "subscriber.repository.create",
		trace.WithAttributes(
			attribute.String("subscriber.id", subscriber.ID.String()),
			attribute.String("subscriber.email", subscriber.Email),
			attribute.String("operation", "database.write"),
			attribute.String("dapr.store", r.storeName),
		))
	defer span.End()

	data, err := json.Marshal(subscriber)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal subscriber: %w", err)
	}

	err = r.client.SaveState(ctx, r.storeName, subscriber.ID.String(), data, nil)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to save subscriber to dapr state store: %w", err)
	}

	span.SetAttributes(attribute.Bool("success", true))
	return nil
}

func (r *DaprSubscriberRepository) GetByID(ctx context.Context, id string) (*models.Subscriber, error) {
	ctx, span := r.tracer.Start(ctx, "subscriber.repository.get_by_id",
		trace.WithAttributes(
			attribute.String("subscriber.id", id),
			attribute.String("operation", "database.read"),
			attribute.String("dapr.store", r.storeName),
		))
	defer span.End()

	item, err := r.client.GetState(ctx, r.storeName, id, nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get subscriber from dapr state store: %w", err)
	}

	if len(item.Value) == 0 {
		span.SetAttributes(attribute.Bool("found", false))
		return nil, models.ErrSubscriberNotFound
	}

	var subscriber models.Subscriber
	err = json.Unmarshal(item.Value, &subscriber)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to unmarshal subscriber: %w", err)
	}

	span.SetAttributes(
		attribute.Bool("success", true),
		attribute.Bool("found", true),
		attribute.String("subscriber.email", subscriber.Email),
	)
	return &subscriber, nil
}

func (r *DaprSubscriberRepository) GetAll(ctx context.Context) ([]*models.Subscriber, error) {
	ctx, span := r.tracer.Start(ctx, "subscriber.repository.get_all",
		trace.WithAttributes(
			attribute.String("operation", "database.read"),
			attribute.String("dapr.store", r.storeName),
		))
	defer span.End()

	// Note: Dapr state store doesn't have a built-in "get all" operation
	// In a real implementation, you might use a secondary index or query API
	// For this example, we'll return an error indicating this limitation
	span.SetAttributes(attribute.String("error", "get_all not supported by dapr state store"))
	return nil, fmt.Errorf("GetAll operation not supported by Dapr state store - use a database with query capabilities for this operation")
}

func (r *DaprSubscriberRepository) Update(ctx context.Context, subscriber *models.Subscriber) error {
	ctx, span := r.tracer.Start(ctx, "subscriber.repository.update",
		trace.WithAttributes(
			attribute.String("subscriber.id", subscriber.ID.String()),
			attribute.String("subscriber.email", subscriber.Email),
			attribute.String("operation", "database.write"),
			attribute.String("dapr.store", r.storeName),
		))
	defer span.End()

	// Check if subscriber exists first
	existing, err := r.GetByID(ctx, subscriber.ID.String())
	if err != nil {
		span.RecordError(err)
		return err
	}
	if existing == nil {
		span.SetAttributes(attribute.Bool("found", false))
		return models.ErrSubscriberNotFound
	}

	data, err := json.Marshal(subscriber)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal subscriber: %w", err)
	}

	err = r.client.SaveState(ctx, r.storeName, subscriber.ID.String(), data, nil)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update subscriber in dapr state store: %w", err)
	}

	span.SetAttributes(attribute.Bool("success", true))
	return nil
}

func (r *DaprSubscriberRepository) Delete(ctx context.Context, id string) error {
	ctx, span := r.tracer.Start(ctx, "subscriber.repository.delete",
		trace.WithAttributes(
			attribute.String("subscriber.id", id),
			attribute.String("operation", "database.delete"),
			attribute.String("dapr.store", r.storeName),
		))
	defer span.End()

	// Check if subscriber exists first
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if existing == nil {
		span.SetAttributes(attribute.Bool("found", false))
		return models.ErrSubscriberNotFound
	}

	err = r.client.DeleteState(ctx, r.storeName, id, nil)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete subscriber from dapr state store: %w", err)
	}

	span.SetAttributes(attribute.Bool("success", true))
	return nil
}