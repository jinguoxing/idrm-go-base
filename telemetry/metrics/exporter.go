package metrics

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/zeromicro/go-zero/core/logx"
)

func newExporter(ctx context.Context, config MetricsConfig) (sdkmetric.Exporter, error) {
	timeout := exportTimeout(config.Timeout)

	grpcRetry := otlpmetricgrpc.RetryConfig{
		Enabled:         true,
		InitialInterval: 5 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxElapsedTime:  time.Minute,
	}
	httpRetry := otlpmetrichttp.RetryConfig{
		Enabled:         true,
		InitialInterval: 5 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxElapsedTime:  time.Minute,
	}

	switch normalizeProtocol(config.Protocol) {
	case "http":
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(normalizeHTTPEndpoint(config.Endpoint)),
			otlpmetrichttp.WithTimeout(timeout),
			otlpmetrichttp.WithRetry(httpRetry),
		}
		if !config.TLSEnabled {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(config.Headers) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(config.Headers))
		}
		exporter, err := otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			logx.Errorf("创建 OTLP Metrics HTTP exporter 失败: %v", err)
			return nil, err
		}
		return exporter, nil
	default:
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(normalizeGRPCEndpoint(config.Endpoint)),
			otlpmetricgrpc.WithTimeout(timeout),
			otlpmetricgrpc.WithRetry(grpcRetry),
		}
		if !config.TLSEnabled {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if len(config.Headers) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(config.Headers))
		}
		exporter, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			logx.Errorf("创建 OTLP Metrics gRPC exporter 失败: %v", err)
			return nil, err
		}
		return exporter, nil
	}
}

func normalizeProtocol(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "http":
		return "http"
	default:
		return "grpc"
	}
}

func normalizeGRPCEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "grpc://")
	if endpoint == "" {
		return "localhost:4317"
	}
	return endpoint
}

func normalizeHTTPEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "localhost:4318"
	}
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return endpoint
}

func exportTimeout(timeoutSeconds int) time.Duration {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		return 10 * time.Second
	}
	return timeout
}
