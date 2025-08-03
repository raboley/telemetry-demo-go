package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

type TestSpanRecorder struct {
	mu       sync.RWMutex
	spans    []trace.ReadOnlySpan
}

func NewTestSpanRecorder() *TestSpanRecorder {
	return &TestSpanRecorder{
		spans: make([]trace.ReadOnlySpan, 0),
	}
}

func (t *TestSpanRecorder) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.spans = append(t.spans, spans...)
	return nil
}

func (t *TestSpanRecorder) Shutdown(ctx context.Context) error {
	return nil
}

func (t *TestSpanRecorder) GetSpans() []trace.ReadOnlySpan {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	result := make([]trace.ReadOnlySpan, len(t.spans))
	copy(result, t.spans)
	return result
}

func (t *TestSpanRecorder) GetSpansByName(name string) []trace.ReadOnlySpan {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var result []trace.ReadOnlySpan
	for _, span := range t.spans {
		if span.Name() == name {
			result = append(result, span)
		}
	}
	return result
}

func (t *TestSpanRecorder) GetSpansByOperation(operation string) []trace.ReadOnlySpan {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var result []trace.ReadOnlySpan
	for _, span := range t.spans {
		for _, attr := range span.Attributes() {
			if attr.Key == "operation" && attr.Value.AsString() == operation {
				result = append(result, span)
				break
			}
		}
	}
	return result
}

func (t *TestSpanRecorder) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.spans = make([]trace.ReadOnlySpan, 0)
}

func (t *TestSpanRecorder) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return len(t.spans)
}

func InitTestTracing(serviceName, serviceVersion string, recorder *TestSpanRecorder) (*trace.TracerProvider, error) {
	res := resource.NewWithAttributes(
		resource.Default().SchemaURL(),
	)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(recorder),
		trace.WithResource(res),
	)

	return tp, nil
}