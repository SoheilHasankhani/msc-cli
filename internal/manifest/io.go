package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads and unmarshals a Manifest from path. It does not Validate.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	m.ApplyDefaults()
	return &m, nil
}

// Save writes m as YAML, creating parent directories as needed.
func (m *Manifest) Save(path string) error {
	m.ApplyDefaults()
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest %s: %w", path, err)
	}
	return nil
}

// Find returns the canonical manifest path under root, or an error if absent.
func Find(root string) (string, error) {
	path := filepath.Join(root, FileName)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("manifest %s not found in %s", FileName, root)
	}
	if info.IsDir() {
		return "", fmt.Errorf("manifest path %s is a directory", path)
	}
	return path, nil
}
