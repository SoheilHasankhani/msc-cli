package progress

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatFallbackLine(t *testing.T) {
	t.Parallel()

	if got := FormatFallback(Update{ID: "doctor", Label: "doctor", Current: 42, Total: 100}); !strings.Contains(got, "42%") {
		t.Fatalf("got %q", got)
	}
	if got := FormatFallback(Update{ID: "doctor", Label: "doctor", Done: true}); !strings.Contains(got, "done") {
		t.Fatalf("got %q", got)
	}
	if got := FormatFallback(Update{ID: "doctor", Label: "doctor", Done: true, Err: errSentinel{}}); !strings.Contains(got, "error") {
		t.Fatalf("got %q", got)
	}
}

func TestWriteFallback(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	WriteFallback(&buf, Update{Label: "alpine", Current: 1, Total: 2})
	if !strings.Contains(buf.String(), "alpine") {
		t.Fatalf("buf = %q", buf.String())
	}
}
