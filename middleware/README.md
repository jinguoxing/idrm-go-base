# Middleware 中间件使用文档

## 📋 已实现的中间件

本项目实现了完整的中间件栈，提供请求追踪、跨域、日志、异常恢复和链路追踪功能。

### 中间件列表

| 序号 | 中间件 | 文件 | 功能描述 |
|------|--------|------|---------|
| 1 | Recovery | `recovery.go` | 捕获 panic 并返回 500 错误 |
| 2 | RequestID | `requestid.go` | 生成唯一请求ID |
| 3 | Trace | `trace.go` | OpenTelemetry 链路追踪 |
| 4 | CORS | `cors.go` | 跨域资源共享 |
| 5 | Logger | `logger.go` | 请求日志记录 |

---

## 🚀 使用方式

### 全局注册（已配置）

在 `api/api.go` 中已按最佳顺序注册：

```go
server.Use(middleware.Recovery())   // 1. Panic recovery
server.Use(middleware.RequestID())  // 2. Request ID generation
server.Use(middleware.Trace())      // 3. OpenTelemetry tracing
server.Use(middleware.CORS())       // 4. CORS handling
server.Use(middleware.Logger())     // 5. Request logging
```

**顺序说明**：
1. **Recovery** 必须第一个，捕获后续所有 panic
2. **RequestID** 第二个，为请求生成唯一ID
3. **Trace** 第三个，创建 OpenTelemetry Span
4. **CORS** 处理跨域请求
5. **Logger** 最后，记录完整请求信息

---

## 📝 各中间件详解

### 1. Recovery - 异常恢复

**功能**：
- 捕获 panic
- 记录完整堆栈信息
- 返回统一的 500 错误响应

**日志示例**：
```json
{
  "level": "error",
  "error": "runtime error: invalid memory address",
  "stack": "goroutine 1 [running]....",
  "method": "POST",
  "path": "/api/v1/category",
  "request_id": "uuid-xxx"
}
```

---

### 2. RequestID - 请求追踪

**功能**：
- 从 `X-Request-ID` header 获取或生成新 UUID
- 注入到 Context
- 添加到响应 header

**使用示例**：
```go
// 在 Logic 中获取 RequestID
func (l *Logic) Handle(req *Req) {
    requestID := middleware.GetRequestID(l.ctx)
    logx.Infof("Request ID: %s", requestID)
}
```

**HTTP Headers**：
```
Request:  X-Request-ID: abc-123
Response: X-Request-ID: abc-123
```

---

### 3. Trace - 链路追踪

**功能**：
- 自动创建 OpenTelemetry Server Span
- 记录 HTTP 元数据（method, url, status, etc）
- 关联 RequestID
- 自动标记错误（status >= 400）

**Span 属性**：
```go
http.method: POST
http.url: http://localhost:8888/api/v1/category
http.status_code: 200
http.user_agent: Mozilla/5.0...
http.client_ip: 127.0.0.1
http.request_id: uuid-xxx
```

**在 Logic 中创建子 Span**：
```go
func (l *Logic) Handle(req *Req) {
    ctx, span := trace.StartInternal(l.ctx)
    defer span.End()
    
    // 业务逻辑...
    
    if err != nil {
        trace.SetError(span, err)
        return err
    }
}
```

---

### 4. CORS - 跨域支持

**功能**：
- 支持所有来源 (`*`)
- 允许常用 HTTP 方法
- 支持自定义 Headers
- 处理 OPTIONS 预检请求

**Headers 设置**：
```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS, PATCH
Access-Control-Allow-Headers: Content-Type, Authorization, X-Request-ID
Access-Control-Expose-Headers: X-Request-ID
Access-Control-Max-Age: 86400
```

**自定义配置**：
如需限制来源，修改 `cors.go`:
```go
w.Header().Set("Access-Control-Allow-Origin", "https://yourdomain.com")
```

---

### 5. Logger - 请求日志

**功能**：
- 记录所有 HTTP 请求
- 包含请求耗时
- 关联 RequestID
- 使用 go-zero logx 格式

