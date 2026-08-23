package progress

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunNonTTYWritesFallback(t *testing.T) {
	t.Parallel()

	src := scriptedSource{id: "alpine", events: []Update{
		{ID: "alpine", Label: "alpine", Current: 1, Total: 2},
		{ID: "alpine", Label: "alpine", Done: true},
	}}
	var buf bytes.Buffer
	tty := false
	if err := Run(context.Background(), []Source{src}, Options{Output: &buf, IsTTY: &tty}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "50%") || !strings.Contains(out, "done") {
		t.Fatalf("output = %q", out)
	}
}

func TestViewTTYIncludesLabels(t *testing.T) {
	t.Parallel()

	var m Model
	m.Apply(Update{ID: "alpine", Label: "alpine:latest", Current: 1, Total: 2})
	got := ViewTTY(m, 80)
	if !strings.Contains(got, "alpine") {
		t.Fatalf("view = %q", got)
	}
}
