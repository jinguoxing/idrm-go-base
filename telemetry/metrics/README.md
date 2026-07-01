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
```

`Enabled` 默认是 `false`，基础库不会在未显式开启时启动周期导出或连接 Collector。

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

Phase 1 只提供业务指标入口，不实现 HTTP 自动指标。go-zero 自带 `RestConf.Middlewares.Prometheus` / `Metrics` 已覆盖 HTTP 框架级指标。后续如果启用 OTel HTTP 指标，应使用 route-aware 包装从 `rest.Route.Path` 获取低基数 `http.route`。
