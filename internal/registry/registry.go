// Package registry owns the local, never-committed Project Registry.
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Registry maps a brand command name to exactly one meta-repo path.
type Registry struct {
	Projects map[string]ProjectEntry `yaml:"projects"`
}

// ProjectEntry is one registered project on this machine.
type ProjectEntry struct {
	Path       string    `yaml:"path"`
	GitHostURL string    `yaml:"git_host_url"`
	GitRemote  string    `yaml:"git_remote"`
	LastUsed   time.Time `yaml:"last_used"`
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{Projects: map[string]ProjectEntry{}}
}

// Load reads a registry from path. A missing file is an empty registry.
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil
		}
		return nil, fmt.Errorf("read registry %s: %w", path, err)
	}
	var r Registry
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse registry %s: %w", path, err)
	}
	if r.Projects == nil {
		r.Projects = map[string]ProjectEntry{}
	}
	return &r, nil
}

// Save writes the registry as YAML, creating parent directories as needed.
func (r *Registry) Save(path string) error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return fmt.Errorf("encode registry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write registry %s: %w", path, err)
	}
	return nil
}

// Resolve returns the entry for name.
func (r *Registry) Resolve(name string) (ProjectEntry, error) {
	entry, ok := r.Projects[name]
	if !ok {
		return ProjectEntry{}, fmt.Errorf("project %q is not registered; run msc init --repo <url>", name)
	}
	return entry, nil
}

// Relink updates the path of an existing entry after CheckPath on newPath is PathOK.
func (r *Registry) Relink(name, newPath string) error {
	if _, ok := r.Projects[name]; !ok {
		return fmt.Errorf("project %q is not registered", name)
	}
	if status := CheckPath(newPath); status != PathOK {
		return fmt.Errorf("cannot relink %q to %s (%v); path must be a directory containing %s", name, newPath, status, "msc.manifest.yml")
	}
	entry := r.Projects[name]
	entry.Path = newPath
	r.Projects[name] = entry
	return nil
}

// Names returns registered project names in map iteration order (caller should sort).
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.Projects))
	for name := range r.Projects {
		out = append(out, name)
	}
	return out
}

// Remove deletes the entry for name.
func (r *Registry) Remove(name string) error {
	if _, ok := r.Projects[name]; !ok {
		return fmt.Errorf("project %q is not registered", name)
	}
	delete(r.Projects, name)
	return nil
}
