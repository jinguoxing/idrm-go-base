# Telemetry Metrics 系统

`telemetry/metrics` 基于 OpenTelemetry Metrics SDK，通过 OTLP gRPC/HTTP 将业务指标导出到 OTel Collector 或其他兼容后端。

## 配置

```yaml
Telemetry:
  Metrics:
    Enabled: true
    Endpoint: localhost:4317
    Protocol: grpc
    Interval: 15
    Timeout: 10
    TLSEnabled: false
    RuntimeEnabled: false
```

`Enabled` 默认是 `false`，基础库不会在未显式开启时启动周期导出或连接 Collector。
`RuntimeEnabled` 默认也是 `false`，开启后会注册 Go runtime observable metrics。

## 业务埋点辅助 API

推荐在 `telemetry.Init()` 之后、服务接收流量之前集中注册业务指标：

```go
import (
    "context"

    "github.com/jinguoxing/idrm-go-base/telemetry/metrics"
    "go.opentelemetry.io/otel/attribute"
)

func InitInstruments() error {
    return metrics.InitInstruments(
        metrics.InstrumentDef{
            Name:        "business.order.created.count",
            Kind:        metrics.InstrumentKindCounter,
            Description: "Created orders",
            Unit:        "{order}",
        },
        metrics.InstrumentDef{
            Name:        "business.order.create.duration",
            Kind:        metrics.InstrumentKindHistogram,
            Description: "Order creation duration",
            Unit:        "s",
        },
    )
}

func RecordOrderCreated(ctx context.Context, status string) error {
    return metrics.IncCounter(ctx, "business.order.created.count", 1,
        attribute.String("order.status", status),
    )
}
```

`IncCounter`、`RecordHistogram`、`RecordDuration` 只写入已通过 `InitInstruments` 注册的指标名。未知指标会返回错误，不会在请求路径动态创建 instrument。

## 直接使用 OTel API

```go
import (
    "context"

    "github.com/jinguoxing/idrm-go-base/telemetry/metrics"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

var orderCounter metric.Int64Counter

func InitInstruments() error {
    var err error
    orderCounter, err = metrics.Meter().Int64Counter(
        "business.order.created.count",
        metric.WithDescription("Created orders"),
        metric.WithUnit("{order}"),
    )
    return err
}

func RecordOrderCreated(ctx context.Context, status string) {
    orderCounter.Add(ctx, 1, metric.WithAttributes(
        attribute.String("order.status", status),
    ))
}
```

## go-zero HTTP 指标

HTTP 自动指标通过 `middleware.WithMetrics` 包装 go-zero routes，使用 `rest.Route.Path` 作为低基数 `http.route`：

```go
server.AddRoutes(middleware.WithMetrics([]rest.Route{
    {
        Method: http.MethodGet,
        Path:   "/api/users/:id",
        Handler: userHandler,
    },
}...))
```

如果路由组使用了 go-zero 的 `rest.WithPrefix`，用 `WithMetricsPrefix` 保持 `http.route` 和最终模板一致：

```go
server.AddRoutes(
    middleware.WithMetricsPrefix("/api", routes...),
    rest.WithPrefix("/api"),
)
```

采集的指标：

| 指标 | 类型 | 单位 |
|------|------|------|
| `http.server.request.duration` | Histogram | `s` |
| `http.server.active_requests` | UpDownCounter | `{request}` |

如果必须使用全局 middleware，可以传入 `WithRouteNamer` 显式返回 route 模板；默认不会把 `r.URL.Path` 写入 `http.route`，避免高基数。

启用 OTel HTTP 指标后，建议关闭或在 Dashboard 中区分 go-zero 自带 `RestConf.Middlewares.Prometheus`，避免 HTTP 延迟和请求量出现两套口径。

## Go runtime 指标

开启配置：

```yaml
Telemetry:
  Metrics:
    Enabled: true
    RuntimeEnabled: true
```

当前注册的 runtime 指标包括：

| 指标 | 说明 |
|------|------|
| `go.goroutine.count` | 当前 goroutine 数 |
| `go.memory.allocated` | 累计 heap 分配字节 |
| `go.memory.allocations` | 累计 heap 分配次数 |
| `go.memory.gc.goal` | GC heap 目标 |
| `go.memory.limit` | Go runtime memory limit，无显式限制时为 `0` |
| `go.memory.used` | Go runtime 从系统获取的内存 |
| `go.processor.limit` | 当前 `GOMAXPROCS` |
