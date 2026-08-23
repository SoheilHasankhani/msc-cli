package gitops

import (
	"fmt"
	"strings"
)

// Kind is how a git SSH failure should be presented.
type Kind int

const (
	KindOK Kind = iota
	KindDenied
	KindNoAgent
	KindUnreachable
	KindOther
)

// Classify inspects git stderr. Permission-denied and "not found" (some Git hosts
// hide private repos) are access denials; missing agent/keys are distinct.
func Classify(stderr string) Kind {
	s := strings.ToLower(stderr)
	switch {
	case strings.Contains(s, "could not open a connection to your authentication agent"),
		strings.Contains(s, "ssh_auth_sock"),
		strings.Contains(s, "no identities"):
		return KindNoAgent
	case strings.Contains(s, "permission denied"),
		strings.Contains(s, "the project you were looking for could not be found"),
		strings.Contains(s, "repository not found"),
		strings.Contains(s, "access denied"):
		return KindDenied
	case strings.Contains(s, "connection timed out"),
		strings.Contains(s, "connection refused"),
		strings.Contains(s, "network is unreachable"),
		strings.Contains(s, "no route to host"),
		strings.Contains(s, "could not resolve hostname"),
		strings.Contains(s, "operation timed out"):
		return KindUnreachable
	default:
		return KindOther
	}
}

// Message is the user-facing translation of a git access failure.
func Message(kind Kind, repoURL string) string {
	switch kind {
	case KindDenied:
		return fmt.Sprintf("no access to %s (or your SSH key is not registered with the Git host) — ask the project admin to grant access", repoURL)
	case KindNoAgent:
		return "no SSH agent or key is available; start ssh-agent and add your key, then retry"
	case KindUnreachable:
		return fmt.Sprintf("cannot reach %s over SSH — connect to the VPN (if required) and check that port 22 is open, then retry", hostOf(repoURL))
	default:
		return fmt.Sprintf("git failed for %s", repoURL)
	}
}

// IsUnreachable reports a Git-host SSH connectivity failure (not an access denial).
func IsUnreachable(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "cannot reach") && strings.Contains(s, "over ssh")
}

func hostOf(repoURL string) string {
	s := repoURL
	if i := strings.Index(s, "@"); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.Index(s, ":"); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return repoURL
	}
	return s
}
