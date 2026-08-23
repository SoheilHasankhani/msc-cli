package gitops

import "testing"

func TestParseIdentitySSH(t *testing.T) {
	t.Parallel()

	remote, base, err := ParseIdentity("git@gitlab.example.com:sos/meta.git")
	if err != nil {
		t.Fatal(err)
	}
	if remote != "git@gitlab.example.com:sos/meta.git" {
		t.Fatalf("remote = %q", remote)
	}
	if base != "https://gitlab.example.com" {
		t.Fatalf("base = %q", base)
	}
}

func TestParseIdentityHTTPS(t *testing.T) {
	t.Parallel()

	remote, base, err := ParseIdentity("https://gitlab.example.com/sos/meta.git")
	if err != nil {
		t.Fatal(err)
	}
	if remote != "https://gitlab.example.com/sos/meta.git" {
		t.Fatalf("remote = %q", remote)
	}
	if base != "https://gitlab.example.com" {
		t.Fatalf("base = %q", base)
	}
}

func TestParseIdentityRejectsEmpty(t *testing.T) {
	t.Parallel()
	if _, _, err := ParseIdentity(""); err == nil {
		t.Fatal("expected error")
	}
}
