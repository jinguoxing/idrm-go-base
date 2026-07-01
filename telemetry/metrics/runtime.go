package metrics

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"runtime/debug"
	"sync"

	"go.opentelemetry.io/otel/metric"
)

var (
	runtimeInstrumentMu sync.Mutex
	runtimeRegistration metric.Registration
)

// InitRuntimeMetrics registers Go runtime observable metrics.
func InitRuntimeMetrics() error {
	return initRuntimeMetrics(Meter())
}

func initRuntimeMetrics(m metric.Meter) error {
	runtimeInstrumentMu.Lock()
	defer runtimeInstrumentMu.Unlock()

	if runtimeRegistration != nil {
		return nil
	}

	goroutines, err := m.Int64ObservableUpDownCounter(
		MetricGoGoroutineCount,
		metric.WithDescription("Count of live goroutines"),
		metric.WithUnit("{goroutine}"),
	)
	if err != nil {
		return fmt.Errorf("create %s: %w", MetricGoGoroutineCount, err)
	}

	memoryAllocated, err := m.Int64ObservableCounter(
		MetricGoMemoryAllocated,
		metric.WithDescription("Memory allocated to the heap by the application"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return fmt.Errorf("create %s: %w", MetricGoMemoryAllocated, err)
	}

	memoryAllocations, err := m.Int64ObservableCounter(
		MetricGoMemoryAllocations,
		metric.WithDescription("Count of allocations to the heap by the application"),
		metric.WithUnit("{allocation}"),
	)
	if err != nil {
		return fmt.Errorf("create %s: %w", MetricGoMemoryAllocations, err)
	}

	memoryGCGoal, err := m.Int64ObservableUpDownCounter(
		MetricGoMemoryGCGoal,
		metric.WithDescription("Heap size target for the end of the GC cycle"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return fmt.Errorf("create %s: %w", MetricGoMemoryGCGoal, err)
	}

	memoryLimit, err := m.Int64ObservableUpDownCounter(
		MetricGoMemoryLimit,
		metric.WithDescription("Go runtime memory limit configured by the user"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return fmt.Errorf("create %s: %w", MetricGoMemoryLimit, err)
	}

	memoryUsed, err := m.Int64ObservableUpDownCounter(
		MetricGoMemoryUsed,
		metric.WithDescription("Memory used by the Go runtime"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return fmt.Errorf("create %s: %w", MetricGoMemoryUsed, err)
	}

	processorLimit, err := m.Int64ObservableUpDownCounter(
		MetricGoProcessorLimit,
		metric.WithDescription("The number of OS threads that can execute user-level Go code simultaneously"),
		metric.WithUnit("{thread}"),
	)
	if err != nil {
		return fmt.Errorf("create %s: %w", MetricGoProcessorLimit, err)
	}

	registration, err := m.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		observer.ObserveInt64(goroutines, int64(runtime.NumGoroutine()))
		observer.ObserveInt64(memoryAllocated, uint64ToInt64(stats.TotalAlloc))
		observer.ObserveInt64(memoryAllocations, uint64ToInt64(stats.Mallocs))
		observer.ObserveInt64(memoryGCGoal, uint64ToInt64(stats.NextGC))
		observer.ObserveInt64(memoryLimit, int64MemoryLimit())
		observer.ObserveInt64(memoryUsed, uint64ToInt64(stats.Sys))
		observer.ObserveInt64(processorLimit, int64(runtime.GOMAXPROCS(0)))
		return nil
	}, goroutines, memoryAllocated, memoryAllocations, memoryGCGoal, memoryLimit, memoryUsed, processorLimit)
	if err != nil {
		return fmt.Errorf("register runtime metrics callback: %w", err)
	}

	runtimeRegistration = registration
	return nil
}

func resetRuntimeMetrics() {
	runtimeInstrumentMu.Lock()
	defer runtimeInstrumentMu.Unlock()

	if runtimeRegistration != nil {
		_ = runtimeRegistration.Unregister()
		runtimeRegistration = nil
	}
}

func uint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(value)
}

func int64MemoryLimit() int64 {
	limit := debug.SetMemoryLimit(-1)
	if limit == math.MaxInt64 {
		return 0
	}
	return limit
}
