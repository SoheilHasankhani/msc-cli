package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitRequiresRepoWhenPathMissing(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"init", "--path", filepath.Join(t.TempDir(), "missing")})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected --repo required for missing path")
	}
}

func TestInitHelpMentionsAsAndNoCommit(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"init", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "--as") || !strings.Contains(got, "never auto-commits") {
		t.Fatalf("%s", got)
	}
}

func TestProjectsHelp(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"projects", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, name := range []string{"list", "remove", "relink"} {
		if !strings.Contains(got, name) {
			t.Fatalf("missing %s:\n%s", name, got)
		}
	}
}
