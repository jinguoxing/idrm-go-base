# Telemetry Metrics 模块设计方案

> 版本：v0.2
> 状态：Phase 2 部分实现
> 最后更新：2026-07-01

---

## 1. 背景与目标

### 1.1 现状

`idrm-go-base` 的 `telemetry` 包目前已具备三大能力：

| 模块 | 状态 | 导出协议 |
|------|------|----------|
| Log | ✅ 本地完善，远程上报待完善 | 自定义 HTTP |
| Trace | ✅ 完整 | OTLP gRPC / HTTP |
| Audit | ✅ 完整 | 自定义 HTTP |
| **Metrics** | ❌ **缺失** | — |

Trace 模块已采用 OpenTelemetry 标准 + OTLP 导出，Metrics 应遵循相同技术路线，形成 **Logs / Traces / Metrics 三支柱** 中缺失的一环。

### 1.2 参考对象

参考 [TelemetrySDK-Go/exporter/ar_metric](https://github.com/kweaver-ai/TelemetrySDK-Go/tree/main/exporter/ar_metric) 的**思路**，但不照搬其实现：

| ar_metric 做法 | 本方案取舍 |
|----------------|-----------|
| 自定义 AnyRobot JSON 格式 | ❌ 不采用，使用标准 **OTLP** |
| 独立 `public.Client` 传输层 | ❌ 不采用，复用 OTel 官方 OTLP Exporter |
| 全局 `MetricProvider` / `Meter` | ✅ 保留便利性，但纳入统一生命周期管理 |
| `PeriodicReader` 周期导出 | ✅ 采用，可配置导出间隔 |
| `MetricResource()` 主机元数据 | ✅ 采用，与 Trace Resource 对齐 |
| 每条 Metric 立即发送、无缓冲 | ❌ 不采用，交给 OTel SDK 聚合与批量导出 |
| 业务侧直接使用 OTel Meter API | ✅ 采用，必要时提供薄封装辅助方法 |

### 1.3 设计目标

1. **通用性**：对接任意支持 OTLP Metrics 的后端（Prometheus via Collector、Grafana Mimir、Datadog Agent、Grafana Cloud 等）
2. **一致性**：与现有 `telemetry/trace` 在配置风格、Init/Close 生命周期、Resource 属性上保持一致
3. **低侵入**：业务代码可直接使用 OTel Metric API；提供更友好的辅助方法作为可选项
4. **生产就绪**：TLS、认证 Header、导出重试、优雅关闭、Enabled 开关
5. **可演进**：分阶段交付，Phase 1 即可独立使用

### 1.4 非目标（Out of Scope）

- 不实现 AnyRobot / 私有 JSON 格式 Exporter
- 不实现 ProtonMQ / 文件 / 控制台等自定义 Sink（开发调试可用 OTel Collector 转发）
- 不替换 go-zero 内置的 Prometheus `/metrics` 端点（与之共存，见 §8）
- 不接管 go-zero 原生 `RestConf.Middlewares.Metrics` / `Prometheus`；HTTP 自动指标只在用户显式接入时启用
- Phase 1 不包含 ExponentialHistogram 专项适配（不主动配置 View / Aggregation）

---

## 2. 架构设计

### 2.1 整体定位

```
┌─────────────────────────────────────────────────────────────┐
│                     telemetry.Init()                         │
│  ┌─────────┐  ┌─────────┐  ┌──────────┐  ┌─────────┐       │
│  │  log    │  │  trace  │  │ metrics  │  │  audit  │       │
│  └────┬────┘  └────┬────┘  └────┬─────┘  └────┬────┘       │
└───────┼────────────┼────────────┼─────────────┼─────────────┘
        │            │            │             │
        ▼            ▼            ▼             ▼
     logx/文件    OTLP Trace   OTLP Metrics   HTTP Audit
                       │            │
                       └─────┬──────┘
                             ▼
                    OTel Collector（可选/推荐）
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
          Prometheus   Grafana Mimir   Datadog/Grafana Cloud
```

Metrics 模块在架构上与 Trace **对称**：同样通过 OTLP 导出，共享 Endpoint / Protocol / TLS / Headers 配置模式。

> 注：Jaeger 属于 Trace 后端，不作为 Metrics 的目标后端；若 Collector 同时配置 traces pipeline，可由 Trace 模块继续导出到 Jaeger。

### 2.2 模块内部分层

```
telemetry/
├── internal/
│   └── otelresource/
│       └── resource.go # Trace/Metrics 共享 Resource 构建，避免根 telemetry 包循环依赖
└── metrics/
    ├── metrics.go      # Init / Close / Meter / MeterProvider
    ├── config.go       # MetricsConfig
    ├── exporter.go     # OTLP gRPC/HTTP exporter 构建
    ├── instrument.go   # 业务辅助 API（Phase 2）
    ├── runtime.go      # Go runtime 指标（Phase 3，可选）
    ├── http.go         # HTTP 服务端指标辅助（Phase 3）
    ├── *_test.go
    ├── README.md       # 使用文档（实现后）
    └── docs/
        ├── README.md
        └── DESIGN.md   # 本文档
```

### 2.3 数据流

```
业务代码                OTel SDK                  Exporter
─────────              ─────────                ─────────
Meter.Counter()  ──►  Aggregator（内存聚合）  ──►  PeriodicReader
Meter.Histogram()       Views / Temporality         │
Meter.ObservableGauge()                             ▼
                                              OTLP Export
                                                    │
                                                    ▼
                                           Collector / Backend
```

与 ar_metric「每条立即发送」不同，本方案完全依赖 OTel SDK 的 **PeriodicReader + 聚合器**，这是 OTel 官方推荐的生产实践，性能与可靠性更好。

---

## 3. 配置设计

### 3.1 MetricsConfig

与 `TraceConfig` 保持对称，新增导出周期相关字段：

```go
// MetricsConfig 指标配置
type MetricsConfig struct {
    Enabled  bool              `json:",default=false"`
    Endpoint string            `json:",default=localhost:4317"` // OTLP endpoint
    Protocol string            `json:",default=grpc"`           // grpc 或 http
    Interval int               `json:",default=15"`             // 导出间隔（秒）
    Timeout  int               `json:",default=10"`             // 单次导出超时（秒）
    TLSEnabled bool            `json:",default=false"`
    Headers    map[string]string `json:",optional"`            // 认证 token 等
    RuntimeEnabled bool        `json:",default=false"`          // Go runtime 指标
}
```

**默认值说明**：

| 字段 | 默认值 | 理由 |
|------|--------|------|
| Enabled | `false` | 基础库默认不主动推送，避免未部署 Collector 时产生周期性错误日志 |
| Endpoint | `localhost:4317` | 与 Trace gRPC 默认一致，可共用 Collector |
| Protocol | `grpc` | 与 Trace 默认一致 |
| Interval | `15s` | OTel 官方示例常用 15–60s；比 ar_metric 示例 10s 略保守 |
| Timeout | `10s` | Reader 单次 collect/export 超时；Exporter RPC 超时也默认使用该值 |
| RuntimeEnabled | `false` | runtime 指标默认按需启用，避免基础库默认增加额外观测项 |

### 3.2 顶层 Config 扩展

```go
// telemetry/config.go
type Config struct {
    ServiceName    string
    ServiceVersion string
    Environment    string

    Log     LogConfig
    Trace   TraceConfig
    Metrics MetricsConfig   // 新增
    Audit   AuditConfig
}
```

### 3.3 YAML 配置示例

```yaml
Telemetry:
  ServiceName: idrm-api
  ServiceVersion: 1.0.0
  Environment: prod

  Trace:
    Enabled: true
    Endpoint: collector.example.com:4317
    Protocol: grpc
    Sampler: 0.1
    TLSEnabled: true
    Headers:
      Authorization: "Bearer <token>"

  Metrics:
    Enabled: true
    Endpoint: collector.example.com:4317   # 可与 Trace 共用同一 Collector
    Protocol: grpc
    Interval: 15                           # 每 15 秒导出一次
    Timeout: 10
    RuntimeEnabled: true                   # 可选：Go runtime 指标
    TLSEnabled: true
    Headers:
      Authorization: "Bearer <token>"
```

**共用 Collector 说明**：OTLP gRPC 在同一端口（4317）可同时接收 Trace 和 Metrics，无需分开部署。

### 3.4 常用后端配置

| 后端 | Endpoint | Protocol | 备注 |
|------|----------|----------|------|
| OTel Collector | `localhost:4317` | grpc | 推荐，统一接入 |
| OTel Collector HTTP | `localhost:4318` | http | |
| Grafana Alloy / Agent | 按部署文档 | grpc | |
| 本地开发 | `localhost:4317` | grpc | 配合 `otelcol`、Prometheus 或 Grafana |

---

## 4. 核心 API 设计

### 4.1 生命周期

与 `trace` 包对称：

```go
package metrics

// Init 初始化 Metrics 系统
func Init(config MetricsConfig, serviceName, version, environment string) error

// Close 优雅关闭，刷新剩余指标
func Close(ctx context.Context) error

// Meter 获取全局 Meter（业务埋点入口）
func Meter() metric.Meter

// MeterProvider 获取全局 Provider（高级场景）
func MeterProvider() metric.MeterProvider
```

**Init 内部流程**：

1. 若 `Enabled == false`：注册 `metric/noop` `MeterProvider`，返回
2. 构建 OTLP Exporter（gRPC 或 HTTP，含 TLS / Headers / Retry）
3. 构建 Resource（复用 §4.4）
4. 创建 `MeterProvider`：
   ```go
   sdkmetric.NewMeterProvider(
       sdkmetric.WithResource(res),
       sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
           sdkmetric.WithInterval(interval),
           sdkmetric.WithTimeout(timeout),
       )),
   )
   ```
5. `otel.SetMeterProvider(provider)`
6. 初始化包级 `meter` 变量

**Close 内部流程**：

1. 若内部存在 `*sdkmetric.MeterProvider`，调用 `Shutdown(ctx)`（会触发最后一次 ForceFlush）
2. 置 nil，防止重复关闭

### 4.2 telemetry 集成

```go
// telemetry/telemetry.go — Init 顺序
func Init(config Config) error {
    log.Init(...)
    trace.Init(...)
    metrics.Init(config.Metrics, config.ServiceName, config.ServiceVersion, config.Environment)  // 新增
    audit.Init(...)
}

// Close 顺序（逆序）
func Close(ctx context.Context) {
    audit.Close()
    metrics.Close(ctx)   // 新增
    trace.Close(ctx)
    log.Close()
}
```

Metrics 在 Trace 之后、Audit 之前初始化：确保 Resource 属性一致，且 Metrics 关闭时 Trace 仍可关联（若后续做 exemplar 关联）。

**失败策略**：

- 配置错误、Exporter 构建失败：返回 error，交由 `telemetry.Init()` 决定是否中断启动
- Collector 暂时不可达：不应在 Init 阶段主动探活，交给 OTLP Exporter retry 与后续日志处理
- `Metrics.Enabled=false`：使用 noop provider，不启动 PeriodicReader，不建立网络连接

### 4.3 业务埋点 API

**原则**：Phase 1 直接暴露 OTel 标准 API，不重复造轮子。

```go
import (
    "github.com/jinguoxing/idrm-go-base/telemetry/metrics"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

var latencyHist metric.Float64Histogram

func InitMetrics() error {
    // 推荐在 telemetry.Init() 之后、服务接收流量之前注册一次。
    var err error
    latencyHist, err = metrics.Meter().Float64Histogram(
        "http.server.request.duration",
        metric.WithDescription("HTTP request duration"),
        metric.WithUnit("s"),
    )
    return err
}

func handleRequest(ctx context.Context) {
    latencyHist.Record(ctx, 0.123,
        metric.WithAttributes(
            attribute.String("http.request.method", "GET"),
            attribute.String("http.route", "/api/users"),
            attribute.Int("http.response.status_code", 200),
        ),
    )
}
```

### 4.4 Resource 设计

与 Trace 共享相同的服务级属性，保证在同一后端可按 `service.name` 关联：

```go
// 目标 Resource 属性
semconv.ServiceName(serviceName)
semconv.ServiceVersion(version)
semconv.DeploymentEnvironment(environment)
semconv.ServiceInstanceID(instanceID)   // HOSTNAME 或 os.Hostname()
attribute.String("environment", environment)
```

**已实现增强**（参考 ar_metric 的 MetricResource，但不过度依赖 gopsutil）：

| 属性 | 来源 | 优先级 |
|------|------|--------|
| `host.name` | `os.Hostname()` | 已实现 |
| `host.arch` | `runtime.GOARCH` | 已实现 |
| `process.runtime.name` | 固定 `go` | 已实现 |
| `process.runtime.version` | `runtime.Version()` | 已实现 |

Trace / Metrics 的 Resource 构建逻辑已抽取到 `telemetry/internal/otelresource` 公共包，避免重复，并避免 `telemetry` 根包与子包之间形成 import cycle。

### 4.5 Phase 2 辅助 API（instrument.go）

在 OTel API 之上提供薄封装，降低业务接入成本，风格对齐 `trace/span.go`：

```go
// IncCounter 递增 Counter，属性可选
func IncCounter(ctx context.Context, name string, value int64, attrs ...attribute.KeyValue) error

// RecordHistogram 记录 Histogram 值
func RecordHistogram(ctx context.Context, name string, value float64, attrs ...attribute.KeyValue) error

// RecordDuration 记录耗时（自动转 seconds）
func RecordDuration(ctx context.Context, name string, duration time.Duration, attrs ...attribute.KeyValue) error

// InitInstruments 批量注册指标定义（启动时调用）
func InitInstruments(defs ...InstrumentDef) error

// MustInitInstruments 批量注册指标定义，失败时 panic
func MustInitInstruments(defs ...InstrumentDef)
```

**InstrumentDef 示例**：

```go
type InstrumentDef struct {
    Name        string
    Kind        InstrumentKind  // Counter | Histogram
    Description string
    Unit        string
}
```

辅助 API 内部维护 `map[string]metric.Int64Counter` 等缓存，避免业务重复创建 Instrument。为防止动态 name 导致 Instrument 爆炸，`IncCounter` / `RecordHistogram` / `RecordDuration` 只允许写入已通过 `InitInstruments` / `MustInitInstruments` 注册的 name；未知 name 返回错误，不在请求路径动态创建 Instrument。

---

## 5. 支持的指标类型

基于 OTel Metrics API，完整支持以下 Instrument：

| 类型 | OTel API | 典型场景 | 导出形态 |
|------|----------|----------|----------|
| Counter | `Int64Counter` / `Float64Counter` | 请求数、错误数、发送消息数 | Sum (monotonic) |
| UpDownCounter | `Int64UpDownCounter` | 连接池大小、队列深度 | Sum (non-monotonic) |
| Histogram | `Float64Histogram` / `Int64Histogram` | 延迟、响应大小 | Histogram |
| Gauge（同步） | `Float64Gauge` / `Int64Gauge` (OTel v1.35+) | 当前内存、温度 | Gauge |
| Observable Gauge | `Int64ObservableGauge` | CPU 使用率、在线用户数 | Gauge |
| Observable Counter | `Int64ObservableCounter` | 累计值只读观测 | Sum |

**Temporality / Aggregation**：使用 OTel SDK 默认选择器（`DefaultTemporalitySelector` / `DefaultAggregationSelector`），与 ar_metric 相同，不在本模块重复实现。

**ExponentialHistogram**：OTel SDK / Exporter 支持相关数据结构，但是否产生 ExponentialHistogram 取决于 View / Aggregation 配置。Phase 1 不主动配置，沿用默认 Explicit Bucket Histogram。

---

## 6. 命名与语义约定

遵循 [OpenTelemetry 语义约定](https://opentelemetry.io/docs/specs/semconv/)：

### 6.1 命名规范

```
{namespace}.{subsystem}.{name}.{unit?}
```

示例：

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `http.server.request.duration` | Histogram | 请求耗时（秒） |
| `http.server.active_requests` | UpDownCounter | 活跃请求数 |
| `db.client.operation.duration` | Histogram | DB 操作耗时 |
| `messaging.publish.count` | Counter | 消息发布数 |
| `business.order.created.count` | Counter | 业务：订单创建数 |

### 6.2 属性（Labels）规范

- 使用 OTel `attribute.KeyValue`，key 采用 dot 命名：`http.request.method`、`http.response.status_code`、`db.system`
- **高基数属性禁止作为 label**：`user_id`、`trace_id`、`request_id`（会导致指标爆炸）
- 错误指标统一用 `error.type` 或 `http.response.status_code` 而非完整 error message
- HTTP 请求数不单独定义 `http.server.request.count`；优先使用 `http.server.request.duration` Histogram 的 count。确需独立 Counter 时，应放入业务命名空间，如 `business.api.request.count`。

### 6.3 与 Trace 的关联（Exemplars）

OTel Metrics 支持 Exemplars，可将特定 Histogram 数据点关联到 TraceID。当前实现会把 HTTP 请求的 `r.Context()` 传给 `RecordHTTPServerRequest`，由 OTel SDK 根据当前 Context / Span 处理 exemplar；本模块不把 `trace_id` 或 `span_id` 手工写入指标 label，避免高基数。

---

## 7. HTTP 自动指标（Phase 3）

本项目基于 go-zero，HTTP 自动指标必须优先使用 go-zero 路由注册阶段的 `rest.Route.Path`，不要依赖请求阶段的 `r.URL.Path` 反推 route pattern。go-zero 原生 `PrometheusHandler(route.Path, route.Method)` 也是在 bind route 时捕获模板路径，这是正确方向。

### 7.1 go-zero 路由感知 API

新增 `middleware/metrics.go`，主推路由感知包装：

```go
// MetricsForRoute 创建绑定 route 模板和 method 的 go-zero middleware。
func MetricsForRoute(routePath, method string, opts ...MetricsOption) func(http.HandlerFunc) http.HandlerFunc

// WithMetrics 批量包装 go-zero routes，推荐在 handler.RegisterHandlers 中使用。
func WithMetrics(routes ...rest.Route) []rest.Route

// WithMetricsPrefix 配合 go-zero rest.WithPrefix 使用，保证 http.route 包含最终前缀。
func WithMetricsPrefix(prefix string, routes ...rest.Route) []rest.Route
```

使用方式：

```go
server.AddRoutes(middleware.WithMetrics([]rest.Route{
    {
        Method: http.MethodGet,
        Path:   "/api/users/:id",
        Handler: userHandler,
    },
}...))
```

若使用 go-zero `rest.WithPrefix("/api")`，应使用：

```go
server.AddRoutes(
    middleware.WithMetricsPrefix("/api", routes...),
    rest.WithPrefix("/api"),
)
```

**自动采集指标**：

| 指标 | 类型 | 属性 |
|------|------|------|
| `http.server.request.duration` | Histogram | `http.request.method`, `http.route`, `http.response.status_code`, `url.scheme` |
| `http.server.active_requests` | UpDownCounter | `http.request.method`, `http.route`, `url.scheme` |

**实现要点**：

- `WithMetrics` 应逐个 route 包装 handler，使 middleware 闭包捕获 `route.Path` 和 `route.Method`
- 使用 go-zero `rest.WithPrefix` 时，需通过 `WithMetricsPrefix` 把 prefix 同步进 metrics 闭包
- 默认不得使用 `r.URL.Path` 作为 `http.route` label；没有 route 模板时应省略 `http.route` 或记录为低基数占位值
- 响应状态码捕获可参考 go-zero `response.NewWithCodeResponseWriter` 的行为，但本仓库不能 import go-zero internal 包，应自行实现兼容 `http.ResponseWriter` / `http.Flusher` / `http.Hijacker` 的 wrapper
- 与 Trace 中间件独立，Metrics 关闭时不影响 Trace

### 7.2 全局中间件仅作兜底

保留可选全局 middleware，但不作为推荐接入方式：

```go
func Metrics(opts ...MetricsOption) func(http.HandlerFunc) http.HandlerFunc

// WithRouteNamer 显式提供低基数 route 模板，如 /api/users/:id。
func WithRouteNamer(fn func(*http.Request) string) MetricsOption
```

全局 `server.Use(middleware.Metrics())` 无法直接拿到 go-zero 注册模板，因为 go-zero router 只把 path 参数写入 request context，没有写入 route pattern。若未提供 `WithRouteNamer`，全局 middleware 不记录 `http.route`。

**中间件注册顺序建议**：

```
Recovery → RequestID → Trace → MetricsForRoute/WithMetrics → CORS → Logger → Auth
```

---

### 7.3 Go runtime 指标

`telemetry/metrics/runtime.go` 通过 `RuntimeEnabled` 控制，默认关闭。开启后注册以下 observable metrics：

| 指标 | 类型 | 说明 |
|------|------|------|
| `go.goroutine.count` | Observable UpDownCounter | 当前 goroutine 数 |
| `go.memory.allocated` | Observable Counter | 累计 heap 分配字节 |
| `go.memory.allocations` | Observable Counter | 累计 heap 分配次数 |
| `go.memory.gc.goal` | Observable UpDownCounter | GC heap 目标 |
| `go.memory.limit` | Observable UpDownCounter | Go runtime memory limit；无限制时记录 `0` |
| `go.memory.used` | Observable UpDownCounter | Go runtime 从系统获取的内存 |
| `go.processor.limit` | Observable UpDownCounter | 当前 `GOMAXPROCS` |

实现仅使用标准库 `runtime` / `runtime/debug`，不引入额外 runtime instrumentation 依赖。

---

## 8. 与 go-zero 观测体系的关系

go-zero v1.9.4 已内置多套观测能力：

- `RestConf.Middlewares.Prometheus`：默认 `true`，按 route 模板、method、code 暴露 HTTP Prometheus 指标
- `RestConf.Middlewares.Metrics`：默认 `true`，写入 go-zero `core/stat`，用于 QPS / 延迟统计日志或 `MetricsUrl` 远程上报
- `ServiceConf.DevServer.EnableMetrics`：默认 `true`，在 dev server 暴露 Prometheus `/metrics`
- `ServiceConf.Telemetry`：go-zero 自带 Trace agent，会设置全局 OTel TracerProvider

| 维度 | go-zero Prometheus | telemetry/metrics (OTel) |
|------|-------------------|--------------------------|
| 协议 | Prometheus Pull (HTTP scrape) | OTLP Push |
| 范围 | HTTP 服务框架级 | 任意业务指标 |
| 后端 | 直连 Prometheus | Collector → 任意后端 |
| 关联 Trace | 无 | 可支持 Exemplars |

**共存策略**：

1. **默认推荐**：保留 go-zero 原生 Prometheus / stat 作为 HTTP 框架级指标，`telemetry/metrics` 只负责业务指标和后续可选 OTel HTTP 指标。
2. **避免重复**：若启用 Phase 3 `middleware.WithMetrics(...)` 导出 OTel HTTP 指标，建议关闭 `RestConf.Middlewares.Prometheus` 或至少在告警/Dashboard 中二选一，避免同一 HTTP 延迟和请求量出现两套口径。
3. **Trace Provider 边界**：如果使用本仓库 `telemetry.Init()` 初始化 Trace，应禁用 go-zero `ServiceConf.Telemetry` agent，否则 `rest.MustNewServer(c.RestConf)` 触发的 `ServiceConf.SetUp()` 可能覆盖全局 OTel TracerProvider。
4. **DevServer 边界**：go-zero dev server `/metrics` 仍可用于 Prometheus scrape；OTLP Metrics 走 Collector，不依赖 dev server。
5. **长期方向**：OTel Metrics 是跨后端主路径，Prometheus 可通过 Collector 的 `prometheusremotewrite` 或 `prometheus` exporter 接收。

### 8.1 推荐配置形态

以下 YAML 片段表示 go-zero `RestConf` 相关字段。若业务配置匿名嵌入 `rest.RestConf`，这些字段通常在顶层；若业务配置命名为 `RestConf`，则放在对应子节点下。

**已有 Prometheus / go-zero 监控体系**：

```yaml
Middlewares:
  Prometheus: true
  Metrics: true

Telemetry:
  Metrics:
    Enabled: true  # 仅业务指标
```

**统一 OTel 指标体系**：

```yaml
Middlewares:
  Prometheus: false  # 避免 HTTP 指标重复
  Metrics: true      # 可保留 go-zero stat 日志，也可按团队规范关闭

Telemetry:
  Metrics:
    Enabled: true
```

**Trace 初始化约束（代码层推荐）**：

```go
// conf.MustLoad 后、rest.MustNewServer 前执行。
// 这里是 go-zero ServiceConf.Telemetry，不是本仓库 telemetry.Config。
c.RestConf.Telemetry.Disabled = true

// 如果继续使用本仓库 middleware.Trace()，还应关闭 go-zero 原生 Trace middleware，避免重复 Server Span。
c.RestConf.Middlewares.Trace = false
```

> 注意：如果业务配置里同时存在本仓库 `Telemetry` 字段和 go-zero `ServiceConf.Telemetry` 字段，建议重命名其中一个配置节点（例如本仓库使用 `Observability`），或在代码层显式设置 `c.RestConf.Telemetry.Disabled=true`，避免 YAML key 冲突导致 go-zero Trace agent 仍以默认值启动。

---

## 9. 依赖规划

### 9.1 新增直接依赖

```go
require (
    go.opentelemetry.io/otel/metric v1.39.0          // 已有 indirect，提升为 direct
    go.opentelemetry.io/otel/sdk/metric v1.39.0      // MeterProvider / PeriodicReader
    go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.39.0
    go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.39.0
)
```

版本与现有 `go.opentelemetry.io/otel v1.39.0` 对齐。

### 9.2 不引入的依赖

| 依赖 | 理由 |
|------|------|
| `gopsutil` | Phase 1 用标准库即可；主机信息 Phase 2 再评估 |
| `prometheus/client_golang` | 不重复造 Pull exporter，交给 go-zero 或 Collector |
| 自定义 HTTP Client | OTLP Exporter 已内置 |

---

## 10. 分阶段实施计划

### Phase 1 — 核心能力（MVP，约 3–5 天）

**交付物**：

- [x] `telemetry/metrics/metrics.go` — Init / Close / Meter
- [x] `telemetry/metrics/exporter.go` — OTLP gRPC/HTTP
- [x] `telemetry/metrics/config.go` — MetricsConfig
- [x] `telemetry/internal/otelresource/resource.go` — 公共 Resource 构建，Trace/Metrics 共用
- [x] `telemetry/config.go` — 增加 Metrics 字段
- [x] `telemetry/telemetry.go` — 集成 Init/Close
- [x] 单元测试：Init 开关、noop 模式、exporter 构建、ManualReader 聚合
- [x] `telemetry/metrics/README.md` — 使用文档

**验收标准**：

- 配置 `Metrics.Enabled: true` 后，业务代码可通过 `metrics.Meter()` 创建 Counter/Histogram 并导出
- 使用 OTel Collector + Prometheus exporter 可查到指标
- `Metrics.Enabled: false` 时使用 noop，零开销
- 与 Trace 共用 Collector endpoint 正常工作
- 不包含 HTTP 自动指标、运行时指标和业务辅助 API，避免 MVP 范围膨胀

### Phase 2 — 辅助 API 与 Resource 增强（约 2–3 天）

**交付物**：

- [x] `telemetry/metrics/instrument.go` — IncCounter / RecordHistogram / RecordDuration
- [x] `telemetry/internal/otelresource/resource.go` — Trace/Metrics 公共 Resource 构建
- [x] 主机/运行时 Resource 属性
- [x] 指标命名常量包（`metrics/semconv.go` 或引用 OTel semconv）

**验收标准**：

- 业务侧一行代码完成 Counter +1 或耗时记录
- Trace 与 Metrics 在同一后端可按 `service.name` 关联

### Phase 3 — HTTP 自动指标与运行时指标（约 2–3 天）

**交付物**：

- [x] `middleware/metrics.go` — HTTP 请求自动指标，主 API 为 `WithMetrics(routes ...rest.Route)` / `MetricsForRoute(routePath, method string)`
- [x] `telemetry/metrics/runtime.go` — Go runtime 指标（goroutine 数、内存、GOMAXPROCS 等，可选）
- [x] Exemplars 支持评估：使用 OTel SDK context 关联，不新增高基数 label

**验收标准**：

- 使用 `middleware.WithMetrics(...)` 包装 go-zero routes 后，无需业务代码即可看到 HTTP 请求耗时、请求量（Histogram count）和活跃请求数
- HTTP 指标中的 `http.route` 必须来自 `rest.Route.Path` 或显式 `WithRouteNamer`，不得使用 `r.URL.Path`
- 文档说明与 go-zero Prometheus 的选型建议

### Phase 4 — 可选增强（按需）

- [ ] 多 Reader 支持（同时 OTLP + stdout debug）
- [ ] 自定义 View（Histogram bucket 边界配置）
- [ ] 与 Audit 模块联动（如操作计数指标）

---

## 11. 关键实现参考

### 11.1 Init 伪代码

```go
func Init(config MetricsConfig, serviceName, version, environment string) error {
    if !config.Enabled {
        provider := noop.NewMeterProvider()
        otel.SetMeterProvider(provider)
        meterProvider = nil
        meter = provider.Meter(instrumentationName)
        logx.Info("Metrics 未启用")
        return nil
    }

    ctx := context.Background()
    exporter, err := newExporter(ctx, config)
    if err != nil {
        return err
    }

    res, err := newResource(serviceName, version, environment)
    if err != nil {
        return err
    }

    interval := time.Duration(config.Interval) * time.Second
    if interval <= 0 {
        interval = 15 * time.Second
    }
    timeout := time.Duration(config.Timeout) * time.Second
    if timeout <= 0 {
        timeout = 10 * time.Second
    }

    meterProvider = sdkmetric.NewMeterProvider(
        sdkmetric.WithResource(res),
        sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
            sdkmetric.WithInterval(interval),
            sdkmetric.WithTimeout(timeout),
        )),
    )

    otel.SetMeterProvider(meterProvider)
    meter = meterProvider.Meter(instrumentationName,
        metric.WithInstrumentationVersion(version),
    )

    logx.Infof("Metrics 初始化完成 [protocol=%s, endpoint=%s, interval=%s]",
        config.Protocol, config.Endpoint, interval)
    return nil
}
```

### 11.2 Exporter 构建（与 trace 对称）

```go
func newExporter(ctx context.Context, config MetricsConfig) (sdkmetric.Exporter, error) {
    timeout := time.Duration(config.Timeout) * time.Second
    if timeout <= 0 {
        timeout = 10 * time.Second
    }

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
        return otlpmetrichttp.New(ctx, opts...)
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
        return otlpmetricgrpc.New(ctx, opts...)
    }
}
```

---

## 12. 测试策略

| 层级 | 方法 |
|------|------|
| 单元测试 | `ManualReader` 替代 OTLP Exporter，验证 Counter/Histogram 聚合结果 |
| Exporter 测试 | 使用 `otlpreceiver` 内存 Collector 或 testcontainers |
| 集成测试 | docker-compose：服务 → otel-collector → prometheus |
| 性能测试 | 对比 Enabled/Disabled 的 benchmark，确保 noop 零开销 |

**示例：ManualReader 测试**

```go
func TestCounter(t *testing.T) {
    reader := sdkmetric.NewManualReader()
    provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
    meter := provider.Meter("test")

    counter, _ := meter.Int64Counter("test.counter")
    counter.Add(context.Background(), 1)

    var rm metricdata.ResourceMetrics
    reader.Collect(context.Background(), &rm)
    // assert sum == 1
}
```

---

## 13. 风险与应对

| 风险 | 影响 | 应对 |
|------|------|------|
| 高基数 Label | 存储爆炸、查询变慢 | 文档约束 + lint 建议；HTTP 指标通过 `WithMetrics(rest.Route...)` 捕获 route template |
| 与 go-zero metrics 重复 | 同一指标两套 | 文档明确选型；通过 `RestConf.Middlewares.Prometheus` 与 Phase 3 OTel HTTP 指标二选一 |
| go-zero Trace agent 覆盖本仓库 Trace | Trace 导出配置失效 | 使用本仓库 `telemetry.Init()` 时设置 `c.RestConf.Telemetry.Disabled=true` |
| OTLP Collector 未部署 | 指标暂时无法导出 | 默认 `Enabled=false`；启用后不做 Init 探活，依赖 Exporter retry |
| 进程退出时指标未导出 | 数据丢失 | Close 时 Shutdown 强制 flush；注册 signal handler |
| Instrument 动态创建或重复定义 | 内存增长、注册错误 | 启动阶段集中注册；辅助 API 只写已注册 name |

---

## 14. 本地开发验证

### 14.1 最小 Collector 配置

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

exporters:
  prometheus:
    endpoint: 0.0.0.0:8889
  debug:
    verbosity: detailed

service:
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [prometheus, debug]
    traces:
      receivers: [otlp]
      exporters: [debug]
```

```bash
docker run -d --name otel-collector \
  -p 4317:4317 -p 8889:8889 \
  -v $(pwd)/otel-collector-config.yaml:/etc/otelcol/config.yaml \
  otel/opentelemetry-collector-contrib:latest
```

访问 Prometheus metrics：`http://localhost:8889/metrics`

### 14.2 验证 Trace + Metrics 共存

同一 Collector 4317 端口同时接收 Trace 和 Metrics，验证 `{service_name="idrm-api"}` 指标与 Trace 均正常。

---

## 15. 对比总结

| 能力 | ar_metric | 本方案 telemetry/metrics |
|------|-----------|-------------------------|
| 导出协议 | AnyRobot JSON | **OTLP**（标准） |
| 后端 | AnyRobot Feed Ingester | **任意 OTLP 兼容后端** |
| Meter 获取 | 全局变量 | **Meter() 函数**（同 trace.Tracer()） |
| 生命周期 | 手动 Shutdown | **telemetry.Init/Close 统一管理** |
| 配置 | Client Option 分散 | **MetricsConfig 统一 YAML** |
| 传输层 | 自建 public.Client | **OTel 官方 Exporter** |
| 聚合/缓冲 | 无（立即发送） | **OTel SDK PeriodicReader** |
| HTTP 自动指标 | 无 | **Phase 3 route-aware middleware** |
| 辅助 API | 无 | **Phase 2 instrument.go** |
| 审计/Trace 集成 | 无 | **同 telemetry 包，Resource 一致** |

---

## 16. 开放问题（待评审）

1. **Interval 默认值**：15s 是否合适？高频业务是否需要更短？
2. **Timeout 是否拆分**：是否需要分别配置 Reader collect/export timeout 与 Exporter RPC timeout？
3. **go-zero Trace 中间件取舍**：是否长期保留本仓库 `middleware.Trace()`，还是改为推荐 go-zero 原生 `RestConf.Middlewares.Trace` + 本仓库 OTel Provider。
4. **指标命名前缀**：当前提供 `business.` 推荐常量；是否需要为本项目固化 `idrm.` 命名空间仍待团队确认。

**已决事项**：

- Resource 已在 Phase 1 抽到 `telemetry/internal/otelresource`，Trace 与 Metrics 共用同一构建逻辑。

---

## 附录 A：文件变更清单（Phase 1）

| 文件 | 变更类型 |
|------|----------|
| `telemetry/metrics/metrics.go` | 新增 |
| `telemetry/metrics/exporter.go` | 新增 |
| `telemetry/metrics/config.go` | 新增 |
| `telemetry/metrics/metrics_test.go` | 新增 |
| `telemetry/metrics/README.md` | 新增 |
| `telemetry/internal/otelresource/resource.go` | 新增（可选，若 Phase 1 抽公共 Resource） |
| `telemetry/config.go` | 修改：增加 MetricsConfig |
| `telemetry/telemetry.go` | 修改：Init/Close 集成 |
| `telemetry/README.md` | 修改：增加 Metrics 说明 |
| `go.mod` / `go.sum` | 修改：新增 OTel metric 依赖 |

## 附录 B：文件变更清单（Phase 2）

| 文件 | 变更类型 |
|------|----------|
| `telemetry/metrics/instrument.go` | 新增：业务指标预注册与记录辅助 API |
| `telemetry/metrics/semconv.go` | 新增：常用指标名与属性名常量 |
| `telemetry/internal/otelresource/resource.go` | 修改：补充 host / Go runtime Resource 属性 |
| `telemetry/metrics/metrics_test.go` | 修改：增加辅助 API 单元测试 |
| `telemetry/metrics/README.md` | 修改：增加辅助 API 使用示例 |

## 附录 C：文件变更清单（Phase 3）

| 文件 | 变更类型 |
|------|----------|
| `middleware/metrics.go` | 新增：go-zero route-aware HTTP metrics middleware |
| `middleware/metrics_test.go` | 新增：HTTP route/status metrics 单元测试 |
| `telemetry/metrics/http.go` | 新增：HTTP 标准指标 instruments 与记录函数 |
| `telemetry/metrics/runtime.go` | 新增：Go runtime observable metrics |
| `telemetry/metrics/config.go` | 修改：增加 `RuntimeEnabled` |
| `telemetry/config.go` | 修改：透出 `RuntimeEnabled` |
| `telemetry/telemetry.go` | 修改：传递 `RuntimeEnabled` |
| `telemetry/metrics/README.md` | 修改：增加 HTTP/runtime 使用文档 |
| `middleware/README.md` | 修改：增加 Metrics 中间件说明 |

## 附录 D：参考资料

- [OpenTelemetry Metrics API](https://opentelemetry.io/docs/specs/otel/metrics/api/)
- [OTel Go Metrics SDK](https://opentelemetry.io/docs/languages/go/instrumentation/#metrics)
- [OTLP Metric Exporter](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc)
- [OpenTelemetry 语义约定 — HTTP](https://opentelemetry.io/docs/specs/semconv/http/)
- 本仓库 `telemetry/trace/` 模块（配置与生命周期参考）
