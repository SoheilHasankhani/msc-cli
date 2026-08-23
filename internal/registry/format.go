package registry

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/ui"
)

// FormatList renders registered projects. Pass ui.For(stdout) for color.
func FormatList(r *Registry, ren ui.Render) string {
	if r == nil || len(r.Projects) == 0 {
		return ui.Dim(ren, "no projects registered; run msc init --repo <url>\n")
	}
	names := r.Names()
	sort.Strings(names)
	var rows [][]string
	for _, name := range names {
		e := r.Projects[name]
		rows = append(rows, []string{name, e.Path, e.GitHostURL})
	}
	tbl := ui.Table{
		Headers: []string{"NAME", "PATH", "GIT HOST"},
		Rows:    rows,
		Render:  ren,
		CellStyle: func(_ int, col int, cell string) string {
			switch col {
			case 0:
				return ui.Name(ren, cell)
			default:
				if ren.Color {
					return ui.Dim(ren, cell)
				}
				return cell
			}
		},
	}
	return tbl.String()
}

// PathErrorMessage describes a broken registry path for non-interactive use.
func PathErrorMessage(name, path string, status PathStatus) string {
	return fmt.Sprintf("registered path for %q is not a valid project (%v): %s", name, status, path)
}

// ApplyRepair executes a repair action on the registry file.
func ApplyRepair(reg *Registry, file, name string, action RepairAction, newPath string) error {
	switch action {
	case RepairRemove:
		if err := reg.Remove(name); err != nil {
			return err
		}
		return reg.Save(file)
	case RepairRelink:
		if err := reg.Relink(name, newPath); err != nil {
			return err
		}
		return reg.Save(file)
	default:
		return fmt.Errorf("unknown repair action")
	}
}

// FormatRepairHint returns CLI guidance when prompts are disabled.
func FormatRepairHint(name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "run: msc projects relink %s --path <dir>\n", name)
	fmt.Fprintf(&b, " or: msc projects remove %s\n", name)
	return strings.TrimSpace(b.String())
}
