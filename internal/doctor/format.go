package doctor

import (
	"fmt"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/ui"
)

// Format renders the doctor report. Pass ui.For(stdout) from the CLI for color.
func Format(r Report, ren ui.Render) string {
	var rows [][]string
	for _, c := range r.Checks {
		detail := c.Message
		if c.Next != "" && c.Status != StatusPass {
			detail = c.Message + " — " + c.Next
		}
		rows = append(rows, []string{c.Name, string(c.Status), detail})
	}
	tbl := ui.Table{
		Headers: []string{"CHECK", "STATUS", "DETAIL"},
		Rows:    rows,
		Render:  ren,
		CellStyle: func(_ int, col int, cell string) string {
			if col == 1 {
				return statusCell(ren, cell)
			}
			if col == 2 && ren.Color {
				return ui.Dim(ren, cell)
			}
			return cell
		},
	}
	var b strings.Builder
	b.WriteString(tbl.String())
	for _, s := range r.Fixed {
		fmt.Fprintf(&b, "%s\n", ui.Success(ren, "fixed: "+s))
	}
	for _, s := range r.Skipped {
		line := "skipped: " + s
		if strings.Contains(strings.ToLower(s), "elevation") || strings.Contains(strings.ToLower(s), "sudo") {
			fmt.Fprintf(&b, "%s\n", ui.Box(ren, ui.Warn(ren, line)+"\n"+ui.Dim(ren, "Re-run msc doctor --fix in an interactive terminal so sudo / UAC can prompt.")))
			continue
		}
		fmt.Fprintf(&b, "%s\n", ui.Warn(ren, line))
	}
	return b.String()
}

func statusCell(ren ui.Render, status string) string {
	switch Status(strings.ToUpper(status)) {
	case StatusPass:
		return ui.Pass(ren, status)
	case StatusFail:
		return ui.Fail(ren, status)
	case StatusWarn:
		return ui.WarnStatus(ren, status)
	default:
		return status
	}
}
