package trace

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestNewOutboundHTTPRequest(t *testing.T) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
	))
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	tracer = otel.Tracer(instrumentationName)

	var receivedTraceparent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTraceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, parent := otel.Tracer("test").Start(context.Background(), "caller")
	defer parent.End()

	req, span, err := NewOutboundHTTPRequest(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, span)
	defer span.End()

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	FinishHTTPClient(span, resp.StatusCode, nil)

	require.NotEmpty(t, receivedTraceparent)
	assert.Equal(t, parent.SpanContext().TraceID(), span.SpanContext().TraceID())
}
