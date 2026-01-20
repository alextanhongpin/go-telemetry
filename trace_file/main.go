package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const name = "test"

func main() {
	l := log.New(os.Stdout, "", 0)
	// Write telemetry data to a file.
	f, err := os.Create("traces.json")
	if err != nil {
		l.Fatal(err)
	}
	defer f.Close()

	tp, err := newStdoutTraceProvider(f, newResource())
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			l.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)

	ctx := context.Background()
	newCtx, span := otel.Tracer(name).Start(ctx, "main")
	defer span.End()
	foo(newCtx)
}

func foo(ctx context.Context) error {
	_, span := otel.Tracer(name).Start(ctx, "foo")
	defer span.End()

	span.SetAttributes(attribute.String("foo.msg", "foo"))
	err := errors.New("foo: error")
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	fmt.Println("foo")
	return nil
}

// newResource returns a resource describing this application.
func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("fib"),
			semconv.ServiceVersionKey.String("v0.1.0"),
			semconv.ServiceInstanceIDKey.String("abcdef123456"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}

func newStdoutTraceProvider(w io.Writer, res *resource.Resource) (*trace.TracerProvider, error) {
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human-readable output.
		stdouttrace.WithPrettyPrint(),
		// Do not print timestamps for the demo.
		stdouttrace.WithoutTimestamps(),
	)
	if err != nil {
		return nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()),
	)

	return tracerProvider, nil
}
