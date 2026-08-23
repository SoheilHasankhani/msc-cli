package progress

import (
	"os"
	"testing"

	"golang.org/x/term"
)

func TestResolveOutputUsesStderrWhenInteractive(t *testing.T) {
	t.Parallel()
	if !term.IsTerminal(int(os.Stdout.Fd())) && !term.IsTerminal(int(os.Stderr.Fd())) {
		t.Skip("non-interactive test runner")
	}
	_, tty := ResolveOutput(nil)
	if !tty {
		t.Fatal("expected TTY progress in an interactive terminal")
	}
}

func TestResolveOutputBufferIsNotTTY(t *testing.T) {
	t.Parallel()
	_, tty := ResolveOutput(new(bytesSink))
	if tty {
		t.Fatal("buffer should not be a TTY")
	}
}

type bytesSink struct{}

func (bytesSink) Write(p []byte) (int, error) { return len(p), nil }
