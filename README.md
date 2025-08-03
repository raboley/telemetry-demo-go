# Telemetry-Go: Observable API Example

A comprehensive example of building an observable Go API using OpenTelemetry, structured logging, and distributed tracing. This project demonstrates how to implement proper observability practices in a web API with caching and database operations.

## 🎯 Purpose

This example API is designed to teach observability concepts to development teams:

- **Distributed Tracing** - Track requests across service boundaries
- **Structured Logging** - Consistent, searchable log format with trace correlation
- **Span Analysis** - Understanding when operations hit cache vs database
- **Black Box Testing** - Verify telemetry behavior without inspecting internals

## 🏗️ Architecture

### In-Memory Version
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │───▶│   Gin Router    │───▶│    Handlers     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │     Cache       │◀───│    Service      │
                       │  (In-Memory)    │    │     Layer       │
                       └─────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
                                              ┌─────────────────┐
                                              │   Repository    │
                                              │  (In-Memory)    │
                                              └─────────────────┘
```

### Dapr Version
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │───▶│   Gin Router    │───▶│    Handlers     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │     Cache       │◀───│    Service      │
                       │  (In-Memory)    │    │     Layer       │
                       └─────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
                                              ┌─────────────────┐
                                              │ Dapr Repository │───┐
                                              └─────────────────┘   │
                                                        │           │
                                                        ▼           ▼
                                              ┌─────────────────┐   ┌─────────────────┐
                                              │  Dapr Sidecar   │   │   Dapr State    │
                                              │   (HTTP API)    │───│  Store (Redis/  │
                                              └─────────────────┘   │ In-Memory/etc.) │
                                                                    └─────────────────┘
```

## 📊 Observability Features

### 1. Distributed Tracing with OpenTelemetry

**What are Traces?**
Traces show the journey of a request through your system. Each trace contains multiple spans representing different operations.

**What are Spans?**
Spans represent individual operations within a trace. They have:
- Start and end times
- Attributes (key-value metadata)
- Status (success/error)
- Parent-child relationships

**Example Trace Flow:**
```
HTTP Request
├── subscriber.handler.create (HTTP Handler)
    ├── subscriber.service.create (Business Logic)
        ├── subscriber.repository.create (Database Write)
        └── cache.set (Cache Write)
```

### 2. Structured Logging

**Benefits of Structured Logs:**
- Consistent format (JSON)
- Searchable fields
- Trace correlation via trace_id and span_id
- Machine-readable

**Example Log Entry:**
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "message": "Successfully created subscriber",
  "trace_id": "abc123...",
  "span_id": "def456...",
  "subscriber_id": "uuid-here",
  "email": "user@example.com",
  "endpoint": "POST /subscribers"
}
```

### 3. Span Correlation for Cache vs Database

The API demonstrates a critical observability pattern: **distinguishing between cache hits and database queries**.

**Cache Miss Flow:**
```
GET /subscribers/{id}
├── cache.get (cache miss)
├── subscriber.repository.get_by_id (database read)
└── cache.set (update cache)
```

**Cache Hit Flow:**
```
GET /subscribers/{id}
└── cache.get (cache hit) ← No database span!
```

## 🚀 Running the Application

### Prerequisites

- Go 1.23+
- Git
- [Dapr CLI](https://docs.dapr.io/getting-started/install-dapr-cli/) (for Dapr version)

### Setup

1. **Clone and initialize:**
```bash
git clone <repository-url>
cd telemetry-go
go mod tidy
```

### Running Options

#### Option 1: In-Memory Version (Original)
```bash
go run cmd/server/main.go
```

#### Option 2: Dapr Version (Distributed State Store)
```bash
# Initialize Dapr (first time only)
dapr init

# Run with Dapr
dapr run --app-id subscriber-api --app-port 8080 --dapr-http-port 3500 --config .dapr/config.yaml --components-path .dapr/components -- go run cmd/dapr/main.go
```

Both versions start on `http://localhost:8080`

