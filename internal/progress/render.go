package progress

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

var (
	labelStyle  = lipgloss.NewStyle().Width(32)
	doneStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

func renderLabel(label string) string {
	return labelStyle.Render(Truncate(label, 32))
}

func renderBarLine(bar Bar, progressLine string) string {
	label := renderLabel(bar.Label)
	switch {
	case bar.Err != nil:
		return fmt.Sprintf("%s %s", label, errStyle.Render(Truncate(shortProgressError(bar.Err), 72)))
	case bar.Status != "" && bar.Warn != nil:
		status := renderStatusText(bar.Status, bar.Done)
		warn := warnStyle.Render("pull failed: " + Truncate(shortProgressError(bar.Warn), 56))
		return fmt.Sprintf("%s %s · %s", label, status, warn)
	case bar.Warn != nil:
		return fmt.Sprintf("%s %s", label, warnStyle.Render("pull failed: "+Truncate(shortProgressError(bar.Warn), 72)))
	case bar.Status != "" && (bar.Done || terminalStatus(bar.Status)):
		return fmt.Sprintf("%s %s", label, doneStyle.Render(bar.Status))
	case bar.Status != "":
		return fmt.Sprintf("%s %s", label, statusStyle.Render(bar.Status))
	case bar.Done:
		return fmt.Sprintf("%s %s", label, doneStyle.Render("done"))
	default:
		if progressLine != "" {
			return fmt.Sprintf("%s %s", label, progressLine)
		}
		return label
	}
}

func renderStatusText(status string, done bool) string {
	if done || terminalStatus(status) {
		return doneStyle.Render(status)
	}
	return statusStyle.Render(status)
}

// ViewTTY renders the current Model as stacked Charm progress bars.
func ViewTTY(m Model, width int) string {
	if width < 40 {
		width = 80
	}
	barWidth := width - 36
	if barWidth < 10 {
		barWidth = 10
	}
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(barWidth),
	)
	var b strings.Builder
	for _, id := range m.Order {
		bar := m.Bars[id]
		progressLine := ""
		if bar.Err == nil && bar.Warn == nil && bar.Status == "" && !bar.Done {
			progressLine = p.ViewAs(bar.Percent)
		}
		fmt.Fprintln(&b, renderBarLine(bar, progressLine))
	}
	return b.String()
}

func terminalStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "started", "running", "healthy", "exited", "recreated", "stopped", "removed",
		"pulled", "skipped", "interrupted", "error":
		return true
	default:
		return false
	}
}
