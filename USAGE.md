# Telemetry Demo Usage Guide

## Running the API

1. **Start the server:**
   ```bash
   go mod tidy
   go run main.go
   ```

2. **Verify it's running:**
   ```bash
   curl http://localhost:8080/health
   ```

## V0 - Basic Logging Demo

### Start the Application
```bash
go mod tidy
go run main.go
```

### Sample API Calls

**Create a subscriber:**
```bash
curl -X POST http://localhost:8080/v0/subscribers \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}'
```

**Create another subscriber:**
```bash
curl -X POST http://localhost:8080/v0/subscribers \
  -H "Content-Type: application/json" \
  -d '{"name": "Jane Smith", "email": "jane@example.com"}'
```

**Get all subscribers:**
```bash
curl http://localhost:8080/v0/subscribers
```

**Get specific subscriber:**
```bash
curl http://localhost:8080/v0/subscribers/1
```

**Test error handling:**
```bash
# Invalid subscriber ID
curl http://localhost:8080/v0/subscribers/999

# Invalid request body
curl -X POST http://localhost:8080/v0/subscribers \
  -H "Content-Type: application/json" \
  -d '{"name": "No Email"}'
```

## What to Observe in V0

### Console Logs
Watch the console for clean, focused logs showing:
- Request method and endpoint
- User context (name, email, subscriber_id)
- Duration timing
- Success/error status
- **No HTTP noise** - just what matters for your business logic

### Log Fields
Each log entry includes:
- `method`: HTTP method
- `endpoint`: API endpoint called
- `subscriber_id`: User identifier (when available)
- `name` & `email`: User context
- `duration`: Request processing time
- `count`: Number of records (for list operations)

### Log Levels
- `INFO`: Successful operations
- `WARN`: Not found scenarios
- `ERROR`: Validation or processing errors

---

## V1 - Manual Tracing Demo

### Prerequisites
Start both Zipkin and Jaeger to collect traces (run each in separate terminals):
```bash
docker run -d --name zipkin -p 9411:9411 openzipkin/zipkin
```

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest
```

### Start the Application
```bash
go mod tidy
go run main.go
```



### Sample API Calls

**Create subscribers with tracing:**
```bash
curl -X POST http://localhost:8080/v1/subscribers \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice Johnson", "email": "alice@example.com"}'

curl -X POST http://localhost:8080/v1/subscribers \
  -H "Content-Type: application/json" \
  -d '{"name": "Bob Wilson", "email": "bob@example.com"}'
```

**Get all subscribers:**
```bash
curl http://localhost:8080/v1/subscribers
```

**Get specific subscriber:**
```bash
curl http://localhost:8080/v1/subscribers/1
```

**Test error scenarios:**
```bash
# Invalid ID
curl http://localhost:8080/v1/subscribers/999

# Invalid JSON
curl -X POST http://localhost:8080/v1/subscribers \
  -H "Content-Type: application/json" \
  -d '{"name": "No Email Here"}'
