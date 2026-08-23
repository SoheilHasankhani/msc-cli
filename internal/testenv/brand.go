package testenv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/registry"
)

// InstallBrandProject registers name under an isolated config home and writes a
// minimal meta-repo with a single compose service. Sets MSC_PROJECT when name
// is non-empty so brand-mode CLI tests can run without the developer registry.
func InstallBrandProject(t *testing.T, name string) {
	t.Helper()

	root := IsolateUserConfig(t)
	if name != "" {
		t.Setenv(project.EnvVar, name)
	}

	meta := filepath.Join(root, "meta")
	if err := os.MkdirAll(filepath.Join(meta, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Brand:       manifest.BrandInfo{DisplayName: name, Command: name},
		GitHost:     manifest.GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain: name + ".local",
		Repos: []manifest.RepoDef{{
			Name: name + "-api",
			Git:  "sos/" + name + "-api",
			Services: []manifest.ServiceDef{{
				ComposeService: "doctor",
				Path:           ".",
				SourcePort:     5010,
			}},
		}},
	}
	if err := m.Save(filepath.Join(meta, manifest.FileName)); err != nil {
		t.Fatal(err)
	}

	dirs := paths.Default()
	reg := registry.New()
	if _, err := reg.Register(name, registry.ProjectEntry{Path: meta}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Save(dirs.RegistryFile()); err != nil {
		t.Fatal(err)
	}
}
