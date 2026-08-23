package progress

import (
	"io"
	"os"

	"golang.org/x/term"
)

// ResolveOutput picks the writer for progress rendering and whether to use Charm.
// Some terminals (including IDE panels) only attach a TTY to stdout while progress
// is written to stderr — we still enable Charm in that case.
func ResolveOutput(out io.Writer) (io.Writer, bool) {
	if out == nil {
		out = os.Stderr
	}
	if isFileTTY(out) {
		return out, true
	}
	if term.IsTerminal(int(os.Stderr.Fd())) {
		return os.Stderr, true
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return os.Stderr, true
	}
	return out, false
}

func isFileTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
