package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	telemetrymetrics "github.com/jinguoxing/idrm-go-base/telemetry/metrics"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/zeromicro/go-zero/rest"
)

func TestWithMetricsRecordsRouteTemplate(t *testing.T) {
	reader, shutdown := setupMetricsReader(t)
	defer shutdown()

	routes := WithMetrics(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/users/:id",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
	})
	require.Len(t, routes, 1)

	req := httptest.NewRequest(http.MethodGet, "/api/users/123", nil)
	rec := httptest.NewRecorder()
	routes[0].Handler(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	point := findHTTPDurationPoint(t, rm)
	assert.Equal(t, uint64(1), point.Count)
	assertStringAttr(t, point.Attributes, telemetrymetrics.AttrHTTPRequestMethod, http.MethodGet)
	assertStringAttr(t, point.Attributes, telemetrymetrics.AttrHTTPRoute, "/api/users/:id")
	assertStringAttr(t, point.Attributes, telemetrymetrics.AttrURLScheme, "http")
	assertIntAttr(t, point.Attributes, telemetrymetrics.AttrHTTPResponseStatusCode, http.StatusCreated)
}

func TestMetricsWithRouteNamerRecordsRouteTemplate(t *testing.T) {
	reader, shutdown := setupMetricsReader(t)
	defer shutdown()

	handler := Metrics(WithRouteNamer(func(r *http.Request) string {
		return "/api/orders/:id"
	}))(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodPost, "/api/orders/abc", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	point := findHTTPDurationPoint(t, rm)
	assert.Equal(t, uint64(1), point.Count)
	assertStringAttr(t, point.Attributes, telemetrymetrics.AttrHTTPRequestMethod, http.MethodPost)
	assertStringAttr(t, point.Attributes, telemetrymetrics.AttrHTTPRoute, "/api/orders/:id")
	assertIntAttr(t, point.Attributes, telemetrymetrics.AttrHTTPResponseStatusCode, http.StatusOK)
}

func TestMetricsWithoutRouteNamerOmitsHTTPRoute(t *testing.T) {
	reader, shutdown := setupMetricsReader(t)
	defer shutdown()

	handler := Metrics()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/high-cardinality/123", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	point := findHTTPDurationPoint(t, rm)
	assertMissingAttr(t, point.Attributes, telemetrymetrics.AttrHTTPRoute)
	assertStringAttr(t, point.Attributes, telemetrymetrics.AttrHTTPRequestMethod, http.MethodPost)
	assertIntAttr(t, point.Attributes, telemetrymetrics.AttrHTTPResponseStatusCode, http.StatusAccepted)
}

func TestWithMetricsPrefixRecordsPrefixedRouteTemplate(t *testing.T) {
	reader, shutdown := setupMetricsReader(t)
	defer shutdown()

	routes := WithMetricsPrefix("/api", rest.Route{
		Method: http.MethodPatch,
		Path:   "/users/:id",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})
	require.Len(t, routes, 1)
	assert.Equal(t, "/users/:id", routes[0].Path)

	req := httptest.NewRequest(http.MethodPatch, "/api/users/123", nil)
	rec := httptest.NewRecorder()
	routes[0].Handler(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	point := findHTTPDurationPoint(t, rm)
	assertStringAttr(t, point.Attributes, telemetrymetrics.AttrHTTPRequestMethod, http.MethodPatch)
	assertStringAttr(t, point.Attributes, telemetrymetrics.AttrHTTPRoute, "/api/users/:id")
	assertIntAttr(t, point.Attributes, telemetrymetrics.AttrHTTPResponseStatusCode, http.StatusNoContent)
}

func setupMetricsReader(t *testing.T) (*sdkmetric.ManualReader, func()) {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)

	return reader, func() {
		require.NoError(t, telemetrymetrics.Close(context.Background()))
		require.NoError(t, provider.Shutdown(context.Background()))
		otel.SetMeterProvider(noop.NewMeterProvider())
	}
}

func findHTTPDurationPoint(t *testing.T, rm metricdata.ResourceMetrics) metricdata.HistogramDataPoint[float64] {
	t.Helper()

	for _, scope := range rm.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name != telemetrymetrics.MetricHTTPServerRequestDuration {
				continue
			}
			histogram, ok := metric.Data.(metricdata.Histogram[float64])
			require.True(t, ok)
			require.NotEmpty(t, histogram.DataPoints)
			return histogram.DataPoints[0]
		}
	}

	require.Failf(t, "missing metric", telemetrymetrics.MetricHTTPServerRequestDuration)
	return metricdata.HistogramDataPoint[float64]{}
}

func assertStringAttr(t *testing.T, attrs attribute.Set, key, want string) {
	t.Helper()

	value, ok := attrs.Value(attribute.Key(key))
	require.True(t, ok, "missing attribute %s", key)
	assert.Equal(t, want, value.AsString())
}

func assertIntAttr(t *testing.T, attrs attribute.Set, key string, want int) {
	t.Helper()

	value, ok := attrs.Value(attribute.Key(key))
	require.True(t, ok, "missing attribute %s", key)
	assert.Equal(t, int64(want), value.AsInt64())
}

func assertMissingAttr(t *testing.T, attrs attribute.Set, key string) {
	t.Helper()

	_, ok := attrs.Value(attribute.Key(key))
	assert.False(t, ok, "unexpected attribute %s", key)
}
