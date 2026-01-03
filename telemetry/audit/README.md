# Telemetry 审计日志系统（Phase 3）

## 📋 概述

独立的审计日志系统，记录关键业务操作，支持操作前后数据对比和 TraceID 关联。

## ✨ 功能特性

- ✅ **结构化日志**：标准化的审计日志格式
- ✅ **操作追踪**：记录操作前后数据
- ✅ **链路关联**：自动提取 TraceID
- ✅ **用户信息**：记录操作用户和 IP
- ✅ **批量发送**：高性能异步上报
- ✅ **Fluent API**：便捷的链式调用

## ⚙️ 配置

### 配置结构

```go
type AuditConfig struct {
    Enabled bool   // 是否启用
    Url     string // 审计服务地址
    Buffer  int    // 缓冲区大小
}
```

### 配置示例

```yaml
# api/etc/api.yaml
Telemetry:
  Audit:
    Enabled: true
    Url: http://audit-service:8080/api/audit
    Buffer: 100
```

## 🚀 使用方法

### 1. 初始化

```go
import (
    "idrm/pkg/telemetry/audit"
)

func main() {
    // 初始化审计日志
    audit.Init(config.Telemetry.Audit, config.Telemetry.ServiceName)
    defer audit.Close()
    
    // 业务代码...
}
```

### 2. 基础使用

#### 方式一：直接使用 Log

```go
import (
    "idrm/pkg/telemetry/audit"
)

func createCategory(ctx context.Context, req *Req) error {
    // 业务逻辑...
    
    // 记录审计日志
    audit.Log(ctx, audit.AuditLog{
        Action:   audit.ActionCreate,
        Resource: audit.ResourceCategory,
        UserID:   "user123",
        Username: "admin",
        IP:       "127.0.0.1",
        After:    category,
        Success:  true,
    })
    
    return nil
}
```

#### 方式二：使用 Helper（推荐）

```go
func createCategory(ctx context.Context, req *Req) error {
    // 创建审计辅助器
    auditLog := audit.NewHelper(ctx).
        WithAction(audit.ActionCreate).
        WithResource(audit.ResourceCategory).
        WithUser("user123", "admin").
        WithIP("127.0.0.1")
    
    // 业务逻辑
    category, err := service.Create(req)
    if err != nil {
        auditLog.Fail(err)
        return err
    }
    
    // 记录成功
    auditLog.WithAfter(category).Success()
    return nil
}
```

### 3. 常用场景

#### 创建操作

```go
func CreateCategory(ctx context.Context, req *CreateReq) error {
    auditLog := audit.NewHelper(ctx).
        WithAction(audit.ActionCreate).
        WithResource(audit.ResourceCategory).
        WithUser(getUserID(ctx), getUsername(ctx)).
        WithIP(getIP(ctx))
    
    category, err := l.svcCtx.CategoryModel.Insert(ctx, data)
    
    auditLog.WithAfter(category).SuccessOrFail(err)
    return err
}
```

#### 更新操作

```go
func UpdateCategory(ctx context.Context, req *UpdateReq) error {
    // 获取操作前数据
    before, _ := l.svcCtx.CategoryModel.FindOne(ctx, req.Id)
    
    auditLog := audit.NewHelper(ctx).
        WithAction(audit.ActionUpdate).
        WithResource(audit.ResourceCategory).
        WithUser(getUserID(ctx), getUsername(ctx)).
        WithBefore(before)
    
    // 执行更新
    err := l.svcCtx.CategoryModel.Update(ctx, data)
    
    // 获取操作后数据
    after, _ := l.svcCtx.CategoryModel.FindOne(ctx, req.Id)
    
    auditLog.WithAfter(after).SuccessOrFail(err)
    return err
}
```

#### 删除操作

