package telemetry

import (
	"context"
	"log"
	
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

const serviceName = "telemetry-demo"

func InitTracer() func() {
	// Create Zipkin exporter
	zipkinExporter, err := zipkin.New("http://localhost:9411/api/v2/spans")
	if err != nil {
		log.Printf("Failed to create Zipkin exporter: %v", err)
	}
	
	// Create Jaeger exporter
	jaegerExporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	if err != nil {
		log.Printf("Failed to create Jaeger exporter: %v", err)
	}
	
	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("v1.0.0"),
		),
	)
	if err != nil {
		log.Printf("Failed to create resource: %v", err)
		return func() {}
	}
	
	// Create trace provider with multiple exporters
	var options []trace.TracerProviderOption
	options = append(options, trace.WithResource(res))
	
	if zipkinExporter != nil {
		options = append(options, trace.WithBatcher(zipkinExporter))
		log.Println("📡 Zipkin exporter configured - traces at http://localhost:9411")
	}
	
	if jaegerExporter != nil {
		options = append(options, trace.WithBatcher(jaegerExporter))
		log.Println("📡 Jaeger exporter configured - traces at http://localhost:16686")
	}
	
	tp := trace.NewTracerProvider(options...)
	
	// Set global trace provider
	otel.SetTracerProvider(tp)
	
	log.Println("🚀 Dual tracing enabled - same traces visible in both UIs!")
	
	// Return cleanup function
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer: %v", err)
		}
	}
}