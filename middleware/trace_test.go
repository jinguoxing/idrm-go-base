package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jinguoxing/idrm-go-base/telemetry/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestTraceMiddlewareExtractsParentContext(t *testing.T) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
	))
	otel.SetTracerProvider(sdktrace.NewTracerProvider())

	var gotTraceID string
	handler := Trace()(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = trace.GetTraceID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	rec := httptest.NewRecorder()

	handler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", gotTraceID)

	span := oteltrace.SpanFromContext(req.Context())
	_ = span
}
