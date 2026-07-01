package trace

import (
	"context"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/zeromicro/go-zero/core/logx"
)

const instrumentationName = "github.com/jinguoxing/idrm-go-base/telemetry/trace"

var (
	tracerProvider         *sdktrace.TracerProvider
	tracer                 trace.Tracer
	instrumentationVersion = "dev"
)

// TraceConfig 链路追踪配置
type TraceConfig struct {
	Enabled  bool
	Endpoint string            // OTLP endpoint，gRPC 为 host:port，HTTP 可为 http(s)://host:port
	Protocol string            // otlp 传输协议：grpc（默认）或 http
	Sampler     float64           // 采样率 0.0-1.0
	TLSEnabled  bool              // 是否启用 TLS 连接 Collector，默认 false
	Headers     map[string]string // OTLP 导出请求头，如 Collector 认证 token
	Batcher  string            // 保留字段，当前固定使用 OTLP batch exporter
}

// Init 初始化链路追踪
func Init(config TraceConfig, serviceName, version, environment string) error {
	if version != "" {
		instrumentationVersion = version
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if !config.Enabled {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		tracer = otel.Tracer(instrumentationName, trace.WithInstrumentationVersion(instrumentationVersion))
		logx.Info("链路追踪未启用")
		return nil
	}

	ctx := context.Background()

	exporter, err := newExporter(ctx, config)
	if err != nil {
		return err
	}

	res, err := newResource(serviceName, version, environment)
	if err != nil {
		logx.Errorf("创建 resource 失败: %v", err)
		return err
	}

	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxQueueSize(1000),
			sdktrace.WithMaxExportBatchSize(100),
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(config.Sampler))),
	)

	otel.SetTracerProvider(tracerProvider)
	tracer = tracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
	)

	logx.Infof("链路追踪初始化完成 [protocol=%s, endpoint=%s, sampler=%.2f]",
		normalizeProtocol(config.Protocol), config.Endpoint, config.Sampler)

	return nil
}

func newExporter(ctx context.Context, config TraceConfig) (sdktrace.SpanExporter, error) {
	grpcRetry := otlptracegrpc.RetryConfig{
		Enabled:         true,
		InitialInterval: 5 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxElapsedTime:  time.Minute,
	}
	httpRetry := otlptracehttp.RetryConfig{
		Enabled:         true,
		InitialInterval: 5 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxElapsedTime:  time.Minute,
	}

	switch normalizeProtocol(config.Protocol) {
	case "http":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(normalizeHTTPEndpoint(config.Endpoint)),
			otlptracehttp.WithTimeout(10 * time.Second),
			otlptracehttp.WithRetry(httpRetry),
		}
		if !config.TLSEnabled {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(config.Headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(config.Headers))
		}
		exporter, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			logx.Errorf("创建 OTLP HTTP exporter 失败: %v", err)
			return nil, err
		}
		return exporter, nil
	default:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(normalizeGRPCEndpoint(config.Endpoint)),
			otlptracegrpc.WithTimeout(10 * time.Second),
			otlptracegrpc.WithRetry(grpcRetry),
		}
		if !config.TLSEnabled {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(config.Headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(config.Headers))
		}
		exporter, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			logx.Errorf("创建 OTLP gRPC exporter 失败: %v", err)
			return nil, err
		}
		return exporter, nil
	}
}

func newResource(serviceName, version, environment string) (*resource.Resource, error) {
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

// Start 开始一个 Span
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil {
		return otel.Tracer(instrumentationName).Start(ctx, spanName, opts...)
	}
	return tracer.Start(ctx, spanName, opts...)
}

// Tracer 获取全局 Tracer
func Tracer() trace.Tracer {
	if tracer == nil {
		return otel.Tracer(instrumentationName, trace.WithInstrumentationVersion(instrumentationVersion))
	}
	return tracer
}

// Close 关闭链路追踪
func Close(ctx context.Context) error {
	if tracerProvider != nil {
		logx.Info("关闭链路追踪...")
		return tracerProvider.Shutdown(ctx)
	}
	return nil
}
