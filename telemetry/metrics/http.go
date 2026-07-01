package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	httpInstrumentMu     sync.RWMutex
	httpRequestDuration  metric.Float64Histogram
	httpActiveRequests   metric.Int64UpDownCounter
	httpInstrumentsReady bool
)

// InitHTTPServerInstruments initializes standard HTTP server metric instruments.
func InitHTTPServerInstruments() error {
	return initHTTPServerInstruments(Meter())
}

func initHTTPServerInstruments(m metric.Meter) error {
	httpInstrumentMu.Lock()
	defer httpInstrumentMu.Unlock()

	if httpInstrumentsReady {
		return nil
	}

	duration, err := m.Float64Histogram(
		MetricHTTPServerRequestDuration,
		metric.WithDescription("HTTP server request duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("create %s: %w", MetricHTTPServerRequestDuration, err)
	}

	active, err := m.Int64UpDownCounter(
		MetricHTTPServerActiveRequests,
		metric.WithDescription("Number of active HTTP server requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return fmt.Errorf("create %s: %w", MetricHTTPServerActiveRequests, err)
	}

	httpRequestDuration = duration
	httpActiveRequests = active
	httpInstrumentsReady = true
	return nil
}

// AddHTTPServerActiveRequest records active HTTP request delta.
func AddHTTPServerActiveRequest(ctx context.Context, delta int64, attrs ...attribute.KeyValue) error {
	if err := InitHTTPServerInstruments(); err != nil {
		return err
	}

	httpInstrumentMu.RLock()
	active := httpActiveRequests
	httpInstrumentMu.RUnlock()

	active.Add(ctx, delta, metric.WithAttributes(attrs...))
	return nil
}

// RecordHTTPServerRequest records HTTP server request duration in seconds.
func RecordHTTPServerRequest(ctx context.Context, duration time.Duration, attrs ...attribute.KeyValue) error {
	if err := InitHTTPServerInstruments(); err != nil {
		return err
	}

	httpInstrumentMu.RLock()
	histogram := httpRequestDuration
	httpInstrumentMu.RUnlock()

	histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	return nil
}

func resetHTTPInstruments() {
	httpInstrumentMu.Lock()
	defer httpInstrumentMu.Unlock()

	httpRequestDuration = nil
	httpActiveRequests = nil
	httpInstrumentsReady = false
}
