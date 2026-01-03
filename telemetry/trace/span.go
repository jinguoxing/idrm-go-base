package trace

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartInternal 开始内部方法追踪（自动提取调用信息）
// 使用 runtime.Caller 自动获取调用位置和函数名
func StartInternal(ctx context.Context) (context.Context, trace.Span) {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return Start(ctx, "unknown", trace.WithSpanKind(trace.SpanKindInternal))
	}

	// 提取函数名作为 span 名称
	funcName := runtime.FuncForPC(pc).Name()
	funcPaths := strings.Split(funcName, "/")
	spanName := funcPaths[len(funcPaths)-1]

	// 创建 span
	ctx, span := Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindInternal))

	// 添加代码位置属性
	span.SetAttributes(
		attribute.String("code.filepath", file),
		attribute.Int("code.lineno", line),
		attribute.String("code.function", funcName),
	)

	return ctx, span
}

// StartServer 开始服务端 Span（用于 HTTP/RPC 服务）
func StartServer(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return ctx, span
}

// StartClient 开始客户端 Span（用于 HTTP/RPC 调用）
func StartClient(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return ctx, span
}

// StartConsumer 开始消费者 Span（用于消息队列消费）
func StartConsumer(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindConsumer))
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return ctx, span
}

// StartProducer 开始生产者 Span（用于消息队列生产）
func StartProducer(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindProducer))
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return ctx, span
}

// End 结束 Span 并记录错误（如果有）
func End(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}
	span.End()
}

// SetError 设置错误状态
func SetError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetAttributes 设置属性
func SetAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
}

// AddEvent 添加事件
func AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// GetSpan 从 Context 获取当前 Span
func GetSpan(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// GetTraceID 从 Context 获取 TraceID
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID 从 Context 获取 SpanID
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// RecordError 记录错误到 Span
func RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	if err == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	if len(attrs) > 0 {
		span.RecordError(err, trace.WithAttributes(attrs...))
	} else {
		span.RecordError(err)
	}
}

// WithAttributes 便捷方法：创建属性
func WithAttributes(kv ...interface{}) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(kv)/2)
	for i := 0; i < len(kv)-1; i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}

		switch v := kv[i+1].(type) {
		case string:
			attrs = append(attrs, attribute.String(key, v))
		case int:
			attrs = append(attrs, attribute.Int(key, v))
		case int64:
			attrs = append(attrs, attribute.Int64(key, v))
		case float64:
			attrs = append(attrs, attribute.Float64(key, v))
		case bool:
			attrs = append(attrs, attribute.Bool(key, v))
		default:
			attrs = append(attrs, attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}
	return attrs
}
