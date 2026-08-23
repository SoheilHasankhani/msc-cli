//go:build !windows

package dockerapi

import (
	"testing"
)

func TestResolveHostFallsBackToEngineSocket(t *testing.T) {
	t.Parallel()
	got := ResolveHost(func(string) string { return "" }, t.TempDir())
	if got != DefaultHost {
		t.Fatal(got)
	}
}
