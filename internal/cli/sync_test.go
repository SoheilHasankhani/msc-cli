package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestSyncRequiresProject(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"sync"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error without --project")
	}
	if !strings.Contains(err.Error(), "brand shim") {
		t.Fatalf("err = %v", err)
	}
}

func TestSyncHelpMentionsSSH(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"sync", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "SSH") || !strings.Contains(got, "ls-remote") || !strings.Contains(got, "--list") {
		t.Fatalf("help missing SSH/ls-remote/--list:\n%s", got)
	}
}
