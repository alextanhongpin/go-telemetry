// https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/examples/otel-collector/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var serviceName = semconv.ServiceNameKey.String("test-service")

func main() {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// The service name used to display traces in backends
			serviceName,
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	shutdownTracerProvider, err := initHTTPTracerProvider(ctx, res)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := shutdownTracerProvider(ctx)
		if err != nil {
			log.Fatalf("failed to shutdown TracerProvider: %s", err)
		}
	}()

	shutdownMeterProvider, err := initHTTPMeterProvider(ctx, res)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := shutdownMeterProvider(ctx)
		if err != nil {
			log.Fatalf("failed to shutdown MeterProvider: %s", err)
		}
	}()

	shutdownLoggerProvider, err := initHTTPLogProvider(ctx, res)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := shutdownLoggerProvider(ctx)
		if err != nil {
			log.Fatalf("failed to shutdown LoggerProvider: %s", err)
		}
	}()

	var (
		name   = "hello-world"
		meter  = otel.Meter(name)
		tracer = otel.Tracer(name)
		logger = otelslog.NewLogger(name)
	)

	// Attributes represent additional key-value descriptors that can be bound
	// to a metric observer or recorder.
	commonAttrs := []attribute.KeyValue{
		attribute.String("attrA", "chocolate"),
		attribute.String("attrB", "raspberry"),
		attribute.String("attrC", "vanilla"),
	}

	runCount, err := meter.Int64Counter("run", metric.WithDescription("The number of times the iteration ran"))
	if err != nil {
		log.Fatal(err)
	}

	for i := range 10 {
		runCount.Add(ctx, 1, metric.WithAttributes(commonAttrs...))
		log.Printf("Doing really hard work (%d / 10)\n", i+1)
	}

	mux := http.NewServeMux()
	h := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx, span := tracer.Start(ctx, "helloHandler")
		defer span.End()

		logger.Info("hello world")
		runCount.Add(ctx, 1, metric.WithAttributes(commonAttrs...))

		span.SetAttributes(attribute.String("foo", "bar"))
		span.SetStatus(codes.Ok, "All is well")

		_, _ = w.Write([]byte("Hello, World!"))
	}
	mux.Handle("/", http.HandlerFunc(h))
	mux.Handle("GET /{name}", http.HandlerFunc(h))

	httpSpanName := func(operation string, r *http.Request) string {
		return fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path)
	}

	// Add HTTP instrumentation for the whole server.
	handler := otelhttp.NewHandler(
		mux,
		"/",
		otelhttp.WithSpanNameFormatter(httpSpanName),
		otelhttp.WithFilter(otelReqFilter),
	)

	log.Println("Server started on port 8000")

	// Instrumeting client.
	// See here: https://uptrace.dev/guides/opentelemetry-net-http
	_ = http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	log.Fatal(http.ListenAndServe(":8000", handler))
}

func otelReqFilter(req *http.Request) bool {
	return req.URL.Path != "/auth"
}
