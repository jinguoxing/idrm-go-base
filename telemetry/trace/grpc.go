package trace

import (
	"context"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// GRPCClientHandler 返回 gRPC 客户端 StatsHandler，配合 grpc.WithStatsHandler 使用。
func GRPCClientHandler(opts ...otelgrpc.Option) grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler(opts...))
}

// GRPCServerHandler 返回 gRPC 服务端 StatsHandler，配合 grpc.StatsHandler 使用。
func GRPCServerHandler(opts ...otelgrpc.Option) grpc.ServerOption {
	return grpc.StatsHandler(otelgrpc.NewServerHandler(opts...))
}

// ExtractGRPC 从 gRPC metadata 提取 trace context。
func ExtractGRPC(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, &grpcMetadataCarrier{md: md})
}

// InjectGRPC 将 trace context 注入 gRPC outgoing metadata。
func InjectGRPC(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	} else {
		md = md.Copy()
	}
	otel.GetTextMapPropagator().Inject(ctx, &grpcMetadataCarrier{md: md})
	return metadata.NewOutgoingContext(ctx, md)
}

type grpcMetadataCarrier struct {
	md metadata.MD
}

func (c *grpcMetadataCarrier) Get(key string) string {
	values := c.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c *grpcMetadataCarrier) Set(key, value string) {
	c.md.Set(key, value)
}

func (c *grpcMetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.md))
	for k := range c.md {
		keys = append(keys, k)
	}
	return keys
}

var _ propagation.TextMapCarrier = (*grpcMetadataCarrier)(nil)
