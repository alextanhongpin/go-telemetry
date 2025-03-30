package main

import (
	"context"
	"math/rand/v2"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	service     = "trace-demo"
	environment = "production"
	id          = 1
)

// https://github.com/open-telemetry/opentelemetry-go/blob/main/example/jaeger/main.go
func main() {
	tp, err := tracerProvider("http://localhost:14268/api/traces")
	if err != nil {
		panic(err)
	}

	// Register our TracerProvider as the global so any imported instrumentation
	// in the future will default to using it.
	otel.SetTracerProvider(tp)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cleanly shutdown and flush telemetry when the application exits.
	defer func(ctx context.Context) {
		// Do not make the application hang when it is shutdown.
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			panic(err)
		}
	}(ctx)

	tr := tp.Tracer("component-main")
	ctx, span := tr.Start(ctx, "foo")
	defer span.End()

	if rand.IntN(2) == 0 {
		span.SetStatus(codes.Error, "An error occurred")
	} else {
		span.SetStatus(codes.Ok, "All is well")
	}
	span.SetStatus(codes.Error, "An error occurred")

	span.SetAttributes(attribute.String("foo", "bar"))

	bar(ctx)
}

func bar(ctx context.Context) {
	// Use the global TracerProvider.
	tr := otel.Tracer("component-bar")
	_, span := tr.Start(ctx, "bar")
	defer span.End()

	span.SetAttributes(
		attribute.Key("testset").String("value"),
		attribute.Key("foo").String("bar"),
	)

	if rand.IntN(2) == 0 {
		span.SetStatus(codes.Error, "An error occurred")
	} else {
		span.SetStatus(codes.Ok, "All is well")
	}
	span.AddEvent("event in bar",
		oteltrace.WithStackTrace(true),
		oteltrace.WithTimestamp(time.Now()),
		oteltrace.WithAttributes(attribute.String("foo", "bar")))
}

// tracerProvider returns an OpenTelemetry TracerProvider, configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// TracerProvider will also use a Resource configured with all the information
// about the application.
func tracerProvider(url string) (*trace.TracerProvider, error) {
	// Create the Jaeger exporter.
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}
	tp := trace.NewTracerProvider(
		// Always be sure to batch in production.
		trace.WithBatcher(exp),
		// Record information about this application in a Resource.
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(service),
			attribute.String("environment", environment),
			attribute.Int64("ID", id),
		)),
	)

	return tp, nil
}
