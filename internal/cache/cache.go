package cache

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

type Cache interface {
	Get(ctx context.Context, key string) (*models.Subscriber, error)
	Set(ctx context.Context, key string, subscriber *models.Subscriber, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

type cacheItem struct {
	subscriber *models.Subscriber
	expireAt   time.Time
}

type InMemoryCache struct {
	mu     sync.RWMutex
	items  map[string]*cacheItem
	tracer trace.Tracer
}

func NewInMemoryCache() *InMemoryCache {
	cache := &InMemoryCache{
		items:  make(map[string]*cacheItem),
		tracer: otel.Tracer("cache"),
	}
	
	go cache.cleanup()
	return cache
}

func (c *InMemoryCache) Get(ctx context.Context, key string) (*models.Subscriber, error) {
	ctx, span := c.tracer.Start(ctx, "cache.get",
		trace.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("operation", "cache.read"),
		))
	defer span.End()

	time.Sleep(1 * time.Millisecond)

	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		span.SetAttributes(
			attribute.Bool("cache.hit", false),
			attribute.String("cache.result", "miss"),
		)
		return nil, fmt.Errorf("key not found in cache")
	}

	if time.Now().After(item.expireAt) {
		span.SetAttributes(
			attribute.Bool("cache.hit", false),
			attribute.String("cache.result", "expired"),
		)
		return nil, fmt.Errorf("key expired in cache")
	}

	span.SetAttributes(
		attribute.Bool("cache.hit", true),
		attribute.String("cache.result", "hit"),
		attribute.String("subscriber.id", item.subscriber.ID.String()),
	)
	return item.subscriber, nil
}

func (c *InMemoryCache) Set(ctx context.Context, key string, subscriber *models.Subscriber, ttl time.Duration) error {
	ctx, span := c.tracer.Start(ctx, "cache.set",
		trace.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("subscriber.id", subscriber.ID.String()),
			attribute.String("operation", "cache.write"),
			attribute.String("ttl", ttl.String()),
		))
	defer span.End()

	time.Sleep(1 * time.Millisecond)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		subscriber: subscriber,
		expireAt:   time.Now().Add(ttl),
	}

	span.SetAttributes(attribute.Bool("success", true))
	return nil
}

func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	ctx, span := c.tracer.Start(ctx, "cache.delete",
		trace.WithAttributes(
			attribute.String("cache.key", key),
			attribute.String("operation", "cache.write"),
		))
	defer span.End()

	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.items[key]
	delete(c.items, key)

	span.SetAttributes(
		attribute.Bool("key.existed", exists),
		attribute.Bool("success", true),
	)
	return nil
}

func (c *InMemoryCache) Clear(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "cache.clear",
		trace.WithAttributes(
			attribute.String("operation", "cache.write"),
		))
	defer span.End()

	c.mu.Lock()
	defer c.mu.Unlock()

	itemCount := len(c.items)
	c.items = make(map[string]*cacheItem)

	span.SetAttributes(
		attribute.Int("items.cleared", itemCount),
		attribute.Bool("success", true),
	)
	return nil
}

func (c *InMemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expireAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

func GenerateCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("subscriber:%s", id.String())
}