```

## What to Observe in V1

### Console Logs
- **Trace IDs**: Every log now includes `trace_id` and `span_id`
- **Same clean format**: Still focused on business data
- **Distributed context**: Trace IDs connect related operations

### View Traces in Both UIs

**Zipkin UI (http://localhost:9411)**
1. **Service Name**: Select "telemetry-demo" from dropdown
2. **Operation Name**: Choose operations like "create_subscriber_request"  
3. **Run Query**: Click "RUN QUERY" to find traces

**Jaeger UI (http://localhost:16686)**
1. **Service**: Select "telemetry-demo" from dropdown
2. **Operation**: Choose operations like "create_subscriber_request"
3. **Find Traces**: Click "Find Traces" button

### Trace Structure
Each request creates a **trace** with multiple **spans**:

**Create Subscriber Trace:**
```
create_subscriber_request (root span)
├── validate_subscriber_data (child span)
└── store_subscriber (child span)
```

**Get Subscriber Trace:**
```
get_subscriber_request (root span)
├── parse_subscriber_id (child span)
└── lookup_subscriber (child span)
```

### Manual Tracing Benefits
- **Explicit control**: You decide what to trace
- **Rich context**: Custom attributes for business logic
- **Error tracking**: Spans capture error states
- **Performance insights**: Child spans show breakdown timing

### Key V1 Concepts Demonstrated
1. **Span Creation**: `tracer.Start()` creates parent/child relationships
2. **Context Propagation**: `ctx` parameter carries trace context
3. **Attributes**: Business data attached to spans
4. **Status Codes**: Success/error states in spans
5. **Span Relationships**: Parent-child hierarchy shows request flow

### Compare Zipkin vs Jaeger UIs
**Same traces, different experiences:**

**Zipkin strengths:**
- **Cleaner UI**: More intuitive trace visualization
- **Better UX**: Easier navigation and filtering  
- **Dependency graphs**: Clear service interaction maps
- **Timeline view**: Excellent span timing visualization

**Jaeger strengths:**
- **Detailed metadata**: More comprehensive span details
- **System architecture**: Better service dependency view
- **Search capabilities**: Advanced filtering options
- **Industry standard**: Widely adopted in enterprises

### Stop Both Services (when done)
```bash
docker stop zipkin jaeger && docker rm zipkin jaeger
```

---

## V1 Technical Deep Dive

### Manual Span Creation Pattern
Looking at V1 code, notice the explicit span management:

```go
// Start root span for entire request
ctx, span := h.tracer.Start(c.Request.Context(), "create_subscriber_request")
defer span.End()

// Child span for specific operation
ctx, validationSpan := h.tracer.Start(ctx, "validate_subscriber_data")
validationSpan.SetAttributes(
    attribute.String("validation.name", req.Name),
    attribute.String("validation.email", req.Email),
)
validationSpan.SetStatus(codes.Ok, "Validation successful")
validationSpan.End()
```

### Context Propagation
The `ctx` parameter carries trace context through the call chain:
```go
// Parent context passed to child
ctx, span := h.tracer.Start(c.Request.Context(), "parent_operation")
ctx, childSpan := h.tracer.Start(ctx, "child_operation")  // Links to parent
```

### Span Lifecycle & Scope
**Span Scope Rules:**
- `tracer.Start()` creates span and returns context
- `defer span.End()` ensures cleanup even with early returns
- Context must be passed to maintain parent-child relationships
- Attributes added during span lifetime become searchable metadata

### Logging Integration
V1 adds trace correlation to logs:
```go
h.logger.WithFields(logrus.Fields{
    "method":         "POST",
    "endpoint":       "/v1/subscribers", 
    "subscriber_id":  subscriber.ID,
    "trace_id":       span.SpanContext().TraceID().String(),  // Links logs to traces
    "span_id":        span.SpanContext().SpanID().String(),   // Pinpoints exact span
}).Info("Subscriber created successfully")
```

**Trace Correlation Benefits:**
- Copy trace ID from log → paste in Zipkin/Jaeger to see full request flow
- Debug issues by following trace ID across services
- Correlate errors in logs with spans that failed

### What V1 Demonstrates Well
✅ **Explicit control** - You decide exactly what gets traced  
✅ **Rich business context** - Custom attributes for domain logic  
✅ **Error handling** - Spans capture success/failure states  
✅ **Performance breakdown** - Child spans show timing per operation  
✅ **Dual export** - Same data viewable in different UIs  

### V1 Pain Points & Why We Need V2

**🔴 Too Much Boilerplate Code**
```go
// 15+ lines of tracing code per endpoint!
ctx, span := h.tracer.Start(c.Request.Context(), "operation")
defer span.End()
span.SetAttributes(...)
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, "message")
}
```
*Business logic gets buried under instrumentation code*

**🔴 Missing HTTP Context**
- No automatic request/response headers in spans
- No HTTP status codes, methods, or routes captured automatically  
- Missing standard OpenTelemetry HTTP semantic conventions

**🔴 Repetitive Patterns**
- Every handler duplicates the same span setup code
- Easy to forget span cleanup or error recording
- Inconsistent attribute naming across endpoints

**🔴 Developer Cognitive Load**
- Must remember to propagate context everywhere
- Manual span lifecycle management prone to leaks
- Focus shifts from business logic to instrumentation

---

## Next Steps

**V2 will solve these problems with middleware:**
- **Automatic HTTP instrumentation** - Request/response data captured automatically
- **Clean business code** - Tracing happens transparently  
- **Standard conventions** - OpenTelemetry HTTP semantic conventions
- **Zero boilerplate** - Just add middleware, everything works

---

## V2 - Middleware Magic ✨

### Same Prerequisites 
```bash
docker run -d --name zipkin -p 9411:9411 openzipkin/zipkin
```

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest
```

