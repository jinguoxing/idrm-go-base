package otelresource

import (
	"os"
	"runtime"

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
	hostname, _ := os.Hostname()

	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(environment),
			semconv.ServiceInstanceID(instanceID),
			attribute.String("host.name", hostname),
			attribute.String("host.arch", runtime.GOARCH),
			attribute.String("process.runtime.name", "go"),
			attribute.String("process.runtime.version", runtime.Version()),
			attribute.String("environment", environment),
		),
	)
}
