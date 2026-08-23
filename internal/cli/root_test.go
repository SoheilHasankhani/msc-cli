package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/version"
)

func TestRootHasProjectFlag(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	flag := cmd.PersistentFlags().Lookup("project")
	if flag == nil {
		t.Fatal("expected persistent --project flag on the root command")
	}
	if !cmd.TraverseChildren {
		t.Fatal("TraverseChildren required so --project is parsed before compose/git")
	}
}

func TestRootVersionPrintsIdentity(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(--version): %v", err)
	}

	got := strings.TrimSpace(out.String())
	if !strings.Contains(got, version.String()) {
		t.Fatalf("version output %q does not contain %q", got, version.String())
	}
}

func TestRootHelpMentionsEngine(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(--help): %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "msc") {
		t.Fatalf("help output does not mention msc:\n%s", got)
	}
	if strings.Contains(got, "__elevated-do") {
		t.Fatal("hidden elevated helper must not appear in help")
	}
}
