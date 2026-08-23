//go:build windows

package dockerapi

import (
	"net/url"
	"testing"
)

func TestNpipePath(t *testing.T) {
	t.Parallel()

	u, err := url.Parse("npipe:////./pipe/docker_engine")
	if err != nil {
		t.Fatal(err)
	}
	got, err := npipePath(u)
	if err != nil {
		t.Fatal(err)
	}
	want := `\\.\pipe\docker_engine`
	if got != want {
		t.Fatalf("npipePath = %q, want %q", got, want)
	}
}

func TestResolveHostFallsBackToWindowsPipe(t *testing.T) {
	t.Parallel()

	got := ResolveHost(func(string) string { return "" }, t.TempDir())
	if got != platformDefaultHost {
		t.Fatalf("got %q, want %q", got, platformDefaultHost)
	}
}
