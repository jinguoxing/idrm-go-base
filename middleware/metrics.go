package middleware

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	telemetrymetrics "github.com/jinguoxing/idrm-go-base/telemetry/metrics"

	"go.opentelemetry.io/otel/attribute"

	"github.com/zeromicro/go-zero/rest"
)

// MetricsOption customizes HTTP metrics middleware behavior.
type MetricsOption func(*metricsOptions)

type metricsOptions struct {
	routeNamer func(*http.Request) string
}

// WithRouteNamer provides a low-cardinality route template for global middleware usage.
func WithRouteNamer(fn func(*http.Request) string) MetricsOption {
	return func(opts *metricsOptions) {
		opts.routeNamer = fn
	}
}

// Metrics creates a fallback global HTTP metrics middleware.
//
// Prefer MetricsForRoute or WithMetrics for go-zero routes so http.route uses
// the registered route template instead of the raw request path.
func Metrics(opts ...MetricsOption) func(http.HandlerFunc) http.HandlerFunc {
	return metricsFor("", "", opts...)
}

// MetricsForRoute creates a route-aware HTTP metrics middleware.
func MetricsForRoute(routePath, method string, opts ...MetricsOption) func(http.HandlerFunc) http.HandlerFunc {
	return metricsFor(routePath, method, opts...)
}

// WithMetrics wraps go-zero routes with route-aware HTTP metrics.
func WithMetrics(routes ...rest.Route) []rest.Route {
	return withMetrics("", routes...)
}

// WithMetricsPrefix wraps go-zero routes with route-aware HTTP metrics and a route prefix.
//
// Use this together with go-zero rest.WithPrefix so the metrics http.route label
// matches the final route template after go-zero applies the prefix.
func WithMetricsPrefix(prefix string, routes ...rest.Route) []rest.Route {
	return withMetrics(prefix, routes...)
}

func withMetrics(prefix string, routes ...rest.Route) []rest.Route {
	wrapped := make([]rest.Route, 0, len(routes))
	for _, route := range routes {
		if route.Handler == nil {
			wrapped = append(wrapped, route)
			continue
		}

		next := route
		next.Handler = MetricsForRoute(prefixedRoutePath(prefix, route.Path), route.Method)(route.Handler)
		wrapped = append(wrapped, next)
	}
	return wrapped
}

func metricsFor(routePath, method string, optFns ...MetricsOption) func(http.HandlerFunc) http.HandlerFunc {
	opts := metricsOptions{}
	for _, opt := range optFns {
		if opt != nil {
			opt(&opts)
		}
	}

	routePath = strings.TrimSpace(routePath)
	method = strings.TrimSpace(method)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			route := routePath
			if route == "" && opts.routeNamer != nil {
				route = strings.TrimSpace(opts.routeNamer(r))
			}
			requestMethod := method
			if requestMethod == "" {
				requestMethod = r.Method
			}

			baseAttrs := httpMetricAttributes(r, route, requestMethod, 0)
			_ = telemetrymetrics.AddHTTPServerActiveRequest(ctx, 1, baseAttrs...)
			defer func() {
				_ = telemetrymetrics.AddHTTPServerActiveRequest(ctx, -1, baseAttrs...)
			}()

			start := time.Now()
			sw := newMetricsStatusWriter(w)
			next(sw, r)

			attrs := httpMetricAttributes(r, route, requestMethod, sw.statusCode)
			_ = telemetrymetrics.RecordHTTPServerRequest(ctx, time.Since(start), attrs...)
		}
	}
}

func httpMetricAttributes(r *http.Request, route, method string, statusCode int) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 4)
	attrs = append(attrs,
		attribute.String(telemetrymetrics.AttrHTTPRequestMethod, method),
		attribute.String(telemetrymetrics.AttrURLScheme, getScheme(r)),
	)
	if route != "" {
		attrs = append(attrs, attribute.String(telemetrymetrics.AttrHTTPRoute, route))
	}
	if statusCode > 0 {
		attrs = append(attrs, attribute.Int(telemetrymetrics.AttrHTTPResponseStatusCode, statusCode))
	}
	return attrs
}

func prefixedRoutePath(prefix, routePath string) string {
	prefix = strings.TrimSpace(prefix)
	routePath = strings.TrimSpace(routePath)
	if prefix == "" {
		return routePath
	}
	if routePath == "" {
		return prefix
	}
	return path.Join(prefix, routePath)
}

type metricsStatusWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
	bytes       int64
}

func newMetricsStatusWriter(w http.ResponseWriter) *metricsStatusWriter {
	return &metricsStatusWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (w *metricsStatusWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.statusCode = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *metricsStatusWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytes += int64(n)
	return n, err
}

func (w *metricsStatusWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *metricsStatusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("underlying ResponseWriter does not support http.Hijacker")
	}
	return hijacker.Hijack()
}

func (w *metricsStatusWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (w *metricsStatusWriter) ReadFrom(reader io.Reader) (int64, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		n, err := rf.ReadFrom(reader)
		w.bytes += n
		return n, err
	}

	n, err := io.Copy(w.ResponseWriter, reader)
	w.bytes += n
	return n, err
}

func (w *metricsStatusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
