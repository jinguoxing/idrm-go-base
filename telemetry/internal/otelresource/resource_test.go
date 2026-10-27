package otelresource

import (
	"testing"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func TestNewNoSchemaConflict(t *testing.T) {
	res, err := New("test-svc", "0.0.0", "dev")
	if err != nil {
		t.Fatalf("New() 不应报错: %v", err)
	}
	if got, want := res.SchemaURL(), resource.Default().SchemaURL(); got != want {
		t.Fatalf("schema URL = %q, want Default %q", got, want)
	}
	val, ok := res.Set().Value(semconv.ServiceNameKey)
	if !ok || val.AsString() != "test-svc" {
		t.Fatalf("service.name = %q, ok = %v", val.AsString(), ok)
	}
}
