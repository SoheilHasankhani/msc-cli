package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteComposeKeepsProject(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--project", "isos", "compose", "config", "--images"})
	err := cmd.Execute()
	if err != nil && strings.Contains(err.Error(), "no project specified") {
		t.Fatalf("project flag lost on compose: %v\nout=%s", err, out.String())
	}
}

func TestComposeSeesRootProjectFlag(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	if err := cmd.PersistentFlags().Parse([]string{"--project", "isos"}); err != nil {
		t.Fatal(err)
	}
	name, err := projectName(cmd)
	if err != nil || name != "isos" {
		t.Fatalf("%q %v", name, err)
	}
}

func TestComposeAndGitAreThinPassthrough(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	compose, _, err := root.Find([]string{"compose"})
	if err != nil || compose == nil || !compose.DisableFlagParsing {
		t.Fatalf("compose: %v %#v", err, compose)
	}
	git, _, err := root.Find([]string{"git"})
	if err != nil || git == nil || !git.DisableFlagParsing {
		t.Fatalf("git: %v %#v", err, git)
	}
}

func TestCompletionCommandExists(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	c, _, err := cmd.Find([]string{"completion"})
	if err != nil || c == nil {
		t.Fatal("completion command missing")
	}
}

func TestRootHelpListsPassthrough(t *testing.T) {
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
	for _, want := range []string{"compose", "git", "completion", "self-update", "support-bundle"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}
