package gitops

import (
	"strings"
	"testing"
)

func TestClassifyAccess(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		stderr string
		want   Kind
	}{
		{"ok empty", "", KindOther},
		{"publickey", "git@gitlab.example.com: Permission denied (publickey).", KindDenied},
		{"gitlab hidden", "ERROR: The project you were looking for could not be found.", KindDenied},
		{"github not found", "ERROR: Repository not found.", KindDenied},
		{"access denied", "remote: Access denied\nfatal: Could not read from remote repository.", KindDenied},
		{"no agent", "Could not open a connection to your authentication agent.", KindNoAgent},
		{"no identities", "ERROR: no identities\nfatal: Could not read from remote repository.", KindNoAgent},
		{"ssh auth sock", "error: SSH_AUTH_SOCK not set", KindNoAgent},
		{"host key", "Host key verification failed.", KindOther},
		{"tcp timeout", "ssh: connect to host git.example.com port 22: Connection timed out", KindUnreachable},
		{"refused", "ssh: connect to host git.example.com port 22: Connection refused", KindUnreachable},
		{"resolve", "ssh: Could not resolve hostname gitlab.example.com", KindUnreachable},
		{"no route", "ssh: connect to host git.example.com port 22: No route to host", KindUnreachable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Classify(tc.stderr); got != tc.want {
				t.Fatalf("Classify() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAccessMessagesAreActionable(t *testing.T) {
	t.Parallel()

	denied := Message(KindDenied, "git@gitlab.example.com:sos/doctor.git")
	if !strings.Contains(strings.ToLower(denied), "access") || !strings.Contains(denied, "SSH") {
		t.Fatalf("denied message not actionable: %q", denied)
	}
	agent := Message(KindNoAgent, "git@gitlab.example.com:sos/doctor.git")
	if !strings.Contains(agent, "SSH") || !strings.Contains(strings.ToLower(agent), "agent") {
		t.Fatalf("no-agent message not actionable: %q", agent)
	}
	unreach := Message(KindUnreachable, "git@git.isos.clinic:sos/identity-api.git")
	if !strings.Contains(strings.ToLower(unreach), "vpn") && !strings.Contains(strings.ToLower(unreach), "ssh") {
		t.Fatalf("unreachable message not actionable: %q", unreach)
	}
}
