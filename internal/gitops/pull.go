package gitops

import (
	"errors"
	"fmt"
	"strings"
)

// ErrPullTransient marks a network/SSH failure that pullWithRetry may repeat.
var ErrPullTransient = errors.New("pull transient")

// ErrPullConflict marks a pull failure that should warn and not stop other repos.
var ErrPullConflict = errors.New("pull conflict")

// DefaultPullRetries is how many times a transient network pull failure is retried.
const DefaultPullRetries = 3

// IsPullConflict reports whether git pull --ff-only failed due to local state.
func IsPullConflict(stderr string) bool {
	s := strings.ToLower(stderr)
	return strings.Contains(s, "not possible to fast-forward") ||
		strings.Contains(s, "diverging branches") ||
		strings.Contains(s, "local changes") ||
		strings.Contains(s, "would be overwritten") ||
		strings.Contains(s, "cannot pull with rebase") ||
		strings.Contains(s, "merge conflict") ||
		strings.Contains(s, "uncommitted changes") ||
		strings.Contains(s, "your branch and") && strings.Contains(s, "have diverged")
}

// IsRetryablePullStderr reports transient SSH/network failures worth retrying.
func IsRetryablePullStderr(stderr string) bool {
	if stderr == "" {
		return false
	}
	if IsPullConflict(stderr) {
		return false
	}
	kind := Classify(stderr)
	if kind == KindDenied || kind == KindNoAgent {
		return false
	}
	if kind == KindUnreachable {
		return true
	}
	s := strings.ToLower(stderr)
	return strings.Contains(s, "connection reset") ||
		strings.Contains(s, "connection timed out") ||
		strings.Contains(s, "operation timed out") ||
		strings.Contains(s, "kex_exchange_identification") ||
		strings.Contains(s, "could not read from remote repository") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "rpc failed") ||
		strings.Contains(s, "early eof") ||
		strings.Contains(s, "remote end hung up") ||
		strings.Contains(s, "index-pack failed") ||
		strings.Contains(s, "the tls connection") ||
		strings.Contains(s, "ssl") && strings.Contains(s, "syscall")
}

// PullNetworkMessage is a one-line summary for transient pull/network failures.
func PullNetworkMessage(stderr string) string {
	s := strings.ToLower(stderr)
	switch {
	case strings.Contains(s, "connection reset"), strings.Contains(s, "kex_exchange_identification"):
		return "SSH connection reset — check VPN/network"
	case strings.Contains(s, "connection timed out"), strings.Contains(s, "operation timed out"):
		return "SSH timed out — check VPN/network"
	case strings.Contains(s, "could not read from remote repository"):
		return "could not read from remote — check VPN/network or SSH access"
	case strings.Contains(s, "broken pipe"):
		return "SSH connection dropped — check VPN/network"
	default:
		return "network error during pull — check VPN/network"
	}
}

// PullUserMessage turns git pull stderr into a short warning for one repo.
func PullUserMessage(stderr string) string {
	s := strings.ToLower(stderr)
	switch {
	case strings.Contains(s, "not possible to fast-forward"):
		return "cannot fast-forward; local branch has diverged — resolve manually, then pull again"
	case strings.Contains(s, "have diverged"):
		return "local branch has diverged from remote — resolve manually, then pull again"
	case strings.Contains(s, "local changes"), strings.Contains(s, "would be overwritten"), strings.Contains(s, "uncommitted changes"):
		return "local uncommitted changes block the pull — commit, stash, or discard them first"
	case strings.Contains(s, "merge conflict"):
		return "merge conflict — resolve manually in the repo, then pull again"
	case IsRetryablePullStderr(stderr):
		return PullNetworkMessage(stderr)
	default:
		return firstLine(stderr, "pull failed — resolve git state manually in the repo")
	}
}

// WrapPullConflict returns an error tagged with ErrPullConflict.
func WrapPullConflict(stderr string) error {
	return fmt.Errorf("%s: %w", PullUserMessage(stderr), ErrPullConflict)
}

// FormatPullError returns a single-line message suitable for progress UI and warnings.
func FormatPullError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, suffix := range []string{": pull transient", ": pull conflict"} {
		if i := strings.Index(msg, suffix); i >= 0 {
			msg = msg[:i]
			break
		}
	}
	return firstLine(msg, msg)
}

func cloneError(stderr, repoURL string) error {
	kind := Classify(stderr)
	switch kind {
	case KindDenied, KindNoAgent:
		return fmt.Errorf("%s", Message(kind, repoURL))
	case KindUnreachable:
		return fmt.Errorf("%s: %w", PullNetworkMessage(stderr), ErrPullTransient)
	}
	if IsRetryablePullStderr(stderr) {
		return fmt.Errorf("%s: %w", PullNetworkMessage(stderr), ErrPullTransient)
	}
	if stderr != "" {
		return fmt.Errorf("git clone %s: %s", repoURL, firstLine(stderr, "clone failed"))
	}
	return fmt.Errorf("git clone %s failed", repoURL)
}

func firstLine(text, fallback string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) > 120 {
				return line[:117] + "..."
			}
			return line
		}
	}
	return fallback
}
