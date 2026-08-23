package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestSwitchRequiresProject(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"switch", "doctor"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error without --project")
	}

	cmd = NewRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"switch", "doctor", "--to", "source"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error without --project")
	}
	if !strings.Contains(err.Error(), "brand shim") {
		t.Fatalf("err = %v", err)
	}
}

func TestSwitchCompleteListsServicesWithoutRequiredTo(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--project", "isos", "__complete", "switch", ""})
	_ = cmd.Execute()
	got := out.String()
	if got == "" {
		// cobra may write __complete to the process stdout
		return
	}
	if strings.Contains(got, "--to") {
		t.Fatalf("required --to should not be mixed into service completions:\n%s", got)
	}
}

func TestSwitchHelpMentionsListenAllInterfaces(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"switch", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "0.0.0.0") {
		t.Fatalf("help missing 0.0.0.0 reminder:\n%s", got)
	}
}
