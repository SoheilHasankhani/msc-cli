package ui

import (
	"fmt"
	"strings"
)

// SuccessLine returns a styled success with optional dim hint on the next line.
func SuccessLine(r Render, msg, hint string) string {
	var b strings.Builder
	fmt.Fprintln(&b, Success(r, msg))
	if hint != "" {
		fmt.Fprintln(&b, Dim(r, hint))
	}
	return b.String()
}

// Infof writes a dim info line when Out is set.
func Infof(r Render, format string, args ...any) {
	if r.Out == nil {
		return
	}
	_, _ = fmt.Fprintln(r.Out, Dim(r, fmt.Sprintf(format, args...)))
}
