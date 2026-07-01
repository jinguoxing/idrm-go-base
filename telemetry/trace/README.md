# Telemetry 链路追踪系统（Phase 2）

## 📋 概述

基于 OpenTelemetry 标准的链路追踪系统，使用 OTLP 协议发送到后端（Jaeger/Zipkin/自定义）。

## ✨ 功能特性

- ✅ **OpenTelemetry 标准**：完全遵循 OTEL 规范
- ✅ **OTLP 导出器**：支持 gRPC / HTTP 协议
- ✅ **分布式传播**：HTTP Extract/Inject + gRPC StatsHandler
- ✅ **自动 Span 创建**：多种 Span 类型支持
- ✅ **代码位置追踪**：`StartInternal` 自动提取调用位置
- ✅ **ParentBased 采样**：尊重上游采样决策
- ✅ **生产就绪**：TLS、认证 Header、导出重试

## ⚙️ 配置

### 配置结构

```go
type TraceConfig struct {
    Enabled    bool              // 是否启用
    Endpoint   string            // OTLP endpoint
    Protocol   string            // grpc（默认）或 http
    Sampler    float64           // 采样率 0.0-1.0
    TLSEnabled bool              // 是否启用 TLS
    Headers    map[string]string // Collector 认证头等
}
```

### 配置示例

```yaml
# api/etc/api.yaml
Telemetry:
  ServiceName: idrm-api
  ServiceVersion: 1.0.0
  Environment: dev
  
  Trace:
    Enabled: true
    Endpoint: localhost:4317  # Jaeger OTLP gRPC
    Protocol: grpc
    Sampler: 1.0              # 100% 采样
    TLSEnabled: false          # 生产环境连接 Collector 时设为 true
```

### 常用后端配置

**Jaeger**:
```yaml
Endpoint: localhost:4317  # OTLP gRPC (推荐)
```

**OTLP HTTP Collector**:
```yaml
Endpoint: localhost:4318
Protocol: http
```

**生产环境 Collector（TLS + Token）**:
```yaml
Endpoint: collector.example.com:4317
Protocol: grpc
TLSEnabled: true
Headers:
  Authorization: "Bearer <token>"
```

**自定义 OTLP Collector**:
```yaml
Endpoint: collector.example.com:4317
```

## 🚀 使用方法

### 1. 初始化

```go
import (
    "github.com/jinguoxing/idrm-go-base/telemetry/trace"
)

func main() {
    // 初始化链路追踪
    err := trace.Init(
        config.Telemetry.Trace,
        config.Telemetry.ServiceName,
        config.Telemetry.ServiceVersion,
        config.Telemetry.Environment,
    )
    if err != nil {
        panic(err)
    }
    defer trace.Close(context.Background())
    
    // 业务代码...
}
```

### 2. 基础使用

#### 创建 Span

```go
import (
    "idrm/pkg/telemetry/trace"
    "go.opentelemetry.io/otel/attribute"
)

func processRequest(ctx context.Context) error {
    // 方式一：普通 Span
    ctx, span := trace.Start(ctx, "processRequest")
    defer span.End()
    
    // 添加属性
    span.SetAttributes(
        attribute.String("user_id", "123"),
        attribute.Int("count", 10),
    )
    
    // 业务逻辑...
    return nil
}
```

#### 自动提取调用信息

```go
func handleRequest(ctx context.Context) error {
    // 自动提取函数名、文件路径、行号
    ctx, span := trace.StartInternal(ctx)
    defer span.End()
    
    // Span 名称自动设置为: logic.handleRequest
    // 自动添加属性: code.filepath, code.lineno, code.function
    
    // 业务逻辑...
    return nil
}
```

### 3. 不同类型的 Span

#### Server Span（HTTP/RPC 请求）

```go
func handleHTTPRequest(ctx context.Context, req *http.Request) {
    ctx, span := trace.StartServer(ctx, req.URL.Path,
        attribute.String("http.method", req.Method),
        attribute.String("http.url", req.URL.String()),
    )
    defer span.End()
    
    // 处理请求...
}
```

#### Client Span（调用外部服务）

推荐方式一：使用出站 HTTP 辅助方法（自动创建 Client Span + Inject）：

```go
func callExternalAPI(ctx context.Context, url string) error {
    req, span, err := trace.NewOutboundHTTPRequest(ctx, http.MethodGet, url, nil)
    if err != nil {
        return err
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        trace.FinishHTTPClient(span, 0, err)
        return err
    }
    defer resp.Body.Close()

    trace.FinishHTTPClient(span, resp.StatusCode, nil)
    return nil
}
```

