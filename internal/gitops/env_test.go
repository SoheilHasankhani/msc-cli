package gitops

import (
	"strings"
	"testing"
)

func TestNonInteractiveGitEnv(t *testing.T) {
	t.Parallel()

	got := NonInteractiveGitEnv([]string{"PATH=/bin", "HOME=/tmp"})
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "GIT_TERMINAL_PROMPT=0") {
		t.Fatalf("missing GIT_TERMINAL_PROMPT:\n%s", joined)
	}
	if !strings.Contains(joined, "BatchMode=yes") || !strings.Contains(joined, "ConnectTimeout=") {
		t.Fatalf("missing SSH batch/timeout:\n%s", joined)
	}
}

func TestNonInteractiveGitEnvKeepsExistingSSHCommand(t *testing.T) {
	t.Parallel()

	got := NonInteractiveGitEnv([]string{"GIT_SSH_COMMAND=ssh -J bastion"})
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "GIT_SSH_COMMAND=ssh -J bastion") {
		t.Fatalf("%s", joined)
	}
	if strings.Count(joined, "GIT_SSH_COMMAND=") != 1 {
		t.Fatalf("overwrote GIT_SSH_COMMAND:\n%s", joined)
	}
}
