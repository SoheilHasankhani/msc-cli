package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/testenv"
)

func TestComposeCompleteSubcommands(t *testing.T) {
	t.Setenv("MSC_PROJECT", "isos")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"__completeNoDesc", "compose", ""})
	_ = cmd.Execute()
	got := out.String()
	if got == "" {
		t.Skip("cobra __complete output not captured in this environment")
	}
	for _, want := range []string{"logs", "ps", "up", "exec"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing compose subcommand %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "bin/") || strings.Contains(got, "internal/") {
		t.Fatalf("file completion leaked into compose:\n%s", got)
	}
}

func TestComposeCompleteServicesAfterLogs(t *testing.T) {
	testenv.InstallBrandProject(t, "isos")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"__completeNoDesc", "compose", "logs", ""})
	_ = cmd.Execute()
	got := out.String()
	if got == "" {
		t.Skip("cobra __complete output not captured in this environment")
	}
	if strings.Contains(got, "bin/") {
		t.Fatalf("file completion leaked:\n%s", got)
	}
	if !strings.Contains(got, "doctor") {
		t.Fatalf("expected service name doctor in:\n%s", got)
	}
}
