package gitops

import "testing"

func TestSSHURLFromGitHostBase(t *testing.T) {
	t.Parallel()

	got, err := SSHURL("https://gitlab.example.com", "sos/identity-api")
	if err != nil {
		t.Fatal(err)
	}
	if got != "git@gitlab.example.com:sos/identity-api.git" {
		t.Fatalf("got %q", got)
	}

	got, err = SSHURL("https://gitlab.example.com/", "sos/identity-api.git")
	if err != nil {
		t.Fatal(err)
	}
	if got != "git@gitlab.example.com:sos/identity-api.git" {
		t.Fatalf("got %q", got)
	}
}

func TestSSHURLPassthrough(t *testing.T) {
	t.Parallel()

	in := "git@gitlab.example.com:sos/doctor.git"
	got, err := SSHURL("https://gitlab.example.com", in)
	if err != nil || got != in {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestSSHURLRejectsEmpty(t *testing.T) {
	t.Parallel()

	if _, err := SSHURL("", "sos/x"); err == nil {
		t.Fatal("empty base")
	}
	if _, err := SSHURL("https://gitlab.example.com", ""); err == nil {
		t.Fatal("empty repo")
	}
	if _, err := SSHURL("://bad", "sos/x"); err == nil {
		t.Fatal("bad base")
	}
}
