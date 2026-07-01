package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNormalizeProtocol(t *testing.T) {
	assert.Equal(t, "grpc", normalizeProtocol(""))
	assert.Equal(t, "grpc", normalizeProtocol("GRPC"))
	assert.Equal(t, "grpc", normalizeProtocol("unknown"))
	assert.Equal(t, "http", normalizeProtocol(" http "))
}

func TestNormalizeGRPCEndpoint(t *testing.T) {
	assert.Equal(t, "localhost:4317", normalizeGRPCEndpoint(""))
	assert.Equal(t, "collector:4317", normalizeGRPCEndpoint("grpc://collector:4317"))
	assert.Equal(t, "collector:4317", normalizeGRPCEndpoint("http://collector:4317"))
	assert.Equal(t, "collector:4317", normalizeGRPCEndpoint("https://collector:4317"))
}

func TestNormalizeHTTPEndpoint(t *testing.T) {
	assert.Equal(t, "localhost:4318", normalizeHTTPEndpoint(""))
	assert.Equal(t, "collector:4318", normalizeHTTPEndpoint("http://collector:4318"))
	assert.Equal(t, "collector:4318", normalizeHTTPEndpoint("https://collector:4318"))
}

func TestExportDefaults(t *testing.T) {
	assert.Equal(t, 15*time.Second, exportInterval(0))
	assert.Equal(t, 15*time.Second, exportInterval(-1))
	assert.Equal(t, 30*time.Second, exportInterval(30))

	assert.Equal(t, 10*time.Second, exportTimeout(0))
	assert.Equal(t, 10*time.Second, exportTimeout(-1))
	assert.Equal(t, 5*time.Second, exportTimeout(5))
}

func TestInitDisabledUsesNoop(t *testing.T) {
	require.NoError(t, Init(MetricsConfig{Enabled: false}, "svc", "1.0.0", "test"))
	t.Cleanup(func() {
		require.NoError(t, Close(context.Background()))
	})

	require.NotNil(t, MeterProvider())
	counter, err := Meter().Int64Counter("test.noop.counter")
	require.NoError(t, err)
	counter.Add(context.Background(), 1)
}

func TestManualReaderCollectsCounter(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := provider.Meter("test")

	counter, err := meter.Int64Counter(
		"test.counter",
		metric.WithDescription("test counter"),
		metric.WithUnit("{count}"),
	)
	require.NoError(t, err)

	counter.Add(context.Background(), 3, metric.WithAttributes(attribute.String("kind", "unit")))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	found, value := findInt64Sum(rm, "test.counter")
	require.True(t, found)
	assert.Equal(t, int64(3), value)
}

func TestHTTPServerInstrumentsCollectDurationAndActiveRequests(t *testing.T) {
	reader := useTestMeterProvider(t)

	attrs := []attribute.KeyValue{
		attribute.String(AttrHTTPRequestMethod, "GET"),
		attribute.String(AttrHTTPRoute, "/api/users/:id"),
		attribute.String(AttrURLScheme, "http"),
	}
	require.NoError(t, AddHTTPServerActiveRequest(context.Background(), 1, attrs...))
	require.NoError(t, RecordHTTPServerRequest(context.Background(), 125*time.Millisecond,
		append(attrs, attribute.Int(AttrHTTPResponseStatusCode, httpStatusOK))...))
	require.NoError(t, AddHTTPServerActiveRequest(context.Background(), -1, attrs...))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	foundHistogram, count, sum := findFloat64Histogram(rm, MetricHTTPServerRequestDuration)
	require.True(t, foundHistogram)
	assert.Equal(t, uint64(1), count)
	assert.InDelta(t, 0.125, sum, 0.000001)

	foundActive, activeValue := findInt64Sum(rm, MetricHTTPServerActiveRequests)
	require.True(t, foundActive)
	assert.Equal(t, int64(0), activeValue)
}

func TestRuntimeMetricsCollect(t *testing.T) {
	reader := useTestMeterProvider(t)

	require.NoError(t, InitRuntimeMetrics())

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	assert.True(t, metricExists(rm, MetricGoGoroutineCount))
	assert.True(t, metricExists(rm, MetricGoMemoryAllocated))
	assert.True(t, metricExists(rm, MetricGoMemoryUsed))
	assert.True(t, metricExists(rm, MetricGoProcessorLimit))
}

