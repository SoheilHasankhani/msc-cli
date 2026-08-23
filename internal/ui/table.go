package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table is a column-aligned table with optional lipgloss cell styling.
type Table struct {
	Headers []string
	Rows    [][]string
	Render  Render
	// CellStyle styles one cell. Return the string unchanged to skip styling.
	CellStyle func(row, col int, cell string) string
}

const tableGap = "  "

// String formats the table with fixed column widths (ANSI-aware).
func (t Table) String() string {
	if len(t.Headers) == 0 {
		return ""
	}
	cols := len(t.Headers)
	widths := columnWidths(t.Headers, t.Rows, cols)

	var b strings.Builder
	writeRow(&b, t.Headers, widths, -1, t.Render, t.CellStyle, true)
	for ri, row := range t.Rows {
		writeRow(&b, row, widths, ri, t.Render, t.CellStyle, false)
	}
	return b.String()
}

func columnWidths(headers []string, rows [][]string, cols int) []int {
	widths := make([]int, cols)
	for i, h := range headers {
		widths[i] = lipgloss.Width(h)
	}
	for _, row := range rows {
		for i := 0; i < cols && i < len(row); i++ {
			if w := lipgloss.Width(row[i]); w > widths[i] {
				widths[i] = w
			}
		}
	}
	return widths
}

func writeRow(b *strings.Builder, cells []string, widths []int, rowIdx int, ren Render, style func(int, int, string) string, header bool) {
	parts := make([]string, len(widths))
	for i := range widths {
		plain := ""
		if i < len(cells) {
			plain = cells[i]
		}
		var styled string
		switch {
		case header && ren.Color:
			styled = headerStyle.Render(plain)
		case header:
			styled = plain
		case style != nil:
			styled = style(rowIdx, i, plain)
		default:
			styled = plain
		}
		parts[i] = padCell(styled, widths[i])
	}
	fmt.Fprintln(b, strings.Join(parts, tableGap))
}

func padCell(s string, width int) string {
	if lipgloss.Width(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}
