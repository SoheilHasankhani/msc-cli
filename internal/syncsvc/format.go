package syncsvc

import (
	"fmt"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/gitops"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
)

// FormatPlan is the read-only sync listing (--list; denied repos omitted).
func FormatPlan(plan gitops.Plan, ren ui.Render) string {
	var b strings.Builder
	for _, r := range plan.Revoked {
		fmt.Fprintf(&b, "%s\n", ui.Warn(ren, fmt.Sprintf("warning: %s: %s", r.Name, r.Warning)))
	}
	if len(plan.Cloned) > 0 {
		fmt.Fprintf(&b, "%s\n", ui.Title(ren, "cloned:"))
		for _, r := range plan.Cloned {
			if r.Accessible {
				fmt.Fprintf(&b, "  %s\n", ui.Name(ren, r.Name))
			}
		}
	}
	if len(plan.Available) > 0 {
		fmt.Fprintf(&b, "%s\n", ui.Title(ren, "available to clone:"))
		for _, r := range plan.Available {
			fmt.Fprintf(&b, "  %s\n", ui.Name(ren, r.Name))
		}
	}
	if len(plan.Cloned) == 0 && len(plan.Available) == 0 {
		b.WriteString(ui.Dim(ren, "no accessible repos\n"))
	}
	fmt.Fprintf(&b, "%s\n", ui.Dim(ren, "run: msc sync to clone missing repos and pull updates (use --list to inspect only; --refresh to re-check access)"))
	return b.String()
}

// FormatUpdateResult prints revoked warnings, per-repo failures, and a summary.
func FormatUpdateResult(plan gitops.Plan, res UpdateResult, ren ui.Render) string {
	var b strings.Builder
	for _, r := range plan.Revoked {
		fmt.Fprintf(&b, "%s\n", ui.Warn(ren, fmt.Sprintf("warning: %s: %s", r.Name, r.Warning)))
	}
	for _, w := range res.Warnings {
		op := w.Op
		if op == "" {
			op = "sync"
		}
		fmt.Fprintf(&b, "%s\n", ui.Warn(ren, fmt.Sprintf("warning: %s (%s): %s", w.Name, op, w.Message)))
	}
	var parts []string
	if res.Cloned > 0 {
		parts = append(parts, fmt.Sprintf("cloned %d", res.Cloned))
	}
	if res.Pulled > 0 {
		parts = append(parts, fmt.Sprintf("pulled %d", res.Pulled))
	}
	switch len(parts) {
	case 0:
		if len(res.Warnings) > 0 {
			fmt.Fprintf(&b, "%s\n", ui.Dim(ren, "no repos were updated"))
		} else {
			fmt.Fprintf(&b, "%s\n", ui.Dim(ren, "already up to date"))
		}
	default:
		fmt.Fprintf(&b, "%s\n", ui.SuccessLine(ren, "synced "+strings.Join(parts, ", "), ""))
	}
	return b.String()
}

// FormatPullResult prints per-repo warnings and a success summary.
func FormatPullResult(res PullResult, ren ui.Render) string {
	var b strings.Builder
	for _, w := range res.Warnings {
		fmt.Fprintf(&b, "%s\n", ui.Warn(ren, fmt.Sprintf("warning: %s: %s", w.Name, w.Message)))
	}
	if res.Pulled > 0 {
		fmt.Fprintf(&b, "%s\n", ui.SuccessLine(ren, fmt.Sprintf("pulled %d repo(s)", res.Pulled), ""))
	} else if len(res.Warnings) > 0 {
		fmt.Fprintf(&b, "%s\n", ui.Dim(ren, "no repos were updated"))
	}
	return b.String()
}
