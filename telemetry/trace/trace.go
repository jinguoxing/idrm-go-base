package trace

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
)

// TraceConfig 链路追踪配置
type TraceConfig struct {
	Enabled  bool
	Endpoint string
	Sampler  float64
	Batcher  string
}

// Init 初始化链路追踪
func Init(config TraceConfig, serviceName, version, environment string) error {
	if !config.Enabled {
		logx.Info("链路追踪未启用")
		return nil
	}

	ctx := context.Background()

	// 1. 创建 OTLP gRPC Exporter
	conn, err := grpc.DialContext(ctx, config.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		logx.Errorf("连接 OTLP endpoint 失败: %v", err)
		return err
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		logx.Errorf("创建 OTLP exporter 失败: %v", err)
		return err
	}

	// 2. 创建 Resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			attribute.String("environment", environment),
		),
	)
	if err != nil {
		logx.Errorf("创建 resource 失败: %v", err)
		return err
	}

	// 3. 创建 TracerProvider
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxQueueSize(1000),
			sdktrace.WithMaxExportBatchSize(100),
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.Sampler)),
	)

	// 4. 设置全局 TracerProvider
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 5. 创建 Tracer
	tracer = tracerProvider.Tracer(serviceName)

	logx.Infof("链路追踪初始化完成 [endpoint=%s, sampler=%.2f]",
		config.Endpoint, config.Sampler)

	return nil
}

// Start 开始一个 Span
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil {
		// 如果未初始化，返回 noop span
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, spanName, opts...)
}

// Tracer 获取全局 Tracer
func Tracer() trace.Tracer {
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
