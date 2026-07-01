package trace

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// ExtractHTTP 从 HTTP 请求头提取 trace context（W3C Trace Context / Baggage）
func ExtractHTTP(ctx context.Context, header http.Header) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(header))
}

// InjectHTTP 将 trace context 注入 HTTP 请求头，供出站 HTTP 调用使用
func InjectHTTP(ctx context.Context, header http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
}
