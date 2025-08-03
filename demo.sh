#!/bin/bash

# Demo script for Telemetry-Go Observable API
# This script demonstrates the API endpoints and observability features

echo "🚀 Starting Telemetry-Go Demo"
echo "=========================================="

# Start the server in the background
echo "📡 Starting server..."
go run cmd/server/main.go &
SERVER_PID=$!

# Wait for server to start
sleep 3

echo ""
echo "📊 Creating subscribers to demonstrate observability..."
echo ""

# Create first subscriber
echo "1️⃣  Creating first subscriber..."
SUBSCRIBER1=$(curl -s -X POST http://localhost:8080/api/v1/subscribers \
  -H "Content-Type: application/json" \
  -d '{"email": "alice@example.com", "name": "Alice Johnson"}')

SUBSCRIBER1_ID=$(echo $SUBSCRIBER1 | jq -r '.id')
echo "   ✅ Created subscriber: $SUBSCRIBER1_ID"

sleep 1

# Create second subscriber
echo ""
echo "2️⃣  Creating second subscriber..."
SUBSCRIBER2=$(curl -s -X POST http://localhost:8080/api/v1/subscribers \
  -H "Content-Type: application/json" \
  -d '{"email": "bob@example.com", "name": "Bob Smith"}')

SUBSCRIBER2_ID=$(echo $SUBSCRIBER2 | jq -r '.id')
echo "   ✅ Created subscriber: $SUBSCRIBER2_ID"

sleep 1

# Get first subscriber (should hit cache)
echo ""
echo "3️⃣  Getting first subscriber (cache hit)..."
curl -s http://localhost:8080/api/v1/subscribers/$SUBSCRIBER1_ID | jq .
echo "   ✅ Retrieved from cache (no database span)"

sleep 1

# Get all subscribers
echo ""
echo "4️⃣  Getting all subscribers..."
curl -s http://localhost:8080/api/v1/subscribers | jq '. | length'
echo "   ✅ Retrieved all subscribers"

sleep 1

# Update subscriber
echo ""
echo "5️⃣  Updating subscriber..."
curl -s -X PUT http://localhost:8080/api/v1/subscribers/$SUBSCRIBER1_ID \
  -H "Content-Type: application/json" \
  -d '{"email": "alice.updated@example.com", "name": "Alice Updated"}' | jq .
echo "   ✅ Updated subscriber"

sleep 1

# Get updated subscriber (should hit database - cache invalidated)
echo ""
echo "6️⃣  Getting updated subscriber (cache miss after update)..."
curl -s http://localhost:8080/api/v1/subscribers/$SUBSCRIBER1_ID | jq .
echo "   ✅ Retrieved from database (cache was invalidated)"

sleep 1

# Health check
echo ""
echo "7️⃣  Health check..."
curl -s http://localhost:8080/health | jq .
echo "   ✅ Server is healthy"

echo ""
echo "🎯 Demo completed! Check the server output above to see:"
echo "   • Structured JSON logs with trace correlation"
echo "   • OpenTelemetry spans showing database vs cache operations"
echo "   • Request/response timing and status codes"
echo ""
echo "📋 Key Observability Features Demonstrated:"
echo "   ✓ Structured logging with trace_id and span_id"
echo "   ✓ Database write spans during creation"
echo "   ✓ Cache hit/miss behavior in logs"
echo "   ✓ Cache invalidation on updates"
echo "   ✓ Request correlation across service layers"
echo ""
echo "🧪 Run the tests to see black box span verification:"
echo "   go test ./test -v"
echo ""

# Clean up
echo "🧹 Stopping server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo "✨ Demo finished!"