### Available Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/subscribers` | Create subscriber |
| GET | `/api/v1/subscribers` | List all subscribers |
| GET | `/api/v1/subscribers/{id}` | Get subscriber by ID |
| PUT | `/api/v1/subscribers/{id}` | Update subscriber |
| DELETE | `/api/v1/subscribers/{id}` | Delete subscriber |
| GET | `/health` | Health check |

### Example API Usage

**Create a subscriber and capture the ID:**
```bash
# Create subscriber and extract the ID for subsequent calls
USER_ID=$(curl -X POST http://localhost:8080/api/v1/subscribers \
  -H "Content-Type: application/json" \
  -d '{"email": "john@example.com", "name": "John Doe"}' \
  -s | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

echo "Created user with ID: $USER_ID"
```

**Get subscriber (first call - cache miss):**
```bash
curl http://localhost:8080/api/v1/subscribers/$USER_ID
```

**Get subscriber again (cache hit):**
```bash
curl http://localhost:8080/api/v1/subscribers/$USER_ID
```

**Complete workflow (copy and paste to run all at once):**
```bash
# Start with a clean slate - create subscriber and capture ID
USER_ID=$(curl -X POST http://localhost:8080/api/v1/subscribers \
  -H "Content-Type: application/json" \
  -d '{"email": "demo@example.com", "name": "Demo User"}' \
  -s | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

echo "Created user with ID: $USER_ID"

# First GET - cache miss (will hit database)
echo "First GET (cache miss):"
curl http://localhost:8080/api/v1/subscribers/$USER_ID

# Second GET - cache hit (no database access)
echo -e "\nSecond GET (cache hit):"
curl http://localhost:8080/api/v1/subscribers/$USER_ID

# List all subscribers
echo -e "\nList all subscribers:"
curl http://localhost:8080/api/v1/subscribers
```

## 🧪 Testing Observability

### Running Tests

```bash
go test ./test -v
```

### Test Strategy

Each test spawns a fresh application instance to ensure isolation. The tests verify:

1. **TestSubscriberCreation:** Verifies subscriber creation and database/cache write spans
2. **TestSubscriberCacheMiss:** Verifies database spans are present when cache is cleared
3. **TestSubscriberCacheHit:** Verifies database spans are ABSENT when cache hits
4. **TestSpanAttributes:** Verifies proper metadata in spans

### Key Test: Cache Hit vs Cache Miss

```go
// TestSubscriberCacheHit: Verifies NO database access on cache hit
func TestSubscriberCacheHit(t *testing.T) {
    app := SpawnTestApp(t)  // Fresh app instance
    defer app.Close()

    subscriber := app.CreateSubscriber(t, "cachehit@example.com", "Cache Hit User")
    app.ClearSpans()

    // First GET - populates cache
    _ = app.GetSubscriber(t, subscriber.ID.String())
    app.ClearSpans()

    // Second GET - should hit cache only
    _ = app.GetSubscriber(t, subscriber.ID.String())

    databaseReadSpans := app.GetSpansByOperation("database.read")
    cacheReadSpans := app.GetSpansByOperation("cache.read")

    assert.Equal(t, 0, len(databaseReadSpans))        // NO database access!
    assert.GreaterOrEqual(t, len(cacheReadSpans), 1)  // Cache accessed!
}
```

## 📝 Understanding the Observability Output

### Trace Output Example

When you run the application, you'll see trace output like:

```json
{
  "Name": "subscriber.handler.create",
  "SpanContext": {
    "TraceID": "abc123...",
    "SpanID": "def456..."
  },
  "Parent": {
    "TraceID": "abc123...",
    "SpanID": "parent456..."
  },
  "Attributes": [
    {
      "Key": "subscriber.email",
      "Value": {
        "Type": "STRING",
        "Value": "john@example.com"
      }
    },
    {
      "Key": "operation",
      "Value": {
        "Type": "STRING", 
        "Value": "database.write"
      }
    }
  ]
}
```

### Log Output Example

