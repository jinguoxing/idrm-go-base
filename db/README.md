# db 模块

数据库初始化工具，支持多种数据库访问方式。

## 支持的数据库

- ✅ MySQL (当前)
- 🔜 PostgreSQL (计划)
- 🔜 TiDB (计划)

## 支持的访问方式

### 1. GORM

**适用场景**：复杂查询、关联查询、ORM 操作

**特点**：
- 功能强大的 ORM
- 支持自动迁移
- 丰富的 Hook 机制
- 关联查询便利

**示例**：
```go
db, err := InitGorm(Config{
    Host:          "localhost",
    Port:          3306,
    Database:      "mydb",
    Username:      "root",
    Password:      "password",
    SingularTable: true,
})

// 查询
var users []User
db.Where("age > ?", 18).Find(&users)

// 关联查询
db.Preload("Orders").Find(&users)
```

### 2. go-zero sqlx

**适用场景**：简单 CRUD、高性能查询

**特点**：
- 轻量级
- 高性能
- 类型安全
- 支持事务

**示例**：
```go
conn := InitSqlx(Config{
    Host:     "localhost",
    Port:     3306,
    Database: "mydb",
    Username: "root",
    Password: "password",
})

// 查询
var user User
err := conn.QueryRow(&user, "SELECT * FROM users WHERE id=?", userId)

// 事务
err := conn.Transact(func(s sqlx.Session) error {
    _, err := s.Exec("UPDATE users SET status=? WHERE id=?", 1, userId)
    return err
})
```

## 配置说明

```go
type Config struct {
    // 连接配置
    Host     string
    Port     int
    Database string
    Username string
    Password string
    Charset  string // 默认 utf8mb4
    
    // 连接池配置
    MaxIdleConns    int // 最大空闲连接数
    MaxOpenConns    int // 最大连接数
    ConnMaxLifetime int // 连接最大生存时间 (秒)
    ConnMaxIdleTime int // 连接最大空闲时间 (秒)
    
    // 日志配置
    LogLevel      string // silent, error, warn, info
    SlowThreshold int    // 慢查询阈值 (毫秒)
    
    // GORM 专用配置
    SkipDefaultTxn    bool // 跳过默认事务
    PrepareStmt       bool // 预编译语句
    SingularTable     bool // 单数表名 (不加 s)
    DisableForeignKey bool // 禁用外键约束
}
```

## 性能对比

| 操作 | GORM | go-zero sqlx | 性能差异 |
|------|------|--------------|----------|
| 简单查询 | 约 10ms | 约 5ms | sqlx 快 2x |
| 批量插入 | 约 100ms | 约 50ms | sqlx 快 2x |
| 复杂关联查询 | 约 50ms | - | GORM 便利 |

## 使用建议

1. **优先使用 GORM**：除非性能是瓶颈
2. **高频读接口**：考虑使用 sqlx
3. **写操作**：两者性能相近，选择 GORM 更便利
4. **复杂业务逻辑**：使用 GORM 的 Hook 和事件

## 测试

```bash
# 运行单元测试
cd db && go test -v

# 跳过需要 MySQL 的集成测试
go test -v -short
```
