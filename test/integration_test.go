package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	"telemetry-go/internal/cache"
	"telemetry-go/internal/handlers"
	"telemetry-go/internal/logging"
	"telemetry-go/internal/models"
	"telemetry-go/internal/repository"
	"telemetry-go/internal/service"
	"telemetry-go/internal/telemetry"
)

type TestApp struct {
	server    *httptest.Server
	recorder  *telemetry.TestSpanRecorder
	tp        *trace.TracerProvider
	repo      *repository.InMemorySubscriberRepository
	cache     *cache.InMemoryCache
	service   *service.SubscriberService
	handler   *handlers.SubscriberHandler
	logger    *logging.ContextLogger
}

func NewTestApp(t *testing.T) *TestApp {
	gin.SetMode(gin.TestMode)

	logger := logging.NewLogger()
	recorder := telemetry.NewTestSpanRecorder()

	res := resource.NewWithAttributes(
		resource.Default().SchemaURL(),
	)

	tp := trace.NewTracerProvider(
		trace.WithSyncer(recorder),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	repo := repository.NewInMemorySubscriberRepository()
	cacheInstance := cache.NewInMemoryCache()
	subscriberService := service.NewSubscriberService(repo, cacheInstance, logger)
	subscriberHandler := handlers.NewSubscriberHandler(subscriberService, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("test-subscriber-api"))

	api := r.Group("/api/v1")
	{
		subscribers := api.Group("/subscribers")
		{
			subscribers.POST("", subscriberHandler.CreateSubscriber)
			subscribers.GET("", subscriberHandler.GetAllSubscribers)
			subscribers.GET("/:id", subscriberHandler.GetSubscriber)
			subscribers.PUT("/:id", subscriberHandler.UpdateSubscriber)
			subscribers.DELETE("/:id", subscriberHandler.DeleteSubscriber)
		}
	}

	server := httptest.NewServer(r)

	return &TestApp{
		server:   server,
		recorder: recorder,
		tp:       tp,
		repo:     repo,
		cache:    cacheInstance,
		service:  subscriberService,
		handler:  subscriberHandler,
		logger:   logger,
	}
}

func (app *TestApp) Close() {
	app.server.Close()
	_ = app.tp.Shutdown(context.Background())
}

func (app *TestApp) ClearSpans() {
	app.recorder.Clear()
}

func (app *TestApp) GetSpansByOperation(operation string) []trace.ReadOnlySpan {
	return app.recorder.GetSpansByOperation(operation)
}

func (app *TestApp) CreateSubscriber(t *testing.T, email, name string) *models.Subscriber {
	reqBody := models.CreateSubscriberRequest{
		Email: email,
		Name:  name,
	}
	jsonBody, err := json.Marshal(reqBody)
	require.NoError(t, err)

	resp, err := http.Post(
		app.server.URL+"/api/v1/subscribers",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var subscriber models.Subscriber
	err = json.NewDecoder(resp.Body).Decode(&subscriber)
	require.NoError(t, err)

	return &subscriber
}

func (app *TestApp) GetSubscriber(t *testing.T, id string) *models.Subscriber {
	resp, err := http.Get(app.server.URL + "/api/v1/subscribers/" + id)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var subscriber models.Subscriber
	err = json.NewDecoder(resp.Body).Decode(&subscriber)
	require.NoError(t, err)

	return &subscriber
}

func TestSubscriberCacheSpanVerification(t *testing.T) {
	app := NewTestApp(t)
	defer app.Close()

	t.Run("Database spans should be present on cache miss", func(t *testing.T) {
		app.ClearSpans()

		subscriber := app.CreateSubscriber(t, "test@example.com", "Test User")

		time.Sleep(100 * time.Millisecond)

		databaseSpans := app.GetSpansByOperation("database.write")
		assert.GreaterOrEqual(t, len(databaseSpans), 1, 
			"Expected at least one database write span for subscriber creation")

		dbReadSpans := app.GetSpansByOperation("database.read")
		cacheReadSpans := app.GetSpansByOperation("cache.read")
		
		t.Logf("Found %d database write spans, %d database read spans, %d cache read spans", 
			len(databaseSpans), len(dbReadSpans), len(cacheReadSpans))

		// Clear cache to force a database read
		err := app.cache.Clear(context.Background())
		require.NoError(t, err)

		app.ClearSpans()

		_ = app.GetSubscriber(t, subscriber.ID.String())

		time.Sleep(100 * time.Millisecond)

		databaseReadSpansAfterCacheMiss := app.GetSpansByOperation("database.read")
		assert.GreaterOrEqual(t, len(databaseReadSpansAfterCacheMiss), 1,
			"Expected at least one database read span on cache miss")

		t.Logf("After cache miss: Found %d database read spans", len(databaseReadSpansAfterCacheMiss))
	})

	t.Run("Database spans should NOT be present on cache hit", func(t *testing.T) {
		app.ClearSpans()

		subscriber := app.CreateSubscriber(t, "cached@example.com", "Cached User")

		time.Sleep(100 * time.Millisecond)

		app.ClearSpans()

		_ = app.GetSubscriber(t, subscriber.ID.String())

		time.Sleep(100 * time.Millisecond)

		app.ClearSpans()

		_ = app.GetSubscriber(t, subscriber.ID.String())

		time.Sleep(100 * time.Millisecond)

		databaseReadSpans := app.GetSpansByOperation("database.read")
		cacheReadSpans := app.GetSpansByOperation("cache.read")

		assert.Equal(t, 0, len(databaseReadSpans),
			"Expected NO database read spans on cache hit, but found %d", len(databaseReadSpans))
		assert.GreaterOrEqual(t, len(cacheReadSpans), 1,
			"Expected at least one cache read span, but found %d", len(cacheReadSpans))

		t.Logf("Cache hit verification: Found %d database read spans (expected: 0), %d cache read spans", 
			len(databaseReadSpans), len(cacheReadSpans))
	})
}

func TestSubscriberDatabaseSpanVerification(t *testing.T) {
	app := NewTestApp(t)
	defer app.Close()

	t.Run("Verify database write spans during creation", func(t *testing.T) {
		app.ClearSpans()

		_ = app.CreateSubscriber(t, "write@example.com", "Write User")

		time.Sleep(100 * time.Millisecond)

		writeSpans := app.GetSpansByOperation("database.write")
		cacheWriteSpans := app.GetSpansByOperation("cache.write")

		assert.GreaterOrEqual(t, len(writeSpans), 1,
			"Expected at least one database write span during creation")
		assert.GreaterOrEqual(t, len(cacheWriteSpans), 1,
			"Expected at least one cache write span during creation")

		t.Logf("Creation verification: Found %d database write spans, %d cache write spans",
			len(writeSpans), len(cacheWriteSpans))
	})

	t.Run("Verify proper span attributes", func(t *testing.T) {
		app.ClearSpans()

		subscriber := app.CreateSubscriber(t, "attrs@example.com", "Attrs User")

		time.Sleep(100 * time.Millisecond)

		writeSpans := app.GetSpansByOperation("database.write")
		require.GreaterOrEqual(t, len(writeSpans), 1, "Expected at least one database write span")

		span := writeSpans[0]
		foundSubscriberID := false
		foundOperation := false

		for _, attr := range span.Attributes() {
			if attr.Key == "subscriber.id" && attr.Value.AsString() == subscriber.ID.String() {
				foundSubscriberID = true
			}
			if attr.Key == "operation" && attr.Value.AsString() == "database.write" {
				foundOperation = true
			}
		}

		assert.True(t, foundSubscriberID, "Expected to find subscriber.id attribute in span")
		assert.True(t, foundOperation, "Expected to find operation attribute in span")

		t.Logf("Span attributes verification: subscriber.id=%v, operation=%v",
			foundSubscriberID, foundOperation)
	})
}

func TestSubscriberOperationsIntegration(t *testing.T) {
	app := NewTestApp(t)
	defer app.Close()

	t.Run("Full CRUD operations with span verification", func(t *testing.T) {
		app.ClearSpans()

		subscriber := app.CreateSubscriber(t, "crud@example.com", "CRUD User")
		assert.NotEmpty(t, subscriber.ID)
		assert.Equal(t, "crud@example.com", subscriber.Email)
		assert.Equal(t, "CRUD User", subscriber.Name)

		time.Sleep(100 * time.Millisecond)

		createSpans := app.GetSpansByOperation("database.write")
		assert.GreaterOrEqual(t, len(createSpans), 1, "Expected database write spans for creation")

		// Clear cache to force a database read
		err := app.cache.Clear(context.Background())
		require.NoError(t, err)

		app.ClearSpans()

		retrieved := app.GetSubscriber(t, subscriber.ID.String())
		assert.Equal(t, subscriber.ID, retrieved.ID)
		assert.Equal(t, subscriber.Email, retrieved.Email)

		time.Sleep(100 * time.Millisecond)

		readSpans := app.GetSpansByOperation("database.read")
		assert.GreaterOrEqual(t, len(readSpans), 1, "Expected database read spans for retrieval")

		app.ClearSpans()

		_ = app.GetSubscriber(t, subscriber.ID.String())

		time.Sleep(100 * time.Millisecond)

		secondReadSpans := app.GetSpansByOperation("database.read")
		assert.Equal(t, 0, len(secondReadSpans), 
			"Expected NO database read spans on second retrieval (cache hit)")

		cacheHitSpans := app.GetSpansByOperation("cache.read")
		assert.GreaterOrEqual(t, len(cacheHitSpans), 1, 
			"Expected cache read spans on second retrieval")

		t.Logf("CRUD verification complete: create_spans=%d, first_read_spans=%d, cache_hit_db_spans=%d, cache_hit_cache_spans=%d",
			len(createSpans), len(readSpans), len(secondReadSpans), len(cacheHitSpans))
	})
}