方式二：手动创建 Client Span：

```go
func callExternalAPI(ctx context.Context, url string) error {
    ctx, span := trace.StartClient(ctx, "CallExternalAPI",
        attribute.String("http.url", url),
    )
    defer span.End()

    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    trace.InjectHTTP(ctx, req.Header)
    // 发送 HTTP 请求...
    return nil
}
```

#### gRPC 服务间调用

客户端（go-zero zrpc 示例）：

```go
conn := grpc.Dial(target,
    trace.GRPCClientHandler(),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
```

服务端：

```go
server := grpc.NewServer(trace.GRPCServerHandler())
```

#### Consumer/Producer Span（消息队列）

```go
// 生产者
func publishMessage(ctx context.Context, topic string, msg []byte) error {
    ctx, span := trace.StartProducer(ctx, "PublishMessage",
        attribute.String("messaging.destination", topic),
        attribute.Int("messaging.message_size", len(msg)),
    )
    defer span.End()
    
    // 发送消息...
    return nil
}

// 消费者
func consumeMessage(ctx context.Context, msg Message) error {
    ctx, span := trace.StartConsumer(ctx, "ConsumeMessage",
        attribute.String("messaging.destination", msg.Topic),
    )
    defer span.End()
    
    // 处理消息...
    return nil
}
```

### 4. 分布式传播

入站 HTTP 请求由 `middleware.Trace()` 自动 Extract。手动场景：

```go
ctx := trace.ExtractHTTP(r.Context(), r.Header)
```

出站 HTTP：

```go
trace.InjectHTTP(ctx, req.Header)
```

> **注意**：使用中间件前必须先调用 `telemetry.Init()`，否则 Propagator 未注册，Extract/Inject 无效。

### 5. 错误处理

#### 使用 End 辅助方法

```go
func processData(ctx context.Context) error {
    ctx, span := trace.StartInternal(ctx)
    defer func() {
        // 自动记录错误状态
        trace.End(span, err)
    }()
    
    var err error
    // 业务逻辑...
    if err != nil {
        return err
    }
    
    return nil
}
```

#### 手动记录错误

```go
func processData(ctx context.Context) error {
    ctx, span := trace.StartInternal(ctx)
    defer span.End()
    
    err := doSomething()
    if err != nil {
        // 记录错误
        trace.SetError(span, err)
        return err
    }
    
    return nil
}
```

### 5. 添加事件和属性

```go
func processOrder(ctx context.Context, order Order) error {
    ctx, span := trace.StartInternal(ctx)
    defer span.End()
    
    // 添加属性
    trace.SetAttributes(span,
        attribute.String("order.id", order.ID),
        attribute.Float64("order.amount", order.Amount),
    )
    
    // 添加事件
    trace.AddEvent(span, "OrderValidated",
        attribute.String("status", "valid"),
    )
    
    // 业务逻辑...
    
    trace.AddEvent(span, "OrderProcessed")
    
    return nil
}
```

### 6. 获取 TraceID 和 SpanID

```go
func logRequest(ctx context.Context) {
    traceID := trace.GetTraceID(ctx)
    spanID := trace.GetSpanID(ctx)
    
    logx.WithContext(ctx).Infow("处理请求",
        logx.Field("trace_id", traceID),
        logx.Field("span_id", spanID))
}
```

## 📝 完整示例

### Logic 层使用

```go
// api/internal/logic/category/createcategorylogic.go
package category

import (
    "context"
    
    "idrm/api/internal/svc"
    "idrm/api/internal/types"
    "idrm/pkg/telemetry/trace"
    
    "github.com/zeromicro/go-zero/core/logx"
    "go.opentelemetry.io/otel/attribute"
)

type CreateCategoryLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func (l *CreateCategoryLogic) CreateCategory(req *types.CreateCategoryReq) (*types.CategoryResp, error) {
    // 1. 创建 Span（自动提取函数信息）
    ctx, span := trace.StartInternal(l.ctx)
    defer span.End()
    
    // 2. 添加业务属性
    span.SetAttributes(
        attribute.String("category.name", req.Name),
        attribute.String("category.code", req.Code),
    )
    
    // 3. 记录日志（自动包含 trace_id）
    logx.WithContext(ctx).Infow("创建类别",
        logx.Field("name", req.Name))
    
    // 4. 验证（子 Span）
    if err := l.validateRequest(ctx, req); err != nil {
        trace.SetError(span, err)
        return nil, err
    }
    
    // 5. 数据库操作
    category, err := l.svcCtx.CategoryModel.Insert(ctx, data)
    if err != nil {
        trace.SetError(span, err)
        return nil, err
    }
    
    // 6. 添加事件
    trace.AddEvent(span, "CategoryCreated",
        attribute.Int64("category.id", category.Id))
    
    return &types.CategoryResp{...}, nil
}

func (l *CreateCategoryLogic) validateRequest(ctx context.Context, req *types.CreateCategoryReq) error {
    // 创建子 Span
    ctx, span := trace.Start(ctx, "ValidateRequest")
    defer span.End()
    
    // 验证逻辑...
    return nil
}
```

