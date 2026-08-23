package project

import (
	"fmt"
	"path/filepath"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/registry"
)

// Context is a resolved project: registry entry + validated manifest.
type Context struct {
	Name     string
	Root     string
	Entry    registry.ProjectEntry
	Manifest *manifest.Manifest
}

// Resolve loads the local registry and the project's Manifest.
func Resolve(name string, dirs paths.Resolver) (*Context, error) {
	if name == "" {
		return nil, fmt.Errorf("%s", MissingMessage())
	}
	reg, err := registry.Load(dirs.RegistryFile())
	if err != nil {
		return nil, err
	}
	entry, err := reg.Resolve(name)
	if err != nil {
		return nil, err
	}
	status := registry.CheckPath(entry.Path)
	if status != registry.PathOK {
		return nil, fmt.Errorf("registered path for %q is not a valid project (%v): %s — run the command again after fixing the path or use msc projects relink", name, status, entry.Path)
	}
	manPath, err := manifest.Find(entry.Path)
	if err != nil {
		return nil, err
	}
	m, err := manifest.Load(manPath)
	if err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("manifest %s: %w", manPath, err)
	}
	return &Context{Name: name, Root: entry.Path, Entry: entry, Manifest: m}, nil
}

// ComposeFile is the absolute compose file path.
func (c *Context) ComposeFile() string {
	return filepath.Join(c.Root, c.Manifest.Layout.ComposeFile)
}

// ConfigDir is the absolute stack config directory.
func (c *Context) ConfigDir() string {
	return filepath.Join(c.Root, c.Manifest.Layout.ConfigDir)
}

// ClonesDir is the absolute clones workspace.
func (c *Context) ClonesDir() string {
	return filepath.Join(c.Root, c.Manifest.Layout.ClonesDir)
}
