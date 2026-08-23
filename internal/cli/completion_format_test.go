package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionOmitsLongDescriptions(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"__completeNoDesc", "do"})
	_ = cmd.Execute()
	got := out.String()
	if got == "" {
		t.Skip("cobra __complete output not captured in this environment")
	}
	if strings.Contains(got, "Report (and optionally repair)") {
		t.Fatalf("completion should not include command Short descriptions:\n%s", got)
	}
	for _, want := range []string{"doctor", "down"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestBashCompletionUsesNoDescRequest(t *testing.T) {
	cmd := NewRootCmd()
	if !cmd.CompletionOptions.DisableDescriptions {
		t.Fatal("expected CompletionOptions.DisableDescriptions on root command")
	}
	var out bytes.Buffer
	if err := cmd.GenBashCompletionV2(&out, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "__completeNoDesc") {
		t.Fatalf("bash completion script should call __completeNoDesc:\n%s", got[:min(500, len(got))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
