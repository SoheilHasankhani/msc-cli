package progress

import (
	"strings"
	"testing"
)

func TestApplyUpdateCreatesAndUpdatesBars(t *testing.T) {
	t.Parallel()

	var m Model
	m.Apply(Update{ID: "doctor", Label: "doctor", Current: 1, Total: 4})
	if len(m.Order) != 1 || m.Bars["doctor"].Label != "doctor" {
		t.Fatalf("create: %#v", m)
	}
	if m.Bars["doctor"].Percent != 0.25 {
		t.Fatalf("percent = %v, want 0.25", m.Bars["doctor"].Percent)
	}

	m.Apply(Update{ID: "doctor", Current: 4, Total: 4})
	if len(m.Order) != 1 {
		t.Fatal("repeat id must not add a new bar")
	}
	if m.Bars["doctor"].Percent != 1 {
		t.Fatalf("percent = %v, want 1", m.Bars["doctor"].Percent)
	}

	m.Apply(Update{ID: "wallet", Label: "wallet", Current: 0, Total: 0})
	if len(m.Order) != 2 || m.Order[1] != "wallet" {
		t.Fatalf("order = %v", m.Order)
	}
}

func TestApplyUpdateDoneAndErr(t *testing.T) {
	t.Parallel()

	var m Model
	m.Apply(Update{ID: "a", Done: true, Err: errSentinel{}})
	bar := m.Bars["a"]
	if !bar.Done || bar.Err == nil {
		t.Fatalf("bar = %#v", bar)
	}
}

func TestApplyUpdateWarnPreservedAcrossStatus(t *testing.T) {
	t.Parallel()

	var m Model
	m.Apply(Update{ID: "rabbitmq", Label: "rabbitmq", Done: true, Warn: errSentinel{}})
	m.Apply(Update{ID: "rabbitmq", Status: "Running", Done: true})
	bar := m.Bars["rabbitmq"]
	if bar.Warn == nil || bar.Status != "Running" {
		t.Fatalf("bar = %#v", bar)
	}
}

func TestRenderBarLineStatusWithWarn(t *testing.T) {
	t.Parallel()

	line := renderBarLine(Bar{
		Label:  "rabbitmq",
		Status: "Running",
		Done:   true,
		Warn:   errSentinel{},
	}, "")
	if !strings.Contains(line, "Running") || !strings.Contains(line, "pull failed") {
		t.Fatalf("line = %q", line)
	}
}

type errSentinel struct{}

func (errSentinel) Error() string { return "boom" }
