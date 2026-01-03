# idrm-go-base

> **Go 通用包仓库** - IDRM 系列项目共享基础库

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

---

## 模块

| 模块 | 说明 |
|------|------|
| **errorx** | 错误码定义和业务错误 |
| **response** | 统一 HTTP 响应格式 |
| **middleware** | 中间件 (auth/cors/logger/recovery/trace) |
| **validator** | 参数校验 (支持中文错误) |
| **telemetry** | 遥测 (log/trace/audit) |

---

## 安装

```bash
go get github.com/jinguoxing/idrm-go-base@latest
```

---

## 使用

```go
import (
    "github.com/jinguoxing/idrm-go-base/errorx"
    "github.com/jinguoxing/idrm-go-base/response"
    "github.com/jinguoxing/idrm-go-base/middleware"
    "github.com/jinguoxing/idrm-go-base/validator"
    "github.com/jinguoxing/idrm-go-base/telemetry"
)
```

### errorx 示例

```go
// 使用预定义错误码
err := errorx.NewWithCode(errorx.ErrCodeNotFound)

// 自定义错误消息
err := errorx.New(errorx.ErrCodeBusiness, "用户不存在")
```

### response 示例

```go
// Handler 中使用
func (l *GetUserLogic) GetUser(req *types.GetUserReq) (resp *types.GetUserResp, err error) {
    user, err := l.svcCtx.UserModel.FindOne(l.ctx, req.Id)
    if err != nil {
        return nil, errorx.NewWithCode(errorx.ErrCodeNotFound)
    }
    return &types.GetUserResp{User: user}, nil
}

// httpx.SetErrorHandler 配置
httpx.SetErrorHandler(response.ErrorHandler)
```

### middleware 示例

```go
server := rest.MustNewServer(c.RestConf,
    rest.WithMiddlewares(
        middleware.AuthMiddleware(c.Auth.AccessSecret),
        middleware.LoggerMiddleware(),
        middleware.RecoveryMiddleware(),
    ),
)
```

---

## 错误码

| 范围 | 类型 |
|------|------|
| 10000-19999 | 系统错误 |
| 20000-29999 | 参数错误 |
| 30000-39999 | 业务错误 |
| 40000-49999 | 认证错误 |

---

## License

[MIT License](LICENSE)
