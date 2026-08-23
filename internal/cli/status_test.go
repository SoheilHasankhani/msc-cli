package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatusRequiresProject(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"status"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error without --project")
	}
	if !strings.Contains(err.Error(), "brand shim") {
		t.Fatalf("err = %v", err)
	}
}

func TestHelpListsStatusUpDown(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, name := range []string{"status", "up", "down", "switch", "sync", "doctor", "init", "projects"} {
		if !strings.Contains(got, name) {
			t.Fatalf("help missing %s:\n%s", name, got)
		}
	}
}
