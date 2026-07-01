package trace

import (
	"context"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// NewOutboundHTTPRequest 创建出站 HTTP 请求，自动创建 Client Span 并注入 trace context。
// 调用方在完成后应调用 FinishHTTPClient 记录响应并结束 Span。
func NewOutboundHTTPRequest(ctx context.Context, method, url string, body io.Reader, attrs ...attribute.KeyValue) (*http.Request, trace.Span, error) {
	clientAttrs := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.url", url),
	}
	clientAttrs = append(clientAttrs, attrs...)

	ctx, span := StartClient(ctx, method+" "+url, clientAttrs...)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		SetError(span, err)
		return nil, span, err
	}
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	InjectHTTP(ctx, req.Header)
	return req, span, nil
}

// FinishHTTPClient 记录出站 HTTP 响应信息并结束 Client Span。
func FinishHTTPClient(span trace.Span, statusCode int, err error) {
	if span == nil {
		return
	}
	if statusCode > 0 {
		span.SetAttributes(attribute.Int("http.status_code", statusCode))
		if statusCode >= 400 {
			span.SetStatus(codes.Error, http.StatusText(statusCode))
		}
	}
	End(span, err)
}
