package trace

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func setupPropagator(t *testing.T) {
	t.Helper()
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
}

func TestExtractInjectHTTP(t *testing.T) {
	setupPropagator(t)

	ctx, span := otel.Tracer("test").Start(context.Background(), "parent")
	defer span.End()

	header := make(http.Header)
	InjectHTTP(ctx, header)

	require.NotEmpty(t, header.Get("traceparent"))

	childCtx := ExtractHTTP(context.Background(), header)
	childSpan := trace.SpanFromContext(childCtx)
	require.True(t, childSpan.SpanContext().IsValid())
	assert.Equal(t, span.SpanContext().TraceID(), childSpan.SpanContext().TraceID())
}

func TestExtractHTTPPreservesRemoteParent(t *testing.T) {
	setupPropagator(t)

	header := http.Header{}
	header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	ctx := ExtractHTTP(context.Background(), header)
	span := trace.SpanFromContext(ctx)
	require.True(t, span.SpanContext().IsValid())
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", span.SpanContext().TraceID().String())
}
