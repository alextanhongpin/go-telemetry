// This package demonstrates how to use OpenTelemetry to create and export metrics.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/unit"

	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

var meter = global.MeterProvider().Meter("app_or_package_name")

// Reference:
// https://github.com/open-telemetry/opentelemetry-go/blob/main/example/prometheus/main.go
func main() {
	exporter := configureMetrics()

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", exporter.ServeHTTP)

	ctx := context.Background()

	makeCounter(ctx)
	makeCounterObserver(ctx)
	makeGaugeObserver(ctx)
	makeHistogram(ctx)
	makeUpDownCounter(ctx)
	makeUpDownCounterObserver(ctx)

	newServer(mux, 8088)
}

func newServer(handler http.Handler, port int) {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}

	// Initializing the server in a goroutine so that it won't block the graceful
	// shutdown handling below.
	go func() {

		fmt.Println("listening on http://localhost:8088/metrics")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	// Listen for interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// the context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

// makeCounter demonstrates how to measure non-decreasing numbers, for example,
// number of requests or connections.
func makeCounter(ctx context.Context) {
	counter, err := meter.SyncInt64().Counter(
		"test.my_counter",
		instrument.WithUnit(unit.Dimensionless),
		instrument.WithDescription("just a test counter"),
	)
	if err != nil {
		panic(err)
	}

	counter.Add(ctx, 1, attribute.String("foo", "bar"))
	counter.Add(ctx, 10, attribute.String("hello", "world"))

	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if rand.Float64() < 0.3 {
					counter.Add(ctx, 1, attribute.String("type", "hits"))
				} else {
					counter.Add(ctx, 1, attribute.String("type", "misses"))
				}
			}
		}
	}()
}

// makeUpDownCounter demonstrates how to measure numbers that can go up and
// down, for example, number of goroutines or customers.
func makeUpDownCounter(ctx context.Context) {
	counter, _ := meter.SyncInt64().UpDownCounter(
		"some.prefix.up_down_counter",
		instrument.WithUnit(unit.Dimensionless),
		instrument.WithDescription("TODO"),
	)

	go func() {
		t := time.NewTicker(time.Millisecond)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if rand.Float64() >= 0.5 {
					counter.Add(ctx, +1)
				} else {
					counter.Add(ctx, -1)
				}
			}
		}
	}()
}

// makeHistogram demonstrates how to record a distribution of individual
// values, for example, request or query timings. With this instrument, you get
// total number of records, avg/min/max values, and heatmaps/percentiles.
func makeHistogram(ctx context.Context) {
	durRecorder, _ := meter.SyncInt64().Histogram(
		"some_prefix.histogram",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("TODO"),
	)

	t := time.NewTicker(time.Millisecond)
	defer t.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				dur := time.Duration(rand.NormFloat64()*5_000_000) * time.Microsecond
				durRecorder.Record(ctx, dur.Microseconds())
			}
		}
	}()
}

// counterObserver demonstrates how to measure monotonic (non-decreasing) numbers,
// for example, number of requests or connections.
func makeCounterObserver(ctx context.Context) {
	counter, _ := meter.AsyncInt64().Counter(
		"some.prefix.counter_observer",
		instrument.WithUnit(unit.Dimensionless),
		instrument.WithDescription("TODO"),
	)

	var number int64
	if err := meter.RegisterCallback(
		[]instrument.Asynchronous{counter},
		// SDK periodically calls this function to collect data.

		func(ctx context.Context) {
			number++
			counter.Observe(ctx, number)
		},
	); err != nil {
		panic(err)
	}
}

// makeUpDownCounterObserver demonstrates how to measure numbers that can go up or down,
// for example, number of goroutines or customers.
func makeUpDownCounterObserver(ctx context.Context) {
	counter, err := meter.AsyncInt64().UpDownCounter(
		"some.prefix.up_down_counter_observer",
		instrument.WithUnit(unit.Dimensionless),
		instrument.WithDescription("TODO"),
	)
	if err != nil {
		panic(err)
	}

	if err := meter.RegisterCallback(
		[]instrument.Asynchronous{counter},
		func(ctx context.Context) {
			num := runtime.NumGoroutine()
			counter.Observe(ctx, int64(num))
		},
	); err != nil {
		panic(err)
	}
}

// makeGaugeObserver demonstrates how to measure non-additive numbers that can
// go up and down, for example, cache hit rate or memory utilization.
func makeGaugeObserver(ctx context.Context) {
	gauge, _ := meter.AsyncFloat64().Gauge(
		"some.prefix.gauge_observer",
		instrument.WithUnit(unit.Dimensionless),
		instrument.WithDescription("TODO"),
	)

	if err := meter.RegisterCallback(
		[]instrument.Asynchronous{gauge},
		func(ctx context.Context) {
			gauge.Observe(ctx, rand.Float64())
		},
	); err != nil {
		panic(err)
	}
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
