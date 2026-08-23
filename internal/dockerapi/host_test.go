package dockerapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveHostPrefersDOCKER_HOST(t *testing.T) {
	t.Parallel()
	got := ResolveHost(func(k string) string {
		if k == "DOCKER_HOST" {
			return "unix:///tmp/custom.sock"
		}
		return ""
	}, t.TempDir())
	if got != "unix:///tmp/custom.sock" {
		t.Fatal(got)
	}
}

func TestResolveHostReadsDesktopContext(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	meta := filepath.Join(home, ".docker", "contexts", "meta", "abc", "meta.json")
	if err := os.MkdirAll(filepath.Dir(meta), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".docker", "config.json")
	if err := os.WriteFile(cfg, []byte(`{"currentContext":"desktop-linux"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(meta, []byte(`{"Name":"desktop-linux","Endpoints":{"docker":{"Host":"unix:///tmp/desktop.sock"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ResolveHost(func(string) string { return "" }, home)
	if got != "unix:///tmp/desktop.sock" {
		t.Fatal(got)
	}
}

func TestResolveHostDOCKER_CONTEXTOverridesCurrent(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".docker", "contexts", "meta", "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".docker", "contexts", "meta", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".docker", "config.json"), []byte(`{"currentContext":"desktop-linux"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".docker", "contexts", "meta", "a", "meta.json"), []byte(`{"Name":"desktop-linux","Endpoints":{"docker":{"Host":"unix:///tmp/desktop.sock"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".docker", "contexts", "meta", "b", "meta.json"), []byte(`{"Name":"other","Endpoints":{"docker":{"Host":"unix:///tmp/other.sock"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ResolveHost(func(k string) string {
		if k == "DOCKER_CONTEXT" {
			return "other"
		}
		return ""
	}, home)
	if got != "unix:///tmp/other.sock" {
		t.Fatal(got)
	}
}