**日志字段**：
```json
{
  "level": "info",
  "method": "POST",
  "path": "/api/v1/category",
  "query": "page=1&size=10",
  "status": 200,
  "duration_ms": 125,
  "remote_addr": "127.0.0.1:50123",
  "user_agent": "Mozilla/5.0...",
  "request_id": "uuid-xxx"
}
```

---

## 🔍 调试和监控

### 查看日志

```bash
# 实时查看日志
tail -f logs/access.log

# 过滤错误日志
grep "error" logs/error.log

# 查看特定请求
grep "uuid-xxx" logs/*.log
```

### Jaeger 链路追踪

1. 访问 Jaeger UI: http://localhost:16686
2. 选择服务: `idrm-api`
3. 搜索 Trace ID
4. 查看完整调用链

### 请求示例

```bash
# 发送请求
curl -H "X-Request-ID: test-123" \
     http://localhost:8888/api/v1/categories

# 响应 Headers 包含
# X-Request-ID: test-123

# 日志中可以看到
# request_id: test-123

# Jaeger 中可以搜索
# http.request_id: test-123
```

---

## ⚙️ 配置说明

### Trace 配置

在 `api/etc/api.yaml`:

```yaml
Telemetry:
  Trace:
    Enabled: true
    Endpoint: jaeger:4317  # OTLP gRPC endpoint
    Sampler: 1.0           # 采样率 (1.0 = 100%)
```

**采样率建议**：
- 开发环境: `1.0` (100%)
- 测试环境: `0.5` (50%)
- 生产环境: `0.1` (10%)

---

## 🎯 最佳实践

### 1. 错误处理

```go
func (l *Logic) Handle(ctx context.Context) error {
    ctx, span := trace.StartInternal(ctx)
    defer span.End()
    
    err := doSomething()
    if err != nil {
        // 记录到 span
        trace.SetError(span, err)
        
        // 记录到日志
        logx.WithContext(ctx).Errorf("操作失败: %v", err)
        
        return err
    }
    
    return nil
}
```

### 2. 性能监控

通过日志的 `duration_ms` 字段监控接口性能：

```bash
# 查找慢请求 (>1秒)
grep "duration_ms.*[0-9]\{4,\}" logs/access.log
```

### 3. 请求追踪

通过 RequestID 追踪完整请求链路：

```bash
# 追踪特定请求
grep "abc-123" logs/*.log
```

---

## 📚 扩展中间件

### 添加认证中间件

创建 `pkg/middleware/auth.go`:

```go
package middleware

import (
    "net/http"
    "github.com/zeromicro/go-zero/rest/httpx"
)

func Auth(secretKey string) func(http.HandlerFunc) http.HandlerFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            if token == "" {
                httpx.Error(w, errors.New("unauthorized"))
                return
            }
            
            // 验证 token...
            
            next(w, r)
        }
    }
}
```

在 `api.go` 中注册：

```go
server.Use(middleware.Auth(c.Auth.AccessSecret))
```

---

## ❓ 常见问题

**Q: 中间件顺序为什么重要？**

A: 中间件按注册顺序执行。Recovery 必须第一个才能捕获后续中间件的 panic。

**Q: 如何禁用某个中间件？**

A: 在 `api.go` 中注释掉对应的 `server.Use()` 行。

**Q: CORS 如何限制特定域名？**

A: 修改 `cors.go` 中的 `Access-Control-Allow-Origin` header。

**Q: 如何查看 Trace 数据？**

A: 访问 Jaeger UI (http://localhost:16686) 查看链路追踪。

---

## ✅ 验证

启动服务后，发送测试请求：

```bash
curl -v -H "X-Request-ID: test-001" \
     http://localhost:8888/api/v1/categories
```

检查：
1. ✅ 响应 Header 包含 `X-Request-ID: test-001`
2. ✅ 日志文件中有请求记录
3. ✅ Jaeger 中可以搜索到 Trace
4. ✅ OPTIONS 请求返回 204

---

## 🎉 完成

中间件栈已完整集成，提供：
- ✅ 请求追踪
- ✅ 链路追踪
- ✅ 跨域支持
- ✅ 请求日志
- ✅ 异常恢复

享受完整的可观测性！
