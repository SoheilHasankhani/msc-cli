package doctor

import (
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/ui"
)

func TestFormatReportTable(t *testing.T) {
	t.Parallel()

	got := Format(Report{Checks: []Check{
		{Name: "git", Status: StatusPass, Message: "git 2.40"},
		{Name: "docker", Status: StatusFail, Message: "cannot reach Docker Engine", Next: "install and start Docker", Fix: FixNone},
		{Name: "hosts", Status: StatusFail, Message: "missing wallet.isos.local", Next: "doctor --fix (needs elevation)", Fix: FixHosts},
	}}, ui.Render{})
	for _, want := range []string{"CHECK", "git", "PASS", "docker", "FAIL", "hosts", "wallet.isos.local"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestReportHasFail(t *testing.T) {
	t.Parallel()
	if (Report{Checks: []Check{{Status: StatusPass}}}).HasFail() {
		t.Fatal("pass-only")
	}
	if !(Report{Checks: []Check{{Status: StatusFail}}}).HasFail() {
		t.Fatal("expected fail")
	}
}