### Start the Application
```bash
go mod tidy
go run main.go
```

### Sample API Calls

**Create subscribers with automatic tracing:**
```bash
curl -X POST http://localhost:8080/v2/subscribers \
  -H "Content-Type: application/json" \
  -d '{"name": "Charlie Brown", "email": "charlie@example.com"}'

curl -X POST http://localhost:8080/v2/subscribers \
  -H "Content-Type: application/json" \
  -d '{"name": "Lucy Van Pelt", "email": "lucy@example.com"}'
```

**Get all subscribers:**
```bash
curl http://localhost:8080/v2/subscribers
```

**Get specific subscriber:**
```bash
curl http://localhost:8080/v2/subscribers/1
```

**Test error scenarios:**
```bash
# Invalid ID
curl http://localhost:8080/v2/subscribers/999

# Invalid JSON
curl -X POST http://localhost:8080/v2/subscribers \
  -H "Content-Type: application/json" \
  -d '{"name": "Missing Email"}'
```

## What V2 Demonstrates

### Clean Handler Code
Compare V2 vs V1 - same functionality, much cleaner:

**V1 (Manual):**
```go
// 15+ lines of tracing boilerplate per endpoint
ctx, span := h.tracer.Start(c.Request.Context(), "create_subscriber_request")
defer span.End()
span.SetAttributes(attribute.String("http.method", "POST"))
// ... more setup code
```

**V2 (Middleware):**
```go
// Just get the span that middleware created!
span := trace.SpanFromContext(c.Request.Context())
// Add business context only
span.SetAttributes(attribute.String("user.name", req.Name))
```

### Automatic HTTP Instrumentation
The `otelgin.Middleware` automatically captures:
- ✅ **HTTP method, route, status code**
- ✅ **Request/response headers** 
- ✅ **Request duration**
- ✅ **Error states**
- ✅ **Standard OpenTelemetry semantic conventions**

### Isolated Middleware
V2 uses a **separate router** with middleware:
```go
// V2 router with middleware (isolated!)
v2Router := routes.CreateV2Router(memStore)
v2Router.Use(otelgin.Middleware("telemetry-demo"))
```

**Isolation Benefits:**
- V0/V1 completely unaffected by middleware
- Can mix instrumentation approaches in same app
- Perfect for gradual migration strategies

### Business Logic Focus
V2 handlers focus on **business logic only:**
- HTTP context handled by middleware
- Custom spans only for business operations  
- Cleaner, more maintainable code
- Easy to read and understand

### Same Rich Observability
Despite cleaner code, V2 provides **richer traces**:
- All V1 custom spans + automatic HTTP instrumentation
- Standard semantic conventions for better tooling
- Consistent span naming across all endpoints

---

## V0 vs V1 vs V2 Comparison

| Aspect | V0 | V1 | V2 |
|--------|----|----|----| 
| **Observability** | Logs only | Manual traces | Auto + custom traces |
| **Code cleanliness** | ✅ Clean | ❌ Lots of boilerplate | ✅ Clean |
| **HTTP context** | ❌ Manual | ❌ Manual | ✅ Automatic |
| **Business spans** | ❌ None | ✅ Full control | ✅ When needed |
| **Standard conventions** | ❌ None | ❌ Manual | ✅ Built-in |
| **Maintenance** | ✅ Easy | ❌ High overhead | ✅ Easy |

**The Evolution:**
- **V0**: Understanding the basics
- **V1**: Learning how tracing works under the hood  
- **V2**: Production-ready observability with minimal effort

Each version builds on the previous, showing the evolution of observability!