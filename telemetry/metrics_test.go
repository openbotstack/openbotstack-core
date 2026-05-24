package telemetry

import (
	"testing"
)

func TestCounterBasic(t *testing.T) {
	m := NewMemoryMeter()
	m.Counter("execution_total", 1, Labels{"status": "success"})
	m.Counter("execution_total", 1, Labels{"status": "success"})
	m.Counter("execution_total", 1, Labels{"status": "failed"})

	snap := m.Snapshot()
	entries := snap.Counters["execution_total"]
	if entries == nil {
		t.Fatal("expected execution_total counter")
	}
	for _, e := range entries {
		if e.Labels["status"] == "success" && e.Value != 2 {
			t.Fatalf("success count = %d, want 2", e.Value)
		}
		if e.Labels["status"] == "failed" && e.Value != 1 {
			t.Fatalf("failed count = %d, want 1", e.Value)
		}
	}
}

func TestGaugeBasic(t *testing.T) {
	m := NewMemoryMeter()
	m.Gauge("active_executions", 3.0, Labels{})
	m.Gauge("active_executions", 5.0, Labels{})

	snap := m.Snapshot()
	entries := snap.Gauges["active_executions"]
	if entries == nil {
		t.Fatal("expected active_executions gauge")
	}
	if len(entries) != 1 {
		t.Fatalf("gauge entries = %d, want 1", len(entries))
	}
	if entries[0].Value != 5.0 {
		t.Fatalf("gauge value = %f, want 5.0", entries[0].Value)
	}
}

func TestHistogramBasic(t *testing.T) {
	m := NewMemoryMeter()
	m.Histogram("execution_duration_ms", 100.0, Labels{"skill": "summarize"})
	m.Histogram("execution_duration_ms", 200.0, Labels{"skill": "summarize"})
	m.Histogram("execution_duration_ms", 300.0, Labels{"skill": "summarize"})

	snap := m.Snapshot()
	entries := snap.Histograms["execution_duration_ms"]
	if entries == nil {
		t.Fatal("expected execution_duration_ms histogram")
	}
	if len(entries) != 1 {
		t.Fatalf("histogram label groups = %d, want 1", len(entries))
	}
	if len(entries[0].Values) != 3 {
		t.Fatalf("histogram count = %d, want 3", len(entries[0].Values))
	}
}

func TestLabelsStringDeterministic(t *testing.T) {
	l1 := Labels{"b": "2", "a": "1"}
	l2 := Labels{"a": "1", "b": "2"}
	if l1.String() != l2.String() {
		t.Fatalf("Labels.String() not deterministic: %q vs %q", l1.String(), l2.String())
	}
}

func TestLabelsAreBounded(t *testing.T) {
	forbidden := []string{"execution_id", "request_id", "user_id"}
	for _, f := range forbidden {
		err := ValidateLabelKey(f)
		if err == nil {
			t.Fatalf("label key %q should be rejected", f)
		}
	}
}

func TestLabelsAllowValidKeys(t *testing.T) {
	allowed := []string{"provider", "skill", "stop_reason", "execution_mode", "status", "span_kind"}
	for _, a := range allowed {
		err := ValidateLabelKey(a)
		if err != nil {
			t.Fatalf("label key %q should be allowed: %v", a, err)
		}
	}
}

func TestEmptyLabelsAllowed(t *testing.T) {
	err := ValidateLabelKey("")
	if err != nil {
		t.Fatalf("empty label key should be allowed: %v", err)
	}
}

func TestFilterLabelsStripsForbidden(t *testing.T) {
	m := NewMemoryMeter()
	m.Counter("test_counter", 1, Labels{"status": "ok", "forbidden_key": "leaked", "execution_id": "abc"})
	snap := m.Snapshot()
	for _, e := range snap.Counters["test_counter"] {
		if _, ok := e.Labels["forbidden_key"]; ok {
			t.Error("forbidden_key should be stripped by filterLabels")
		}
		if _, ok := e.Labels["execution_id"]; ok {
			t.Error("execution_id should be stripped by filterLabels")
		}
		if e.Labels["status"] != "ok" {
			t.Error("status key should be preserved")
		}
	}
}

func TestNilLabelsStringNoPanic(t *testing.T) {
	var l Labels = nil
	result := l.String()
	if result != "" {
		t.Fatalf("nil Labels.String() = %q, want empty", result)
	}
}

func TestCounterConcurrentSafety(t *testing.T) {
	m := NewMemoryMeter()
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.Counter("test_counter", 1, Labels{})
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	snap := m.Snapshot()
	for _, e := range snap.Counters["test_counter"] {
		if e.Value != 1000 {
			t.Fatalf("concurrent counter = %d, want 1000", e.Value)
		}
	}
}
