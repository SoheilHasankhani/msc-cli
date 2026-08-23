package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Snapshot is the on-disk cache of last-known service states.
type Snapshot struct {
	Services map[string]ServiceState `json:"services"`
}

// LoadCache reads a snapshot. A missing file is an empty snapshot.
func LoadCache(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Snapshot{Services: map[string]ServiceState{}}, nil
		}
		return nil, fmt.Errorf("read state cache %s: %w", path, err)
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state cache %s: %w", path, err)
	}
	if s.Services == nil {
		s.Services = map[string]ServiceState{}
	}
	return &s, nil
}

// SyncLive replaces cached observations with live state and keeps switch timestamps.
func (s *Snapshot) SyncLive(live map[string]ServiceState) *Snapshot {
	if s == nil {
		s = &Snapshot{}
	}
	next := &Snapshot{Services: make(map[string]ServiceState, len(live))}
	for name, lv := range live {
		if cached, ok := s.Services[name]; ok {
			lv.LastSwitchedAt = cached.LastSwitchedAt
		}
		next.Services[name] = lv
	}
	return next
}

// SaveCache writes the snapshot as JSON.
func SaveCache(path string, s *Snapshot) error {
	if s.Services == nil {
		s.Services = map[string]ServiceState{}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// CachePath is the per-project cache file under the engine state dir.
func CachePath(stateDir, project string) string {
	return filepath.Join(stateDir, project+".json")
}
