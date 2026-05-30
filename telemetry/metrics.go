package telemetry

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Labels are bounded key-value pairs for metric classification.
type Labels map[string]string

// String returns a deterministic serialized form for use as a map key.
func (l Labels) String() string {
	if len(l) == 0 {
		return ""
	}
	keys := make([]string, 0, len(l))
	for k := range l {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + l[k]
	}
	return strings.Join(parts, ",")
}

var allowedLabelKeys = map[string]bool{
	"provider":       true,
	"skill":          true,
	"stop_reason":    true,
	"execution_mode": true,
	"status":         true,
	"span_kind":      true,
	"component":      true,
	"operation":      true,
}

// ValidateLabelKey returns an error if the key is not allowed for telemetry labels.
func ValidateLabelKey(key string) error {
	if key == "" {
		return nil
	}
	if !allowedLabelKeys[key] {
		return fmt.Errorf("telemetry: label key %q is not allowed (high cardinality or forbidden)", key)
	}
	return nil
}

// filterLabels returns a copy with only allowed keys retained.
func filterLabels(labels Labels) Labels {
	if labels == nil {
		return Labels{}
	}
	filtered := make(Labels, len(labels))
	for k, v := range labels {
		if allowedLabelKeys[k] {
			filtered[k] = v
		}
	}
	return filtered
}

// Meter emits metrics through an in-memory backend.
type Meter interface {
	Counter(name string, delta int64, labels Labels)
	Gauge(name string, value float64, labels Labels)
	Histogram(name string, value float64, labels Labels)
	Snapshot() *MetricsSnapshot
}

// CounterEntry holds a counter value keyed by label string.
type CounterEntry struct {
	Labels Labels `json:"Labels"`
	Value  int64  `json:"Value"`
}

// GaugeEntry holds a gauge value keyed by label string.
type GaugeEntry struct {
	Labels Labels  `json:"Labels"`
	Value  float64 `json:"Value"`
}

// HistogramEntry holds histogram values keyed by label string.
type HistogramEntry struct {
	Labels Labels    `json:"Labels"`
	Values []float64 `json:"Values"`
}

// MetricsSnapshot captures a point-in-time view of all metrics.
type MetricsSnapshot struct {
	Counters   map[string][]CounterEntry   `json:"Counters"`
	Gauges     map[string][]GaugeEntry     `json:"Gauges"`
	Histograms map[string][]HistogramEntry `json:"Histograms"`
}

// MemoryMeter is an in-memory implementation of Meter.
type MemoryMeter struct {
	mu         sync.RWMutex
	counters   map[string]map[string]int64     // name -> labelKey -> value
	gauges     map[string]map[string]float64   // name -> labelKey -> value
	histograms map[string]map[string][]float64  // name -> labelKey -> values
	labelKeys  map[string]map[string]Labels     // name -> labelKey -> original Labels
}

// NewMemoryMeter creates a new in-memory meter.
func NewMemoryMeter() *MemoryMeter {
	return &MemoryMeter{
		counters:   make(map[string]map[string]int64),
		gauges:     make(map[string]map[string]float64),
		histograms: make(map[string]map[string][]float64),
		labelKeys:  make(map[string]map[string]Labels),
	}
}

func (m *MemoryMeter) ensureLabelKey(name, lk string, labels Labels) {
	if m.labelKeys[name] == nil {
		m.labelKeys[name] = make(map[string]Labels)
	}
	if _, exists := m.labelKeys[name][lk]; !exists {
		m.labelKeys[name][lk] = labels
	}
}

// Counter increments a named counter by delta.
func (m *MemoryMeter) Counter(name string, delta int64, labels Labels) {
	labels = filterLabels(labels)
	m.mu.Lock()
	defer m.mu.Unlock()
	lk := labels.String()
	if m.counters[name] == nil {
		m.counters[name] = make(map[string]int64)
	}
	m.ensureLabelKey(name, lk, labels)
	m.counters[name][lk] += delta
}

// Gauge sets a named gauge to value.
func (m *MemoryMeter) Gauge(name string, value float64, labels Labels) {
	labels = filterLabels(labels)
	m.mu.Lock()
	defer m.mu.Unlock()
	lk := labels.String()
	if m.gauges[name] == nil {
		m.gauges[name] = make(map[string]float64)
	}
	m.ensureLabelKey(name, lk, labels)
	m.gauges[name][lk] = value
}

// Histogram records a value in a named histogram.
func (m *MemoryMeter) Histogram(name string, value float64, labels Labels) {
	labels = filterLabels(labels)
	m.mu.Lock()
	defer m.mu.Unlock()
	lk := labels.String()
	if m.histograms[name] == nil {
		m.histograms[name] = make(map[string][]float64)
	}
	m.ensureLabelKey(name, lk, labels)
	m.histograms[name][lk] = append(m.histograms[name][lk], value)
}

// Snapshot returns a point-in-time copy of all metrics.
func (m *MemoryMeter) Snapshot() *MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap := &MetricsSnapshot{
		Counters:   make(map[string][]CounterEntry),
		Gauges:     make(map[string][]GaugeEntry),
		Histograms: make(map[string][]HistogramEntry),
	}

	for name, labelMap := range m.counters {
		entries := make([]CounterEntry, 0, len(labelMap))
		for lk, v := range labelMap {
			entries = append(entries, CounterEntry{Labels: m.labelKeys[name][lk], Value: v})
		}
		snap.Counters[name] = entries
	}
	for name, labelMap := range m.gauges {
		entries := make([]GaugeEntry, 0, len(labelMap))
		for lk, v := range labelMap {
			entries = append(entries, GaugeEntry{Labels: m.labelKeys[name][lk], Value: v})
		}
		snap.Gauges[name] = entries
	}
	for name, labelMap := range m.histograms {
		entries := make([]HistogramEntry, 0, len(labelMap))
		for lk, values := range labelMap {
			copied := make([]float64, len(values))
			copy(copied, values)
			entries = append(entries, HistogramEntry{Labels: m.labelKeys[name][lk], Values: copied})
		}
		snap.Histograms[name] = entries
	}
	return snap
}
