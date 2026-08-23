// Package ui holds Charm/lipgloss terminal styling shared by command output.
package ui

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Render controls color and optional output writer for one command.
type Render struct {
	Color bool
	Out   io.Writer
}

// For builds a Render from a writer (stdout/stderr). Color is off when piped
// or when NO_COLOR is set.
func For(w io.Writer) Render {
	return Render{Color: ColorEnabled(w), Out: w}
}

// ColorEnabled reports whether lipgloss should emit ANSI styles.
func ColorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Underline(true)
	passStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	failStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStatStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sourceStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	dockerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	upStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	downStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	nameStyle     = lipgloss.NewStyle().Bold(true)
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).BorderForeground(lipgloss.Color("214"))
)

func paint(enabled bool, style lipgloss.Style, s string) string {
	if !enabled || s == "" {
		return s
	}
	return style.Render(s)
}

// Title styles a section heading.
func Title(r Render, s string) string { return paint(r.Color, titleStyle, s) }

// Success styles a one-line success message.
func Success(r Render, s string) string { return paint(r.Color, successStyle, s) }

// Warn styles a warning line.
func Warn(r Render, s string) string { return paint(r.Color, warnStyle, s) }

// Error styles an error for stderr.
func Error(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return errStyle.Render(s)
}

// Dim styles secondary hint text.
func Dim(r Render, s string) string { return paint(r.Color, dimStyle, s) }

// Box draws a bordered note (e.g. elevation hint).
func Box(r Render, s string) string {
	if !r.Color {
		return s
	}
	return boxStyle.Render(s)
}

func modePaint(r Render, mode string) string {
	switch strings.ToLower(mode) {
	case "source":
		return paint(r.Color, sourceStyle, mode)
	case "docker":
		return paint(r.Color, dockerStyle, mode)
	default:
		return mode
	}
}

func containerPaint(r Render, up bool) string {
	if up {
		return paint(r.Color, upStyle, "up")
	}
	return paint(r.Color, downStyle, "down")
}

// Pass styles PASS status text.
func Pass(r Render, s string) string { return paint(r.Color, passStyle, s) }

// Fail styles FAIL status text.
func Fail(r Render, s string) string { return paint(r.Color, failStyle, s) }

// WarnStatus styles WARN status text.
func WarnStatus(r Render, s string) string { return paint(r.Color, warnStatStyle, s) }

// Mode styles docker/source mode.
func Mode(r Render, mode string) string { return modePaint(r, mode) }

// Container styles up/down container state.
func Container(r Render, up bool) string { return containerPaint(r, up) }

// Name styles a bold identifier.
func Name(r Render, s string) string { return paint(r.Color, nameStyle, s) }
