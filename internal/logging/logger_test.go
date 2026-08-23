package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewWritesJSONLines(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	lg, closer, err := New(Options{Writer: &buf, Level: slog.LevelInfo})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = closer.Close() })

	lg.Info("hello", "project", "isos", "cmd", "status")
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	line := strings.TrimSpace(buf.String())
	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("not JSON-lines: %q %v", line, err)
	}
	if rec["msg"] != "hello" || rec["project"] != "isos" {
		t.Fatalf("%v", rec)
	}
	if rec["level"] != "INFO" {
		t.Fatalf("level %v", rec["level"])
	}
}

func TestParseLevel(t *testing.T) {
	t.Parallel()
	if ParseLevel("debug") != slog.LevelDebug || ParseLevel("") != slog.LevelInfo {
		t.Fatal(ParseLevel("debug"), ParseLevel(""))
	}
	if ParseLevel("warn") != slog.LevelWarn || ParseLevel("error") != slog.LevelError {
		t.Fatal(ParseLevel("warn"), ParseLevel("error"))
	}
}