## 🔍 Span 层级示例

```
processRequest                           [Server Span]
├── validateRequest                      [Internal Span]
├── CategoryModel.Insert                 [Internal Span]
│   └── SQL Query                        [自动创建]
└── callExternalAPI                      [Client Span]
    └── HTTP Request                     [自动创建]
```

## 🎯 最佳实践

### 1. Span 命名

```go
// ✅ 好的命名：清晰、具体
trace.Start(ctx, "CreateCategory")
trace.Start(ctx, "ValidateUserInput")
trace.Start(ctx, "QueryDatabase")

// ❌ 不好的命名：模糊、通用
trace.Start(ctx, "process")
trace.Start(ctx, "handle")
```

### 2. 属性使用

```go
// ✅ 使用有意义的属性
span.SetAttributes(
    attribute.String("user.id", userID),
    attribute.String("order.id", orderID),
    attribute.Int("item.count", count),
)

// ❌ 避免敏感信息
span.SetAttributes(
    attribute.String("password", pwd),        // ❌
    attribute.String("credit_card", card),    // ❌
)
```

### 3. 错误处理

```go
// ✅ 始终记录错误
if err != nil {
    trace.SetError(span, err)
    return err
}

// ✅ 使用 defer + End
ctx, span := trace.StartInternal(ctx)
defer trace.End(span, err)
```

### 4. 控制 Span 粒度

```go
// ✅ 合理粒度：重要操作创建 Span
func processOrder(ctx context.Context) {
    ctx, span := trace.StartInternal(ctx)
    defer span.End()
    
    // 子操作
    validateOrder(ctx)  // 创建子 Span
    saveOrder(ctx)      // 创建子 Span
}

// ❌ 过细粒度：每个小函数都创建
func add(ctx context.Context, a, b int) int {
    ctx, span := trace.Start(ctx, "add")  // 不必要
    defer span.End()
    return a + b
}
```

## 🔧 与 go-zero 集成

go-zero 的 HTTP 服务会自动创建 Server Span，无需手动添加。

```go
// HTTP Handler 层不需要手动创建 Span
// go-zero 自动创建

// Logic 层创建内部 Span
func (l *Logic) Handle(req *Req) (*Resp, error) {
    ctx, span := trace.StartInternal(l.ctx)
    defer span.End()
    
    // 业务逻辑...
}
```

## 📊 查看链路数据

### 使用 Jaeger

1. 启动 Jaeger：
```bash
docker run -d --name jaeger \
  -p 4317:4317 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

2. 访问 UI：`http://localhost:16686`

3. 查询链路：选择服务名 `idrm-api`

## ❓ 常见问题

**Q: 如何关闭链路追踪？**

A: 设置 `Trace.Enabled: false`

**Q: 采样率如何设置？**

A: `Sampler: 1.0` 表示 100%，`0.5` 表示 50%。生产环境建议 0.1-0.5。

**Q: Span 太多会影响性能吗？**

A: 使用批量发送，影响很小。但避免在循环中创建过多 Span。

**Q: 如何在日志中关联 TraceID？**

A: 使用 `logx.WithContext(ctx)` 会自动提取 TraceID。

## 🆘 故障排除

**无法连接到 OTLP endpoint**:
- 检查 endpoint 配置是否正确
- 确认后端服务正在运行
- 检查网络连通性

**Span 未显示**:
- 检查采样率设置
- 确认 TraceProvider 已正确初始化
- 查看后端服务日志

## 📚 参考资料

- [OpenTelemetry 官方文档](https://opentelemetry.io/docs/)
- [OTLP 规范](https://opentelemetry.io/docs/specs/otlp/)
- [Jaeger 文档](https://www.jaegertracing.io/docs/)
