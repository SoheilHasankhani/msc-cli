package gitops

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func TestExecClonePullLsRemoteLocalBare(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	bare := filepath.Join(root, "src.git")
	if err := exec.Command("git", "init", "--bare", bare).Run(); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(root, "seed")
	if err := exec.Command("git", "clone", bare, work).Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = exec.Command("git", "-C", work, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", work, "config", "user.name", "test").Run()
	if err := exec.Command("git", "-C", work, "add", "README").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", work, "commit", "-m", "init").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", work, "push", "origin", "HEAD").Run(); err != nil {
		t.Fatal(err)
	}

	e := Exec{}
	ctx := context.Background()
	ok, err := e.LsRemote(ctx, bare)
	if err != nil || !ok {
		t.Fatalf("ls-remote accessible=%v err=%v", ok, err)
	}

	dest := filepath.Join(root, "clone")
	if err := e.Clone(ctx, bare, dest, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "README")); err != nil {
		t.Fatal(err)
	}
	if err := e.Pull(ctx, dest, nil); err != nil {
		t.Fatal(err)
	}
}

func TestExecLsRemoteMissingIsNotAccessible(t *testing.T) {
	requireGit(t)

	e := Exec{}
	ok, err := e.LsRemote(context.Background(), filepath.Join(t.TempDir(), "nope.git"))
	if ok {
		t.Fatal("missing repo should not be accessible")
	}
	if err == nil {
		t.Fatal("expected an error for a missing local path")
	}
}
