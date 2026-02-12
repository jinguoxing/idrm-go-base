# IDRM Go-Base Auth 使用指南

本模块 (`auth`) 提供了基于 Ory 生态的身份认证与访问控制组件，以及轻量级的 JWT 认证中间件。

## 📦 安装

确保在 `go.mod` 中引入了 `idrm-go-base`：

```bash
go get github.com/jinguoxing/idrm-go-base
```

## 🚀 快速开始

### 1. 使用 Ory Kratos 进行身份验证 (Session 模式)

适用于服务直接对接 Kratos，验证 Cookie 或 X-Session-Token。

```go
package main

import (
    "context"
    "fmt"
    "github.com/jinguoxing/idrm-go-base/auth/ory"
)

func main() {
    // 1. 配置
    cfg := ory.Config{
        KratosPublicURL: "http://127.0.0.1:4433", // Kratos Public API
    }

    // 2. 初始化管理器
    identityMgr := ory.NewKratosManager(cfg)

    // 3. 验证会话 (通常在中间件中进行)
    sessionToken := "ory_kratos_session_..." 
    session, err := identityMgr.ValidateSession(context.Background(), sessionToken)
    if err != nil {
        panic(err)
    }

    fmt.Printf("当前用户: %s (活跃: %v)\n", session.Identity.ID, session.Active)
}
```

### 2. 使用 Ory Keto 进行权限检查 (ReBAC)

适用于细粒度的权限控制 (Zanzibar 模型)。

```go
func checkPermission() {
    // 1. 配置
    cfg := ory.Config{
        KetoReadURL: "http://127.0.0.1:4466", // Keto Read API
    }

    // 2. 初始化引擎
    permEngine := ory.NewKetoEngine(cfg)

    // 3. 检查权限
    // 问: User "alice" 在 "files" 命名空间下 是否有权 "view" 对象 "report.pdf" ?
    allowed, err := permEngine.Check(context.Background(), 
        "files",       // Namespace
        "report.pdf",  // Object
        "view",        // Relation
        "alice",       // Subject ID
    )
    
    if allowed {
        fmt.Println("Access Granted")
    } else {
        fmt.Println("Access Denied")
    }
}
```

### 3. 使用 Oathkeeper 网关模式 (推荐)

最简单的集成方式。服务部署在 Oathkeeper 网关后面，直接信任网关传入的 Header。

**Go-Zero 中间件配置:**

```go
// internal/svc/service_context.go
import (
    "github.com/jinguoxing/idrm-go-base/auth/ory"
    "github.com/zeromicro/go-zero/rest"
)

type ServiceContext struct {
    // ...
    AuthMiddleware rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
    return &ServiceContext{
        // ...
        // 初始化中间件，自动从 X-User-Id 读取用户并注入 Context
        AuthMiddleware: ory.OathkeeperMiddleware(),
    }
}
```

**路由使用:**

```go
// internal/handler/routes.go
server.AddRoutes(
    []rest.Route{
        {
            Method:  http.MethodGet,
            Path:    "/user/profile",
            Handler: user.ProfileHandler(serverCtx),
        },
    },
    rest.WithMiddleware(serverCtx.AuthMiddleware), // 应用中间件
)
```

**Logic 层获取用户 ID:**

```go
import "github.com/jinguoxing/idrm-go-base/auth"

func (l *ProfileLogic) Profile(req *types.ProfileReq) (resp *types.ProfileResp, err error) {
    // 从 Context 获取用户 ID
    userID, ok := auth.UserIDFromContext(l.ctx)
    if !ok {
        return nil, errorx.NewCodeError(401, "未登录")
    }
    // ...
}
```

### 4. 使用 JWT 认证 (轻量级/本地模式)

如果不使用 Ory 全家桶，仅需简单的 JWT 签发与校验。

**签发 (Issue):**

```go
import "github.com/jinguoxing/idrm-go-base/auth"

cfg := auth.Config{
    Mode:         "local",
    AccessSecret: "your-secret-key-must-be-long",
    UserIDClaim:  "uid",
}

token, exp, err := auth.IssueToken(cfg, "user-123", "user@example.com", false)
```

**校验 (Middleware):**

```go
// 1. 创建校验器
validator, _ := auth.NewValidator(cfg)

// 2. 使用中间件 (MustJWT 强制校验, OptionalJWT 可选)
server.AddRoutes(..., rest.WithMiddleware(auth.MustJWT(validator)))
```

## ⚙️ 配置说明 (Ory)

在 `ory.Config` 中支持以下配置：

| 字段 | 说明 | 示例 |
| :--- | :--- | :--- |
| `KratosPublicURL` | Kratos Public API (用于验证 Session) | `http://kratos:4433` |
| `KetoReadURL` | Keto Read API (用于鉴权) | `http://keto:4466` |
| `HydraAdminURL` | Hydra Admin API (用于 OAuth 管理) | `http://hydra:4445` |

## ⚠️ 注意事项

1.  **Ory 版本**: 推荐使用 Ory Kratos v1.0+, Keto v0.11+, Hydra v2.0+。
2.  **网络隔离**: 若使用 `OathkeeperMiddleware`，务必确保服务无法被外部直接访问，否则攻击者可伪造 `X-User-Id` Header。
3.  **依赖**: 本模块通过 HTTP 调用 Ory 服务，不依赖繁重的 Ory SDK，保持了 `go-base` 的轻量性。