```go
func DeleteCategory(ctx context.Context, id int64) error {
    // 记录删除前的数据
    before, _ := l.svcCtx.CategoryModel.FindOne(ctx, id)
    
    auditLog := audit.NewHelper(ctx).
        WithAction(audit.ActionDelete).
        WithResource(audit.ResourceCategory).
        WithUser(getUserID(ctx), getUsername(ctx)).
        WithBefore(before)
    
    err := l.svcCtx.CategoryModel.Delete(ctx, id)
    
    auditLog.SuccessOrFail(err)
    return err
}
```

#### 查询操作

```go
func QueryCategories(ctx context.Context, req *QueryReq) error {
    auditLog := audit.NewHelper(ctx).
        WithAction(audit.ActionQuery).
        WithResource(audit.ResourceCategory).
        WithUser(getUserID(ctx), getUsername(ctx)).
        WithExtra("page", req.Page).
        WithExtra("page_size", req.PageSize)
    
    categories, total, err := l.svcCtx.CategoryModel.List(ctx, req.Page, req.PageSize)
    
    auditLog.WithExtra("total", total).SuccessOrFail(err)
    return err
}
```

#### 登录/登出

```go
// 登录
func Login(ctx context.Context, username, password string) error {
    auditLog := audit.NewHelper(ctx).
        WithAction(audit.ActionLogin).
        WithResource(audit.ResourceUser).
        WithUser("", username).
        WithIP(getIP(ctx))
    
    user, err := authenticate(username, password)
    
    if err != nil {
        auditLog.WithExtra("reason", "invalid_credentials").Fail(err)
        return err
    }
    
    auditLog.WithUser(user.ID, user.Username).Success()
    return nil
}

// 登出
func Logout(ctx context.Context) error {
    audit.NewHelper(ctx).
        WithAction(audit.ActionLogout).
        WithResource(audit.ResourceUser).
        WithUser(getUserID(ctx), getUsername(ctx)).
        Success()
    
    return nil
}
```

### 4. HTTP 请求信息

```go
func HandleRequest(ctx context.Context, req *http.Request) error {
    auditLog := audit.NewHelper(ctx).
        WithAction(audit.ActionCreate).
        WithResource(audit.ResourceCategory).
        WithRequest(req)  // 自动提取 Method, Path, IP
    
    // 业务逻辑...
    
    auditLog.Success()
    return nil
}
```

### 5. 扩展字段

```go
func ProcessOrder(ctx context.Context, order Order) error {
    auditLog := audit.NewHelper(ctx).
        WithAction(audit.ActionCreate).
        WithResource("order").
        WithUser(order.UserID, order.Username).
        WithExtra("order_id", order.ID).
        WithExtra("amount", order.Amount).
        WithExtra("items_count", len(order.Items))
    
    err := processOrder(order)
    
    auditLog.SuccessOrFail(err)
    return err
}
```

## 📝 完整示例

```go
// api/internal/logic/category/createcategorylogic.go
package category

import (
    "context"
    
    "idrm/api/internal/svc"
    "idrm/api/internal/types"
    "idrm/pkg/telemetry/audit"
    "idrm/pkg/telemetry/trace"
    
    "github.com/zeromicro/go-zero/core/logx"
)

type CreateCategoryLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func (l *CreateCategoryLogic) CreateCategory(req *types.CreateCategoryReq) (*types.CategoryResp, error) {
    // 1. 创建链路追踪
    ctx, span := trace.StartInternal(l.ctx)
    defer span.End()
    
    // 2. 创建审计日志
    auditLog := audit.NewHelper(ctx).
        WithAction(audit.ActionCreate).
        WithResource(audit.ResourceCategory).
        WithUser(l.getUserID(), l.getUsername()).
        WithIP(l.getIP())
    
    // 3. 记录日志
    logx.WithContext(ctx).Infow("创建类别",
        logx.Field("name", req.Name),
        logx.Field("code", req.Code))
    
    // 4. 业务逻辑
    data := &resource_catalog.Category{
        Name:        req.Name,
        Code:        req.Code,
        ParentId:    req.ParentId,
        Description: req.Description,
    }
    
    category, err := l.svcCtx.CategoryModel.Insert(ctx, data)
    if err != nil {
        logx.WithContext(ctx).Errorf("创建类别失败: %v", err)
        trace.SetError(span, err)
        auditLog.Fail(err)
        return nil, err
    }
    
    // 5. 记录审计日志
    auditLog.WithAfter(category).Success()
    
    return &types.CategoryResp{
        Id:   category.Id,
        Name: category.Name,
        Code: category.Code,
    }, nil
}

func (l *CreateCategoryLogic) getUserID() string {
    // 从 context 提取用户ID
    return "user123"
}

func (l *CreateCategoryLogic) getUsername() string {
    // 从 context 提取用户名
    return "admin"
}

func (l *CreateCategoryLogic) getIP() string {
    // 从 context 提取IP
    return "127.0.0.1"
}
```

