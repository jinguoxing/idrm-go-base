package log

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// RemoteWriter 远程日志写入器
type RemoteWriter struct {
	serviceName string
	url         string
	client      *http.Client
	batchSize   int
	buffer      []LogEntry
	mu          sync.Mutex
	closeChan   chan struct{}
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp   int64                  `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	ServiceName string                 `json:"service_name"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
}

// NewRemoteWriter 创建远程日志写入器
func NewRemoteWriter(serviceName, url string, batchSize int, timeout time.Duration) *RemoteWriter {
	rw := &RemoteWriter{
		serviceName: serviceName,
		url:         url,
		batchSize:   batchSize,
		buffer:      make([]LogEntry, 0, batchSize),
		closeChan:   make(chan struct{}),
		client: &http.Client{
			Timeout: timeout,
		},
	}

	// 启动定时发送
	go rw.flushLoop()

	return rw
}

// Write 实现 io.Writer 接口
func (w *RemoteWriter) Write(p []byte) (n int, err error) {
	// 解析日志内容并添加到缓冲区
	entry := w.parseLogEntry(p)

	w.mu.Lock()
	w.buffer = append(w.buffer, entry)
	shouldFlush := len(w.buffer) >= w.batchSize
	w.mu.Unlock()

	if shouldFlush {
		w.flush()
	}

	return len(p), nil
}

// flush 发送日志到远程服务器
func (w *RemoteWriter) flush() {
	w.mu.Lock()
	if len(w.buffer) == 0 {
		w.mu.Unlock()
		return
	}

	logs := make([]LogEntry, len(w.buffer))
	copy(logs, w.buffer)
	w.buffer = w.buffer[:0]
	w.mu.Unlock()

	// 异步发送到远程
	go w.send(logs)
}

// send 发送日志（异步）
func (w *RemoteWriter) send(logs []LogEntry) {
	data, err := json.Marshal(map[string]interface{}{
		"logs": logs,
	})
	if err != nil {
		logx.Errorf("marshal remote logs failed: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", w.url, bytes.NewReader(data))
	if err != nil {
		logx.Errorf("create remote log request failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		logx.Errorf("send remote logs failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logx.Errorf("remote log server returned status: %d", resp.StatusCode)
	}
}

// flushLoop 定时刷新
func (w *RemoteWriter) flushLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.flush()
		case <-w.closeChan:
			w.flush()
			return
		}
	}
}

// Close 关闭
func (w *RemoteWriter) Close() error {
	close(w.closeChan)
	time.Sleep(100 * time.Millisecond) // 等待最后一次flush
	return nil
}

// parseLogEntry 解析日志条目
func (w *RemoteWriter) parseLogEntry(p []byte) LogEntry {
	// 简单解析，实际可以更复杂
	entry := LogEntry{
		Timestamp:   time.Now().Unix(),
		Message:     string(p),
		ServiceName: w.serviceName,
		Fields:      make(map[string]interface{}),
	}

	// 尝试从消息中提取level
	msg := string(p)
	if len(msg) > 0 {
		// 简单判断（实际应该解析JSON格式的logx输出）
		switch {
		case contains(msg, "ERROR"):
			entry.Level = "error"
		case contains(msg, "WARN"):
			entry.Level = "warn"
		case contains(msg, "INFO"):
			entry.Level = "info"
		case contains(msg, "DEBUG"):
			entry.Level = "debug"
		default:
			entry.Level = "info"
		}
	}

	return entry
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}
