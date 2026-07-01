package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/jinguoxing/idrm-go-base/telemetry/internal/otelresource"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/zeromicro/go-zero/core/logx"
)

const instrumentationName = "github.com/jinguoxing/idrm-go-base/telemetry/metrics"

var (
	mu                     sync.RWMutex
	meterProvider          *sdkmetric.MeterProvider
	meter                  metric.Meter
	instrumentationVersion = "dev"
)

// Init 初始化 Metrics 系统。
func Init(config MetricsConfig, serviceName, version, environment string) error {
	mu.Lock()
	defer mu.Unlock()

	if version != "" {
		instrumentationVersion = version
	}

	if meterProvider != nil {
		_ = meterProvider.Shutdown(context.Background())
		meterProvider = nil
	}

	if !config.Enabled {
		provider := noop.NewMeterProvider()
		otel.SetMeterProvider(provider)
		meter = provider.Meter(instrumentationName, metric.WithInstrumentationVersion(instrumentationVersion))
		logx.Info("Metrics 未启用")
		return nil
	}

	ctx := context.Background()
	exporter, err := newExporter(ctx, config)
	if err != nil {
		return err
	}

	res, err := otelresource.New(serviceName, version, environment)
	if err != nil {
		logx.Errorf("创建 Metrics resource 失败: %v", err)
		return err
	}

	interval := exportInterval(config.Interval)
	timeout := exportTimeout(config.Timeout)

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(interval),
			sdkmetric.WithTimeout(timeout),
		)),
	)

	otel.SetMeterProvider(provider)
	meterProvider = provider
	meter = provider.Meter(instrumentationName, metric.WithInstrumentationVersion(instrumentationVersion))

	logx.Infof("Metrics 初始化完成 [protocol=%s, endpoint=%s, interval=%s]",
		normalizeProtocol(config.Protocol), config.Endpoint, interval)
	return nil
}

// Close 优雅关闭 Metrics 系统。
func Close(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()

	if meterProvider == nil {
		return nil
	}

	logx.Info("关闭 Metrics...")
	err := meterProvider.Shutdown(ctx)
	meterProvider = nil

	provider := noop.NewMeterProvider()
	otel.SetMeterProvider(provider)
	meter = provider.Meter(instrumentationName, metric.WithInstrumentationVersion(instrumentationVersion))
	return err
}

// Meter 获取全局 Meter。
func Meter() metric.Meter {
	mu.RLock()
	m := meter
	version := instrumentationVersion
	mu.RUnlock()

	if m != nil {
		return m
	}
	return otel.Meter(instrumentationName, metric.WithInstrumentationVersion(version))
}

// MeterProvider 获取全局 MeterProvider。
func MeterProvider() metric.MeterProvider {
	return otel.GetMeterProvider()
}

func exportInterval(intervalSeconds int) time.Duration {
	interval := time.Duration(intervalSeconds) * time.Second
	if interval <= 0 {
		return 15 * time.Second
	}
	return interval
}
