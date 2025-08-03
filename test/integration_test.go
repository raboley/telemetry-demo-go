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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	"telemetry-go/internal/app"
	"telemetry-go/internal/logging"
	"telemetry-go/internal/models"
	"telemetry-go/internal/telemetry"
)

type TestApp struct {
	server      *httptest.Server
	recorder    *telemetry.TestSpanRecorder
	tp          *trace.TracerProvider
	application *app.Application
}

func SpawnTestApp(t *testing.T) *TestApp {
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

	config := &app.Config{
		ServiceName:    "test-subscriber-api",
		ServiceVersion: "1.0.0",
		Port:           "0", // Let httptest.Server choose the port
		Logger:         logger,
		TracerProvider: tp,
		GinMode:        gin.TestMode,
	}

	application := app.Build(config)
	server := httptest.NewServer(application.GetRouter())

	return &TestApp{
		server:      server,
		recorder:    recorder,
		tp:          tp,
		application: application,
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

func TestSubscriberCreation(t *testing.T) {
	app := SpawnTestApp(t)
	defer app.Close()

	subscriber := app.CreateSubscriber(t, "test@example.com", "Test User")
	
	assert.NotEmpty(t, subscriber.ID)
	assert.Equal(t, "test@example.com", subscriber.Email)
	assert.Equal(t, "Test User", subscriber.Name)

	time.Sleep(100 * time.Millisecond)

	writeSpans := app.GetSpansByOperation("database.write")
	cacheWriteSpans := app.GetSpansByOperation("cache.write")

	assert.GreaterOrEqual(t, len(writeSpans), 1, "Expected database write spans during creation")
	assert.GreaterOrEqual(t, len(cacheWriteSpans), 1, "Expected cache write spans during creation")
}

func TestSubscriberCacheMiss(t *testing.T) {
	app := SpawnTestApp(t)
	defer app.Close()

	subscriber := app.CreateSubscriber(t, "cachemiss@example.com", "Cache Miss User")
	
	// Clear cache to force database read
	err := app.application.GetCache().Clear(context.Background())
	require.NoError(t, err)

	app.ClearSpans()

	retrieved := app.GetSubscriber(t, subscriber.ID.String())
	assert.Equal(t, subscriber.ID, retrieved.ID)
	assert.Equal(t, subscriber.Email, retrieved.Email)

	time.Sleep(100 * time.Millisecond)

	databaseReadSpans := app.GetSpansByOperation("database.read")
	assert.GreaterOrEqual(t, len(databaseReadSpans), 1, "Expected database read spans on cache miss")
}

func TestSubscriberCacheHit(t *testing.T) {
	app := SpawnTestApp(t)
	defer app.Close()

	subscriber := app.CreateSubscriber(t, "cachehit@example.com", "Cache Hit User")

	app.ClearSpans()

	// First GET - should populate cache
	_ = app.GetSubscriber(t, subscriber.ID.String())

	app.ClearSpans()

	// Second GET - should hit cache
	_ = app.GetSubscriber(t, subscriber.ID.String())

	time.Sleep(100 * time.Millisecond)

	databaseReadSpans := app.GetSpansByOperation("database.read")
	cacheReadSpans := app.GetSpansByOperation("cache.read")

	assert.Equal(t, 0, len(databaseReadSpans), "Expected NO database read spans on cache hit")
	assert.GreaterOrEqual(t, len(cacheReadSpans), 1, "Expected cache read spans on cache hit")
}

func TestSpanAttributes(t *testing.T) {
	app := SpawnTestApp(t)
	defer app.Close()

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
}