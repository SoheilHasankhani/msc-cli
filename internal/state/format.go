package state

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/ui"
)

// FormatStatus renders live service states. Pass ui.For(stdout) for color.
func FormatStatus(states map[string]ServiceState, warnings []string, ren ui.Render) string {
	var b strings.Builder
	for _, w := range warnings {
		fmt.Fprintf(&b, "%s\n", ui.Warn(ren, "warning: "+w))
	}
	names := make([]string, 0, len(states))
	for name := range states {
		names = append(names, name)
	}
	sort.Strings(names)
	var rows [][]string
	for _, name := range names {
		s := states[name]
		ctr := "down"
		if s.ContainerUp {
			ctr = "up"
		}
		rows = append(rows, []string{s.ComposeService, string(s.Mode), ctr, s.NginxTarget})
	}
	tbl := ui.Table{
		Headers: []string{"SERVICE", "MODE", "CONTAINER", "NGINX"},
		Rows:    rows,
		Render:  ren,
		CellStyle: func(_ int, col int, cell string) string {
			plain := strings.TrimSpace(cell)
			switch col {
			case 0:
				return ui.Name(ren, plain)
			case 1:
				return ui.Mode(ren, plain)
			case 2:
				return ui.Container(ren, plain == "up")
			default:
				if ren.Color {
					return ui.Dim(ren, plain)
				}
				return plain
			}
		},
	}
	b.WriteString(tbl.String())
	return b.String()
}
