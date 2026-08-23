package gitops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultAccessTTL is how long ls-remote results are reused without --refresh.
const DefaultAccessTTL = 7 * 24 * time.Hour

// AccessCache stores per-repo ls-remote results.
type AccessCache struct {
	CheckedAt time.Time       `json:"checked_at"`
	Repos     map[string]bool `json:"repos"`
}

// AccessCachePath is the per-project access cache under the engine state dir.
func AccessCachePath(stateDir, project string) string {
	return filepath.Join(stateDir, project+"-access.json")
}

// LoadAccessCache reads a cache. A missing file is an empty, invalid cache.
func LoadAccessCache(path string) (*AccessCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AccessCache{Repos: map[string]bool{}}, nil
		}
		return nil, fmt.Errorf("read access cache %s: %w", path, err)
	}
	var c AccessCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse access cache %s: %w", path, err)
	}
	if c.Repos == nil {
		c.Repos = map[string]bool{}
	}
	return &c, nil
}

// SaveAccessCache writes the cache as JSON.
func SaveAccessCache(path string, c *AccessCache) error {
	if c.Repos == nil {
		c.Repos = map[string]bool{}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Valid reports whether the cache is fresh at now for the given TTL.
func (c *AccessCache) Valid(now time.Time, ttl time.Duration) bool {
	if c == nil || c.CheckedAt.IsZero() || len(c.Repos) == 0 {
		return false
	}
	return now.Sub(c.CheckedAt) < ttl
}