## 📊 审计日志格式

发送到审计服务的日志格式：

```json
{
  "audit_logs": [
    {
      "timestamp": "2024-01-01T12:00:00Z",
      "service_name": "idrm-api",
      "action": "create",
      "resource": "category",
      "user_id": "user123",
      "username": "admin",
      "ip": "127.0.0.1",
      "method": "POST",
      "path": "/api/v1/category",
      "trace_id": "abc123def456",
      "before": null,
      "after": {
        "id": 1,
        "name": "测试类别",
        "code": "TEST001"
      },
      "success": true,
      "error": "",
      "duration": 120,
      "extra": {
        "note": "首次创建"
      }
    }
  ]
}
```

## 🎯 最佳实践

### 1. 记录关键操作

```go
// ✅ 需要记录审计
- 创建/更新/删除数据
- 用户登录/登出
- 权限变更
- 配置修改
- 数据导出/导入

// ❌ 不需要记录
- 普通查询
- 健康检查
- 静态资源访问
```

### 2. 记录操作前后数据

```go
// ✅ 更新和删除操作记录前后数据
auditLog.WithBefore(oldData).WithAfter(newData)

// ✅ 创建操作只记录后数据
auditLog.WithAfter(newData)
```

### 3. 保护敏感信息

```go
// ❌ 不要记录敏感信息
auditLog.WithAfter(map[string]interface{}{
    "password": user.Password,  // ❌
    "token": user.Token,         // ❌
})

// ✅ 过滤敏感字段
auditLog.WithAfter(map[string]interface{}{
    "id": user.ID,
    "username": user.Username,
    "password": "***",  // 脱敏
})
```

### 4. 使用常量

```go
// ✅ 使用预定义常量
audit.ActionCreate
audit.ResourceCategory

// ❌ 避免硬编码字符串
"create"
"category"
```

## 🔧 工作原理

```
业务代码
  ↓
audit.NewHelper()
  ↓
WithAction/WithResource/...
  ↓
Success/Fail
  ↓
缓冲区 (Buffer)
  ↓
批量发送 (每10秒或100条)
  ↓
HTTP POST
  ↓
审计服务
```

## ⚡ 性能优化

1. **批量发送**：减少网络请求
2. **异步处理**：不阻塞业务
3. **自动刷新**：定时发送
4. **轻量级**：结构紧凑

## ❓ 常见问题

**Q: 审计日志和普通日志有什么区别？**

A: 审计日志专注于记录业务操作，包含操作前后数据对比，用于合规审计。普通日志用于调试和监控。

**Q: 如何关闭审计日志？**

A: 设置 `Audit.Enabled: false`

**Q: 审计日志失败会影响业务吗？**

A: 不会。审计日志是异步发送的，失败只会记录到本地日志。

**Q: 如何查询审计日志？**

A: 通过审计服务的 API 查询，可以根据用户、操作类型、时间范围等条件过滤。

**Q: TraceID 如何关联？**

A: 审计日志自动提取 Context 中的 TraceID，可以关联到链路追踪系统。

## 📚 参考资料

- [审计日志最佳实践](https://www.owasp.org/index.php/Logging_Cheat_Sheet)
- [GDPR 审计要求](https://gdpr-info.eu/)