Structured logs with trace correlation:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "message": "Successfully created subscriber",
  "trace_id": "abc123def456...",
  "span_id": "def456ghi789...",
  "subscriber_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "john@example.com",
  "endpoint": "POST /subscribers"
}
```

## 🎓 Learning Objectives

After exploring this example, you should understand:

### 1. **Trace Correlation**
- How to connect logs to traces using trace_id and span_id
- How to follow a request through multiple service layers
- How parent-child span relationships work

### 2. **Performance Insights**
- Identifying slow database operations vs fast cache hits
- Understanding operation timing through span duration
- Spotting performance bottlenecks in traces

### 3. **Debugging Production Issues**
- Using trace IDs to find all related logs
- Understanding error propagation through spans
- Correlating frontend errors with backend operations

### 4. **Testing Observability**
- Writing tests that verify telemetry behavior
- Black box testing of spans and metrics
- Ensuring observability doesn't break over time

## 🔧 Implementation Details

### Span Creation Pattern

```go
func (r *Repository) Create(ctx context.Context, subscriber *Subscriber) error {
    ctx, span := r.tracer.Start(ctx, "subscriber.repository.create",
        trace.WithAttributes(
            attribute.String("subscriber.id", subscriber.ID.String()),
            attribute.String("operation", "database.write"),
        ))
    defer span.End()
    
    // Actual work here...
    
    if err != nil {
        span.RecordError(err)
        return err
    }
    
    span.SetAttributes(attribute.Bool("success", true))
    return nil
}
```

### Structured Logging Pattern

```go
func (l *ContextLogger) InfoWithTracing(ctx context.Context, msg string, fields logrus.Fields) {
    entry := l.WithContext(ctx)
    
    // Extract trace information from context
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        spanCtx := span.SpanContext()
        entry = entry.WithFields(logrus.Fields{
            "trace_id": spanCtx.TraceID().String(),
            "span_id":  spanCtx.SpanID().String(),
        })
    }
    
    if fields != nil {
        entry = entry.WithFields(fields)
    }
    entry.Info(msg)
}
```

## 🔧 Dapr Integration Features

### State Store Operations
The Dapr version demonstrates additional observability patterns:

- **Distributed State Management:** Uses Dapr state store instead of in-memory storage
- **External HTTP Calls:** Traces calls to Dapr sidecar API
- **Configuration Management:** Shows how to configure Dapr components
- **Service Discovery:** Demonstrates microservice communication patterns

### Dapr Span Analysis
When running the Dapr version, you'll see additional spans:
```
HTTP Request
├── subscriber.handler.create (HTTP Handler)
    ├── subscriber.service.create (Business Logic)
        ├── subscriber.repository.create (Dapr Repository)
            └── HTTP POST to Dapr Sidecar (External Service Call)
        └── cache.set (Cache Write)
```

### Switching Between Implementations
The application demonstrates the **Repository Pattern** with dependency injection:
- **In-Memory:** `cmd/server/main.go` - Uses in-memory repository
- **Dapr:** `cmd/dapr/main.go` - Uses Dapr state store repository
- **Same Business Logic:** Both use identical service and handler layers

## 🚀 Next Steps

To extend this example:

1. **Add Metrics:** Implement Prometheus metrics for request counts, durations, error rates
2. **External Services:** Add HTTP client calls to other microservices with trace propagation
3. **Real Database:** Replace Dapr in-memory state store with Redis/PostgreSQL
4. **Message Queues:** Add async processing with Dapr pub/sub and trace context propagation
5. **Error Handling:** Implement comprehensive error tracking and alerting

## 📚 Resources

- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [Dapr Documentation](https://docs.dapr.io/)
- [Dapr Go SDK](https://github.com/dapr/go-sdk)
- [Structured Logging Best Practices](https://blog.treasuredata.com/post/the-power-of-structured-logging/)
- [Distributed Tracing Concepts](https://opentelemetry.io/docs/concepts/observability-primer/)

## 🤝 Contributing

This is an educational example. Feel free to:
- Add more observability features
- Improve test coverage
- Add documentation
- Create additional examples

---

Happy observing! 🔍📊