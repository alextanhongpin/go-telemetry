package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/global"

	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

var Meter = global.MeterProvider().Meter("app_or_package_name")

func main() {
	counter, err := Meter.SyncInt64().Counter(
		"test.my_counter",
		instrument.WithUnit("1"),
		instrument.WithDescription("just a test counter"),
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	counter.Add(ctx, 1, attribute.String("foo", "bar"))
	counter.Add(ctx, 10, attribute.String("hello", "world"))
	configureOpenTelemetry()
	time.Sleep(10 * time.Second)
	//makeHistogram(ctx)
}

func makeHistogram(ctx context.Context) {
	durRecorder, _ := Meter.SyncInt64().Histogram(
		"some_prefix.histogram",
		instrument.WithUnit("microseconds"),
		instrument.WithDescription("TODO"),
	)

	for {
		dur := time.Duration(rand.NormFloat64()*5_000_000) * time.Microsecond
		durRecorder.Record(ctx, dur.Microseconds())
		time.Sleep(time.Millisecond)
	}
}

func configureOpenTelemetry() {
	exporter := configureMetrics()
	http.HandleFunc("/metrics", exporter.ServeHTTP)
	fmt.Println("listening on http://localhost:8088/metrics")

	go func() {
		_ = http.ListenAndServe(":8088", nil)
	}()
}

func configureMetrics() *prometheus.Exporter {
	config := prometheus.Config{}
	ctrl := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			nil,
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
