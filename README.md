# idrm-go-base

> **Go 通用包仓库** - IDRM 系列项目共享基础库

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

---

## 模块

| 模块 | 说明 |
|------|------|
| **db** | 数据库初始化 (GORM, go-zero sqlx) ✨ |
| **errorx** | 错误码定义和业务错误 |
| **response** | 统一 HTTP 响应格式 |
| **middleware** | 中间件 (auth/cors/logger/recovery/trace) |
| **validator** | 参数校验 (支持中文错误) |
| **telemetry** | 遥测 (log/trace/audit) |
| **uuid** | UUID v7 生成器 |

---

## 安装

```bash
go get github.com/jinguoxing/idrm-go-base@latest
```

---

## 使用

```go
import (
    "github.com/jinguoxing/idrm-go-base/db"
    "github.com/jinguoxing/idrm-go-base/errorx"
    "github.com/jinguoxing/idrm-go-base/response"
    "github.com/jinguoxing/idrm-go-base/middleware"
    "github.com/jinguoxing/idrm-go-base/validator"
    "github.com/jinguoxing/idrm-go-base/telemetry"
)
```

### db 模块（数据库）

支持三种数据库访问方式：

#### 1. GORM (推荐用于复杂查询)

```go
import "github.com/jinguoxing/idrm-go-base/db"

// 初始化
gormDB, err := db.InitGorm(db.Config{
    Host:          "localhost",
    Port:          3306,
    Database:      "mydb",
    Username:      "root",
    Password:      "password",
    MaxIdleConns:  10,
    MaxOpenConns:  100,
    SingularTable: true,  // 表名不加 s
})

// 使用
var user User
gormDB.Where("id = ?", 1).First(&user)

// 关闭
defer db.CloseDB(gormDB)
```

#### 2. go-zero sqlx (推荐用于简单 CRUD)

```go
import "github.com/jinguoxing/idrm-go-base/db"

// 初始化
sqlxConn := db.InitSqlx(db.Config{
    Host:     "localhost",
    Port:     3306,
    Database: "mydb",
    Username: "root",
    Password: "password",
})

// 查询
var user User
err := sqlxConn.QueryRow(&user, "SELECT * FROM users WHERE id=?", userId)

// 插入
result, err := sqlxConn.Exec("INSERT INTO users(name) VALUES(?)", name)

// 事务
err := sqlxConn.Transact(func(s sqlx.Session) error {
    _, err := s.Exec("UPDATE users SET status=? WHERE id=?", 1, userId)
    return err
})
```

### errorx 模块

```go
// 使用预定义错误码
err := errorx.NewWithCode(errorx.ErrCodeNotFound)

// 自定义错误消息
err := errorx.New(errorx.ErrCodeBusiness, "用户不存在")
```

### response 模块

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

### middleware 模块

```go
import "github.com/jinguoxing/idrm-go-base/middleware"

// 在 main.go 中注册全局中间件
server := rest.MustNewServer(c.RestConf)

server.Use(middleware.Recovery())  // Panic 恢复
server.Use(middleware.RequestID()) // 请求 ID
server.Use(middleware.Trace())     // 链路追踪
server.Use(middleware.CORS())      // 跨域
server.Use(middleware.Logger())    // 日志

// 路由级别认证中间件
server.Use(middleware.Auth(c.Auth.AccessSecret))
```

### telemetry 模块

```go
import "github.com/jinguoxing/idrm-go-base/telemetry"

// 初始化
err := telemetry.Init(telemetry.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    Log: telemetry.LogConfig{
        Level: "info",
        Mode:  "file",
        Path:  "logs",
    },
    Trace: telemetry.TraceConfig{
        Enabled:  true,
        Endpoint: "localhost:4317",
    },
})
```

### uuid 模块

```go
import "github.com/jinguoxing/idrm-go-base/uuid"

// 生成 UUID v7
id := uuid.New()  // 返回 string

// 适合作为数据库主键
type User struct {
    ID   string `gorm:"primaryKey;type:varchar(36)"`
    Name string
}

user := User{
    ID:   uuid.New(),
    Name: "张三",
}
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

## 技术选型建议

| 场景 | 推荐 | 理由 |
|------|------|------|
| 复杂查询、关联查询 | GORM | ORM 便利性 |
| 简单 CRUD、高性能要求 | go-zero sqlx | 性能高、轻量 |
| 用户认证、权限管理 | middleware.Auth | 内置 JWT 支持 |
| 日志、链路追踪 | telemetry | OpenTelemetry 集成 |

---

## License

[MIT License](LICENSE)
