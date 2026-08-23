package dockerapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecComposeBuildsExpectedArgs(t *testing.T) {
	t.Parallel()

	e := ExecCompose{}
	cmd := e.cmd(t.Context(), t.TempDir(), "local/docker-compose.yml", []string{"standard"}, "up", "-d", "--pull", "never")
	got := strings.Join(cmd.Args, " ")
	want := "docker compose -f local/docker-compose.yml --profile standard up -d --pull never"
	if got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestExecComposeIncludesSiblingOverlay(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "local", "docker-compose.msc.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := ExecCompose{}
	cmd := e.cmd(t.Context(), root, "local/docker-compose.yml", nil, "up", "-d")
	got := strings.Join(cmd.Args, " ")
	want := "docker compose -f local/docker-compose.yml -f local/docker-compose.msc.yml up -d"
	if got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestComposeUpArgsOmitsDepsForSelectedServices(t *testing.T) {
	t.Parallel()

	got := strings.Join(composeUpArgs(ComposeRunOpts{NoDeps: true, Services: []string{"nginx", "wallet"}}), " ")
	want := "up -d --pull never --remove-orphans --no-deps nginx wallet"
	if got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestExecComposeSupportsMultipleProfiles(t *testing.T) {
	t.Parallel()

	e := ExecCompose{}
	cmd := e.cmd(t.Context(), t.TempDir(), "local/docker-compose.yml", []string{"backend", "frontend"}, "down")
	got := strings.Join(cmd.Args, " ")
	want := "docker compose -f local/docker-compose.yml --profile backend --profile frontend down"
	if got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}
