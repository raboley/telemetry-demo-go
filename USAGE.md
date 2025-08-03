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

## Next Steps

After exploring V0, you'll implement:
- **V1**: Manual tracing and spans
- **V2**: OpenTelemetry middleware magic

Each version builds on the previous, showing the evolution of observability!