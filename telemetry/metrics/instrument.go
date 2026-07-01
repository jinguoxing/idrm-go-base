package metrics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// InstrumentKind describes the OTel synchronous instrument type managed by
// the metrics helper registry.
type InstrumentKind string

const (
	InstrumentKindCounter   InstrumentKind = "counter"
	InstrumentKindHistogram InstrumentKind = "histogram"
)

var (
	// ErrInvalidInstrumentDef indicates an invalid metric definition.
	ErrInvalidInstrumentDef = errors.New("invalid metrics instrument definition")
	// ErrInstrumentAlreadyRegistered indicates a duplicate metric definition.
	ErrInstrumentAlreadyRegistered = errors.New("metrics instrument already registered")
	// ErrInstrumentNotFound indicates the metric name was not registered.
	ErrInstrumentNotFound = errors.New("metrics instrument not found")
	// ErrInstrumentKindMismatch indicates the metric name exists with another kind.
	ErrInstrumentKindMismatch = errors.New("metrics instrument kind mismatch")
	// ErrInvalidMeasurement indicates the metric measurement is not valid for the instrument.
	ErrInvalidMeasurement = errors.New("invalid metrics measurement")
)

// InstrumentDef defines one business metric to register at service startup.
type InstrumentDef struct {
	Name        string
	Kind        InstrumentKind
	Description string
	Unit        string
}

var (
	instrumentMu sync.RWMutex
	counters     = make(map[string]metric.Int64Counter)
	histograms   = make(map[string]metric.Float64Histogram)
)

// InitInstruments registers all business metric instruments.
//
// Call this once during service startup after telemetry.Init and before serving
// requests. Runtime recording helpers only write to instruments registered here
// to avoid dynamic metric-name growth on hot paths.
func InitInstruments(defs ...InstrumentDef) error {
	if len(defs) == 0 {
		return nil
	}

	normalized, err := validateInstrumentDefs(defs)
	if err != nil {
		return err
	}

	m := Meter()

	instrumentMu.Lock()
	defer instrumentMu.Unlock()

	for _, def := range normalized {
		if _, ok := counters[def.Name]; ok {
			return fmt.Errorf("%w: %s", ErrInstrumentAlreadyRegistered, def.Name)
		}
		if _, ok := histograms[def.Name]; ok {
			return fmt.Errorf("%w: %s", ErrInstrumentAlreadyRegistered, def.Name)
		}
	}

	newCounters := make(map[string]metric.Int64Counter)
	newHistograms := make(map[string]metric.Float64Histogram)
	for _, def := range normalized {
		switch def.Kind {
		case InstrumentKindCounter:
			counter, err := m.Int64Counter(def.Name, int64CounterOptions(def)...)
			if err != nil {
				return fmt.Errorf("create counter %q: %w", def.Name, err)
			}
			newCounters[def.Name] = counter
		case InstrumentKindHistogram:
			histogram, err := m.Float64Histogram(def.Name, float64HistogramOptions(def)...)
			if err != nil {
				return fmt.Errorf("create histogram %q: %w", def.Name, err)
			}
			newHistograms[def.Name] = histogram
		}
	}

	for name, counter := range newCounters {
		counters[name] = counter
	}
	for name, histogram := range newHistograms {
		histograms[name] = histogram
	}

	return nil
}

// MustInitInstruments registers business metric instruments and panics on error.
func MustInitInstruments(defs ...InstrumentDef) {
	if err := InitInstruments(defs...); err != nil {
		panic(err)
	}
}

// IncCounter increments a registered Int64Counter.
func IncCounter(ctx context.Context, name string, value int64, attrs ...attribute.KeyValue) error {
	if value < 0 {
		return fmt.Errorf("%w: counter %q cannot record negative value %d", ErrInvalidMeasurement, name, value)
	}

	counter, err := lookupCounter(name)
	if err != nil {
		return err
	}

	counter.Add(ctx, value, metric.WithAttributes(attrs...))
	return nil
}

// RecordHistogram records a value into a registered Float64Histogram.
func RecordHistogram(ctx context.Context, name string, value float64, attrs ...attribute.KeyValue) error {
	histogram, err := lookupHistogram(name)
	if err != nil {
		return err
	}

	histogram.Record(ctx, value, metric.WithAttributes(attrs...))
	return nil
}

// RecordDuration records a duration in seconds into a registered Float64Histogram.
func RecordDuration(ctx context.Context, name string, duration time.Duration, attrs ...attribute.KeyValue) error {
	return RecordHistogram(ctx, name, duration.Seconds(), attrs...)
}

func validateInstrumentDefs(defs []InstrumentDef) ([]InstrumentDef, error) {
	normalized := make([]InstrumentDef, 0, len(defs))
	seen := make(map[string]struct{}, len(defs))

	for _, def := range defs {
		def.Name = strings.TrimSpace(def.Name)
		if def.Name == "" {
			return nil, fmt.Errorf("%w: empty name", ErrInvalidInstrumentDef)
		}
		if _, ok := seen[def.Name]; ok {
			return nil, fmt.Errorf("%w: %s", ErrInstrumentAlreadyRegistered, def.Name)
		}
		seen[def.Name] = struct{}{}

		switch def.Kind {
		case InstrumentKindCounter, InstrumentKindHistogram:
		default:
			return nil, fmt.Errorf("%w: unsupported kind %q for %s", ErrInvalidInstrumentDef, def.Kind, def.Name)
		}

		normalized = append(normalized, def)
	}

	return normalized, nil
}

func int64CounterOptions(def InstrumentDef) []metric.Int64CounterOption {
	opts := make([]metric.Int64CounterOption, 0, 2)
	if def.Description != "" {
		opts = append(opts, metric.WithDescription(def.Description))
	}
	if def.Unit != "" {
		opts = append(opts, metric.WithUnit(def.Unit))
	}
	return opts
}

func float64HistogramOptions(def InstrumentDef) []metric.Float64HistogramOption {
	opts := make([]metric.Float64HistogramOption, 0, 2)
	if def.Description != "" {
		opts = append(opts, metric.WithDescription(def.Description))
	}
	if def.Unit != "" {
		opts = append(opts, metric.WithUnit(def.Unit))
	}
	return opts
}

func lookupCounter(name string) (metric.Int64Counter, error) {
	name = strings.TrimSpace(name)

	instrumentMu.RLock()
	counter, ok := counters[name]
	_, histogramExists := histograms[name]
	instrumentMu.RUnlock()

	if ok {
		return counter, nil
	}
	if histogramExists {
		return nil, fmt.Errorf("%w: %s is histogram", ErrInstrumentKindMismatch, name)
	}
	return nil, fmt.Errorf("%w: %s", ErrInstrumentNotFound, name)
}

func lookupHistogram(name string) (metric.Float64Histogram, error) {
	name = strings.TrimSpace(name)

	instrumentMu.RLock()
	histogram, ok := histograms[name]
	_, counterExists := counters[name]
	instrumentMu.RUnlock()

	if ok {
		return histogram, nil
	}
	if counterExists {
		return nil, fmt.Errorf("%w: %s is counter", ErrInstrumentKindMismatch, name)
	}
	return nil, fmt.Errorf("%w: %s", ErrInstrumentNotFound, name)
}

func resetInstruments() {
	instrumentMu.Lock()
	defer instrumentMu.Unlock()

	counters = make(map[string]metric.Int64Counter)
	histograms = make(map[string]metric.Float64Histogram)
}
