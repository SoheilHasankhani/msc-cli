package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTableColumnAlignmentPlain(t *testing.T) {
	t.Parallel()
	headers := []string{"CHECK", "STATUS", "DETAIL"}
	rows := [][]string{
		{"git", "PASS", "git found"},
		{"docker", "PASS", "Docker Engine reachable"},
	}
	got := Table{Headers: headers, Rows: rows}.String()
	assertColumnsAlign(t, got, columnWidths(headers, rows, len(headers)))
}

func TestTableColumnAlignmentColored(t *testing.T) {
	t.Parallel()
	headers := []string{"SERVICE", "MODE", "CONTAINER", "NGINX"}
	rows := [][]string{
		{"identity.api", "docker", "up", "container"},
		{"bffaggregator", "source", "down", "host.docker.internal:5010"},
	}
	got := Table{
		Headers: headers,
		Rows:    rows,
		Render:  Render{Color: true},
		CellStyle: func(_ int, col int, cell string) string {
			switch col {
			case 0:
				return nameStyle.Render(cell)
			case 1:
				return modePaint(Render{Color: true}, cell)
			case 2:
				return containerPaint(Render{Color: true}, cell == "up")
			default:
				return dimStyle.Render(cell)
			}
		},
	}.String()
	assertColumnsAlign(t, got, columnWidths(headers, rows, len(headers)))
	plain := ansi.Strip(strings.Split(strings.TrimSuffix(got, "\n"), "\n")[1])
	if !strings.Contains(plain, "up         container") {
		t.Fatalf("CONTAINER/NGINX gap collapsed:\n%s", got)
	}
}

func assertColumnsAlign(t *testing.T, table string, widths []int) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(table, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("need header + row, got %q", table)
	}
	starts := columnStarts(widths)
	for li, line := range lines {
		plain := ansi.Strip(line)
		for c, start := range starts {
			if start >= len(plain) {
				t.Fatalf("line %d too short for column %d: %q", li, c, plain)
			}
			end := len(plain)
			if c+1 < len(starts) {
				end = starts[c+1] - len(tableGap)
			}
			if end <= start {
				t.Fatalf("line %d column %d empty: %q", li, c, plain)
			}
			if plain[start] == ' ' && strings.TrimSpace(plain[start:end]) == "" {
				t.Fatalf("line %d column %d starts with padding gap: %q", li, c, plain)
			}
		}
	}
}

func columnStarts(widths []int) []int {
	starts := make([]int, len(widths))
	pos := 0
	for i, w := range widths {
		starts[i] = pos
		pos += w
		if i < len(widths)-1 {
			pos += len(tableGap)
		}
	}
	return starts
}
