package otelresource

import (
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// New builds the shared OpenTelemetry resource used by telemetry signals.
func New(serviceName, version, environment string) (*resource.Resource, error) {
	instanceID := os.Getenv("HOSTNAME")
	if instanceID == "" {
		hostname, err := os.Hostname()
		if err == nil {
			instanceID = hostname
		}
	}

	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(environment),
			semconv.ServiceInstanceID(instanceID),
			attribute.String("environment", environment),
		),
	)
}
