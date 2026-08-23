package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCacheMissingIsEmpty(t *testing.T) {
	t.Parallel()

	s, err := LoadCache(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Services) != 0 {
		t.Fatalf("expected empty, got %#v", s)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "isos.json")
	in := &Snapshot{Services: map[string]ServiceState{
		"doctor": {ComposeService: "doctor", Mode: ModeSource, ContainerUp: false, NginxTarget: NginxSourceTarget(5010)},
	}}
	if err := SaveCache(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := LoadCache(path)
	if err != nil {
		t.Fatal(err)
	}
	got := out.Services["doctor"]
	if got.Mode != ModeSource || got.NginxTarget != NginxSourceTarget(5010) {
		t.Fatalf("round-trip = %#v", got)
	}
}

func TestSyncLiveUpdatesContainerUpAndKeepsSwitchTime(t *testing.T) {
	t.Parallel()

	switched := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	snap := &Snapshot{Services: map[string]ServiceState{
		"wallet":  {ComposeService: "wallet", Mode: ModeDocker, ContainerUp: false, NginxTarget: NginxTargetContainer, LastSwitchedAt: &switched},
		"patient": {ComposeService: "patient", Mode: ModeSource, ContainerUp: false, NginxTarget: NginxSourceTarget(9000), LastSwitchedAt: &switched},
	}}
	live := map[string]ServiceState{
		"wallet":  {ComposeService: "wallet", Mode: ModeDocker, ContainerUp: true, NginxTarget: NginxTargetContainer},
		"patient": {ComposeService: "patient", Mode: ModeSource, ContainerUp: false, NginxTarget: NginxSourceTarget(9000)},
	}

	got := snap.SyncLive(live)
	if !got.Services["wallet"].ContainerUp {
		t.Fatal("wallet should be up after live sync")
	}
	if got.Services["wallet"].LastSwitchedAt == nil || !got.Services["wallet"].LastSwitchedAt.Equal(switched) {
		t.Fatalf("wallet LastSwitchedAt = %#v", got.Services["wallet"].LastSwitchedAt)
	}
	if got.Services["patient"].Mode != ModeSource || got.Services["patient"].ContainerUp {
		t.Fatalf("patient = %#v", got.Services["patient"])
	}
}

func TestCachePathUsesProjectName(t *testing.T) {
	t.Parallel()

	got := CachePath("/cfg/state", "isos")
	if filepath.Base(got) != "isos.json" {
		t.Fatalf("CachePath = %q", got)
	}
}
