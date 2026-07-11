# Telemetry 日志系统（Phase 1）

## 📋 概述

基于 go-zero logx 的日志系统，支持本地日志和远程日志上报。

## ✨ 功能特性

- ✅ **本地日志**：基于 go-zero logx，支持文件/控制台输出
- ✅ **按大小切分**：支持按天或按大小切分，并设置备份保留数
- ✅ **字符串内容截取**：可限制单条 string 主内容的字节长度
- ✅ **远程日志**：自定义 Writer，批量异步上报
- ✅ **日志级别**：debug/info/error/severe
- ✅ **自动刷新**：每3秒或达到批量大小自动发送
- ✅ **故障容错**：远程发送失败不影响本地日志
- ✅ **优雅关闭**：确保所有日志发送完成

## ⚙️ 配置

### 配置结构

```go
type LogConfig struct {
    Level    string // 日志级别
    Mode     string // 输出模式：console/file
    Path     string // 日志文件路径
    KeepDays int    // 保留天数

    // 文件切分与内容截取
    Rotation         string // daily（默认）或 size
    MaxSize          int    // 单个日志文件上限（MB），仅 Rotation=size 生效
    MaxBackups       int    // 每类日志保留的备份数，0=不限
    MaxContentLength uint32 // string 主内容上限（字节），0=不截取

    // 远程日志
    RemoteEnabled bool   // 是否启用远程上报
    RemoteUrl     string // 远程接收地址
    RemoteBatch   int    // 批量大小
    RemoteTimeout int    // 超时时间(秒)
}
```

### 配置示例

```yaml
# api/etc/api.yaml
Telemetry:
  ServiceName: idrm-api
  ServiceVersion: 1.0.0
  
  Log:
    Level: info
    Mode: file
    Path: logs
    KeepDays: 7
    Rotation: size
    MaxSize: 100
    MaxBackups: 30
    MaxContentLength: 65536
    RemoteEnabled: true
    RemoteUrl: http://log-collector:8080/api/logs
    RemoteBatch: 100
    RemoteTimeout: 5
```

`Rotation=size` 时必须设置大于 0 的 `MaxSize`。`MaxBackups` 按 access/error/severe/slow/stat 等日志类别分别计算，`KeepDays` 只限制保留时间，不是总容量上限。

`MaxContentLength` 仅截取 Go 类型为 `string` 的日志主内容，不截取结构化 fields、error 或对象，也不限制最终 JSON 日志行的总长度。超限时 go-zero 会写入 `"truncated":true`。它对 console/file/volume 模式都生效，按字节截取可能在 UTF-8 多字节字符中间切断。

## 🚀 使用方法

### 1. 初始化

```go
import (
    "github.com/jinguoxing/idrm-go-base/telemetry/log"
)

func main() {
    // 初始化日志系统
    if err := log.SetUp(config.Telemetry.Log, config.Telemetry.ServiceName); err != nil {
        panic(err)
    }
    defer log.Close()
    
    // 业务代码...
}
```

### 2. 记录日志（使用 go-zero logx）

```go
import "github.com/zeromicro/go-zero/core/logx"

// 基础日志
logx.Info("用户登录")
logx.Infof("用户 %s 登录成功", username)

// 结构化日志
logx.Infow("用户操作",
    logx.Field("action", "login"),
    logx.Field("user_id", 123),
    logx.Field("ip", "127.0.0.1"))

// 错误日志
logx.Error("操作失败")
logx.Errorf("处理失败: %v", err)

// 带 Context 的日志（自动提取 trace 信息）
logx.WithContext(ctx).Info("处理请求")
logx.WithContext(ctx).Errorf("处理失败: %v", err)
```

### 3. 日志级别

```go
logx.Debug("调试信息")   // debug
logx.Info("普通信息")    // info
logx.Slow("慢日志")      // info (go-zero特色)
logx.Stat("统计信息")    // info (go-zero特色)
logx.Error("错误信息")   // error
```

## 📊 远程日志格式

发送到远程服务器的日志格式：

```json
{
  "logs": [
    {
      "timestamp": 1703307600,
      "level": "info",
      "message": "用户登录成功",
      "service_name": "idrm-api",
      "trace_id": "abc123",
      "span_id": "def456",
      "fields": {
        "user_id": 123,
        "action": "login"
      }
    }
  ]
}
```

## 🔧 工作原理

### 本地日志流程

```
业务代码
  ↓
logx.Info()
  ↓
go-zero logx
  ↓
文件/控制台
```

### 远程日志流程

```
业务代码
  ↓
logx.Info()
  ↓
RemoteWriter.Write()
  ↓
缓冲区 (Buffer)
  ↓
批量发送 (每3秒或100条)
  ↓
HTTP POST
  ↓
远程服务器
```

## ⚡ 性能优化

1. **批量发送**：减少网络请求次数
2. **异步处理**：不阻塞业务逻辑
3. **自动刷新**：定时发送，避免积压
4. **故障容错**：发送失败只记录本地

## 📝 完整示例

```go
// main.go
package main

import (
    "context"
    "idrm/api/internal/config"
    "github.com/jinguoxing/idrm-go-base/telemetry/log"
    
    "github.com/zeromicro/go-zero/core/conf"
    "github.com/zeromicro/go-zero/core/logx"
)

func main() {
    // 1. 加载配置
    var c config.Config
    conf.MustLoad("etc/api.yaml", &c)
    
    // 2. 初始化日志
    if err := log.SetUp(c.Telemetry.Log, c.Telemetry.ServiceName); err != nil {
        panic(err)
    }
    defer log.Close()
    
    // 3. 使用日志
    logx.Info("服务启动")
    
    // 4. 业务逻辑
    processRequest(context.Background())
    
    logx.Info("服务停止")
}

func processRequest(ctx context.Context) {
    // 带 context 的日志
    logx.WithContext(ctx).Infow("处理请求",
        logx.Field("request_id", "req123"),
        logx.Field("user_id", 456))
    
    // 错误处理
    if err := doSomething(); err != nil {
        logx.WithContext(ctx).Errorf("处理失败: %v", err)
        return
    }
    
    logx.WithContext(ctx).Info("处理成功")
}
```

## 🎯 下一步

Phase 1 完成后，继续：

- **Phase 2**：实现 OpenTelemetry 链路追踪
- **Phase 3**：实现审计日志功能

## ❓ 常见问题

**Q: 远程日志发送失败会影响业务吗？**

A: 不会。远程发送是异步的，失败只会记录到本地日志。

**Q: 如何测试远程日志？**

A: 可以启动一个简单的 HTTP 服务接收日志，或使用 `nc -l 8080` 监听。

**Q: 日志太多会影响性能吗？**

A: 使用批量+异步方式，对性能影响很小。建议生产环境使用 `info` 级别。

**Q: 如何关闭远程日志？**

A: 设置 `RemoteEnabled: false` 即可。
