package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"

	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

// Reference:
// https://github.com/open-telemetry/opentelemetry-go/blob/main/example/prometheus/main.go
func main() {
	exporter := configureMetrics()
	meter := exporter.MeterProvider().Meter("app_or_package_name")

	ctx := context.Background()
	makeCounter(ctx, meter)
	makeHistogram(ctx, meter)

	http.HandleFunc("/metrics", exporter.ServeHTTP)

	fmt.Println("listening on http://localhost:8088/metrics")
	panic(http.ListenAndServe(":8088", nil))
}

func makeCounter(ctx context.Context, meter metric.Meter) {
	counter, err := meter.SyncInt64().Counter(
		"test.my_counter",
		instrument.WithUnit("1"),
		instrument.WithDescription("just a test counter"),
	)
	if err != nil {
		panic(err)
	}

	counter.Add(ctx, 1, attribute.String("foo", "bar"))
	counter.Add(ctx, 10, attribute.String("hello", "world"))
}

func makeHistogram(ctx context.Context, meter metric.Meter) {
	durRecorder, _ := meter.SyncInt64().Histogram(
		"some_prefix.histogram",
		instrument.WithUnit("microseconds"),
		instrument.WithDescription("TODO"),
	)

	go func() {
		for {
			dur := time.Duration(rand.NormFloat64()*5_000_000) * time.Microsecond
			durRecorder.Record(ctx, dur.Microseconds())
			time.Sleep(time.Millisecond)
		}
	}()
}

func configureMetrics() *prometheus.Exporter {
	config := prometheus.Config{}
	ctrl := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			aggregation.CumulativeTemporalitySelector(),
			processor.WithMemory(true),
		),
	)
	exporter, err := prometheus.New(config, ctrl)
	if err != nil {
		panic(err)
	}
	global.SetMeterProvider(exporter.MeterProvider())

	return exporter
}