func TestInstrumentHelpersCollectCounterHistogramAndDuration(t *testing.T) {
	reader := useTestMeterProvider(t)

	require.NoError(t, InitInstruments(
		InstrumentDef{
			Name:        "business.test.created.count",
			Kind:        InstrumentKindCounter,
			Description: "Created test records",
			Unit:        "{record}",
		},
		InstrumentDef{
			Name:        "business.test.operation.duration",
			Kind:        InstrumentKindHistogram,
			Description: "Test operation duration",
			Unit:        "s",
		},
	))

	require.NoError(t, IncCounter(context.Background(), "business.test.created.count", 2,
		attribute.String("test.status", "ok")))
	require.NoError(t, RecordHistogram(context.Background(), "business.test.operation.duration", 1.5))
	require.NoError(t, RecordDuration(context.Background(), "business.test.operation.duration", 250*time.Millisecond))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	foundCounter, counterValue := findInt64Sum(rm, "business.test.created.count")
	require.True(t, foundCounter)
	assert.Equal(t, int64(2), counterValue)

	foundHistogram, count, sum := findFloat64Histogram(rm, "business.test.operation.duration")
	require.True(t, foundHistogram)
	assert.Equal(t, uint64(2), count)
	assert.InDelta(t, 1.75, sum, 0.000001)
}

func TestInitInstrumentsRejectsDuplicateDefinitions(t *testing.T) {
	useTestMeterProvider(t)

	err := InitInstruments(
		InstrumentDef{Name: "business.test.duplicate.count", Kind: InstrumentKindCounter},
		InstrumentDef{Name: "business.test.duplicate.count", Kind: InstrumentKindHistogram},
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInstrumentAlreadyRegistered))
}

func TestInitInstrumentsRejectsExistingName(t *testing.T) {
	useTestMeterProvider(t)

	require.NoError(t, InitInstruments(
		InstrumentDef{Name: "business.test.existing.count", Kind: InstrumentKindCounter},
	))

	err := InitInstruments(
		InstrumentDef{Name: "business.test.existing.count", Kind: InstrumentKindCounter},
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInstrumentAlreadyRegistered))
}

func TestRecordUnknownAndWrongKindInstruments(t *testing.T) {
	useTestMeterProvider(t)

	require.NoError(t, InitInstruments(
		InstrumentDef{Name: "business.test.kind.duration", Kind: InstrumentKindHistogram},
	))

	err := IncCounter(context.Background(), "business.test.missing.count", 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInstrumentNotFound))

	err = IncCounter(context.Background(), "business.test.kind.duration", 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInstrumentKindMismatch))
}

func TestIncCounterRejectsNegativeValue(t *testing.T) {
	useTestMeterProvider(t)

	require.NoError(t, InitInstruments(
		InstrumentDef{Name: "business.test.negative.count", Kind: InstrumentKindCounter},
	))

	err := IncCounter(context.Background(), "business.test.negative.count", -1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidMeasurement))
}

func findInt64Sum(rm metricdata.ResourceMetrics, name string) (bool, int64) {
	for _, scope := range rm.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name != name {
				continue
			}
			sum, ok := metric.Data.(metricdata.Sum[int64])
			if !ok || len(sum.DataPoints) == 0 {
				return false, 0
			}
			return true, sum.DataPoints[0].Value
		}
	}
	return false, 0
}

func findFloat64Histogram(rm metricdata.ResourceMetrics, name string) (bool, uint64, float64) {
	for _, scope := range rm.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name != name {
				continue
			}
			histogram, ok := metric.Data.(metricdata.Histogram[float64])
			if !ok || len(histogram.DataPoints) == 0 {
				return false, 0, 0
			}
			return true, histogram.DataPoints[0].Count, histogram.DataPoints[0].Sum
		}
	}
	return false, 0, 0
}

func metricExists(rm metricdata.ResourceMetrics, name string) bool {
	for _, scope := range rm.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name == name {
				return true
			}
		}
	}
	return false
}

func useTestMeterProvider(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()

	resetInstruments()
	resetHTTPInstruments()
	resetRuntimeMetrics()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	mu.Lock()
	if meterProvider != nil {
		_ = meterProvider.Shutdown(context.Background())
	}
	meterProvider = provider
	meter = provider.Meter(instrumentationName, metric.WithInstrumentationVersion("test"))
	mu.Unlock()

	t.Cleanup(func() {
		require.NoError(t, Close(context.Background()))
	})

	return reader
}

const httpStatusOK = 200
