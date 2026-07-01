package metrics

import (
	"context"
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
