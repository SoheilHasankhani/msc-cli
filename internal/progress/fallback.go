package progress

import (
	"fmt"
	"io"
)

// FormatFallback renders one non-TTY progress line.
func FormatFallback(u Update) string {
	label := u.Label
	if label == "" {
		label = u.ID
	}
	switch {
	case u.Err != nil:
		return fmt.Sprintf("%s: error: %s", label, shortProgressError(u.Err))
	case u.Status != "" && u.Warn != nil:
		return fmt.Sprintf("%s: %s · pull failed: %s", label, u.Status, shortProgressError(u.Warn))
	case u.Warn != nil:
		return fmt.Sprintf("%s: warning: %s", label, shortProgressError(u.Warn))
	case u.Status != "":
		return fmt.Sprintf("%s: %s", label, u.Status)
	case u.Done:
		return fmt.Sprintf("%s: done", label)
	case u.Total > 0:
		pct := 100 * u.Current / u.Total
		return fmt.Sprintf("%s: %d%%", label, pct)
	default:
		return fmt.Sprintf("%s: working...", label)
	}
}

// WriteFallback writes a fallback line plus newline.
func WriteFallback(w io.Writer, u Update) {
	if w == nil {
		return
	}
	fmt.Fprintln(w, FormatFallback(u))
}
