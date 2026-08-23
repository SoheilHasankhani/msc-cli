package gitops

import (
	"path/filepath"
	"testing"
	"time"
)

func TestAccessCacheRoundTripAndTTL(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "access.json")
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	c := &AccessCache{
		CheckedAt: now,
		Repos:     map[string]bool{"git@host:a.git": true, "git@host:b.git": false},
	}
	if err := SaveAccessCache(path, c); err != nil {
		t.Fatal(err)
	}

	got, err := LoadAccessCache(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Valid(now.Add(3*24*time.Hour), DefaultAccessTTL) {
		t.Fatal("should be valid within TTL")
	}
	if got.Valid(now.Add(8*24*time.Hour), DefaultAccessTTL) {
		t.Fatal("should expire after TTL")
	}
	if !got.Repos["git@host:a.git"] || got.Repos["git@host:b.git"] {
		t.Fatalf("repos = %#v", got.Repos)
	}
}

func TestAccessCacheMissingIsEmpty(t *testing.T) {
	t.Parallel()

	got, err := LoadAccessCache(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Valid(time.Now(), DefaultAccessTTL) {
		t.Fatal("empty cache must be invalid")
	}
}
