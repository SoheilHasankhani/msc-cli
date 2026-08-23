package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestSelfUpdateCommandExists(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	cmd, _, err := root.Find([]string{"self-update"})
	if err != nil || cmd == nil || cmd.Name() != "self-update" {
		t.Fatalf("self-update: %v %#v", err, cmd)
	}
	if cmd.Flags().Lookup("check") == nil || cmd.Flags().Lookup("force") == nil {
		t.Fatal("expected --check and --force")
	}
}

func TestSelfUpdateHelp(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"self-update", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"GitHub", "checksums", "--check", "--force"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}
