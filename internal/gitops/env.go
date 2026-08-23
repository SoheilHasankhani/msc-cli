package gitops

import (
	"strings"
	"time"
)

// DefaultSSHCommand is used when the developer has not set GIT_SSH_COMMAND.
// BatchMode prevents hidden passphrase/host-key prompts; ConnectTimeout
// stops a dead Git host from looking like a hang.
const DefaultSSHCommand = "ssh -o BatchMode=yes -o ConnectTimeout=8"

// DefaultLsRemoteTimeout is a hard cap around each git ls-remote.
const DefaultLsRemoteTimeout = 15 * time.Second

// DefaultProbeParallel is how many ls-remote calls run at once.
const DefaultProbeParallel = 8

// NonInteractiveGitEnv copies base and forces git/ssh not to wait on a TTY.
func NonInteractiveGitEnv(base []string) []string {
	out := append([]string{}, base...)
	out = append(out, "GIT_TERMINAL_PROMPT=0")
	if !envHas(base, "GIT_SSH_COMMAND=") {
		out = append(out, "GIT_SSH_COMMAND="+DefaultSSHCommand)
	}
	return out
}

func envHas(env []string, prefix string) bool {
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}
