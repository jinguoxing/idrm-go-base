package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	auditLogger *AuditLogger
)

// AuditLogger 审计日志记录器
type AuditLogger struct {
	serviceName string
	url         string
	client      *http.Client
	buffer      []AuditLog
	bufferSize  int
	mu          sync.Mutex
	closeChan   chan struct{}
}

// Init 初始化审计日志
func Init(config AuditConfig, serviceName string) {
	if !config.Enabled {
		logx.Info("审计日志未启用")
		return
	}

	auditLogger = &AuditLogger{
		serviceName: serviceName,
		url:         config.Url,
		buffer:      make([]AuditLog, 0, config.Buffer),
		bufferSize:  config.Buffer,
		closeChan:   make(chan struct{}),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// 启动定时刷新
	go auditLogger.flushLoop()

	logx.Infof("审计日志初始化完成 [url=%s, buffer=%d]", config.Url, config.Buffer)
}

// Log 记录审计日志
func Log(ctx context.Context, log AuditLog) {
	if auditLogger == nil {
		return
	}

	// 补充基础信息
	log.Timestamp = time.Now()
	log.ServiceName = auditLogger.serviceName

	// 提取 TraceID
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		log.TraceID = span.SpanContext().TraceID().String()
	}

	auditLogger.add(log)
}

// LogWithDuration 记录审计日志（带执行时长）
func LogWithDuration(ctx context.Context, log AuditLog, start time.Time) {
	log.Duration = time.Since(start).Milliseconds()
	Log(ctx, log)
}

// add 添加审计日志到缓冲区
func (a *AuditLogger) add(log AuditLog) {
	a.mu.Lock()
	a.buffer = append(a.buffer, log)
	shouldFlush := len(a.buffer) >= a.bufferSize
	a.mu.Unlock()

	if shouldFlush {
		a.flush()
	}
}

// flush 发送审计日志
func (a *AuditLogger) flush() {
	a.mu.Lock()
	if len(a.buffer) == 0 {
		a.mu.Unlock()
		return
	}

	logs := make([]AuditLog, len(a.buffer))
	copy(logs, a.buffer)
	a.buffer = a.buffer[:0]
	a.mu.Unlock()

	// 异步发送
	go a.send(logs)
}

// send 发送到审计服务
func (a *AuditLogger) send(logs []AuditLog) {
	data, err := json.Marshal(map[string]interface{}{
		"audit_logs": logs,
	})
	if err != nil {
		logx.Errorf("marshal audit logs failed: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", a.url, bytes.NewReader(data))
	if err != nil {
		logx.Errorf("create audit request failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		logx.Errorf("send audit logs failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logx.Errorf("audit server returned status: %d", resp.StatusCode)
	}
}

// flushLoop 定时刷新
func (a *AuditLogger) flushLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.flush()
		case <-a.closeChan:
			a.flush()
			return
		}
	}
}

// Close 关闭审计日志
func Close() {
	if auditLogger != nil {
		close(auditLogger.closeChan)
		time.Sleep(100 * time.Millisecond) // 等待最后一次flush
	}
}

// IsEnabled 是否启用审计日志
func IsEnabled() bool {
	return auditLogger != nil
}
