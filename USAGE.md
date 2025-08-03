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

### Stop Both telemetry clients Services (when done)
```bash
docker stop jaeger && docker rm jaeger
```

```shell
docker stop zipkin && docker rm zipkin
```

---

## Next Steps

After exploring V1, you'll implement:
- **V2**: OpenTelemetry middleware magic (automatic instrumentation)

Each version builds on the previous, showing the evolution of observability!