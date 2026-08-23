package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/registry"
)

func TestResolveRequiresName(t *testing.T) {
	t.Parallel()
	if _, err := Resolve("", paths.Resolver{Home: t.TempDir()}); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveUnknownProject(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	dirs := paths.Resolver{GOOS: "linux", Home: home, ConfigHome: filepath.Join(home, ".config")}
	if err := os.MkdirAll(dirs.ConfigDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve("isos", dirs); err == nil {
		t.Fatal("expected not registered")
	}
}

func TestResolveValidProject(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	root := t.TempDir()
	m := &manifest.Manifest{
		Brand:       manifest.BrandInfo{DisplayName: "isos", Command: "isos"},
		GitHost:     manifest.GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain: "isos.local",
	}
	if err := m.Save(filepath.Join(root, manifest.FileName)); err != nil {
		t.Fatal(err)
	}

	dirs := paths.Resolver{GOOS: "linux", Home: home, ConfigHome: filepath.Join(home, ".config")}
	reg := registry.New()
	if _, err := reg.Register("isos", registry.ProjectEntry{
		Path:       root,
		GitHostURL: "https://gitlab.example.com",
		GitRemote:  "git@gitlab.example.com:sos/meta.git",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dirs.RegistryFile()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := reg.Save(dirs.RegistryFile()); err != nil {
		t.Fatal(err)
	}

	ctx, err := Resolve("isos", dirs)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Root != root || ctx.Manifest.Brand.Command != "isos" {
		t.Fatalf("%#v", ctx)
	}
	if ctx.ComposeFile() != filepath.Join(root, manifest.DefaultComposeFile) {
		t.Fatalf("compose = %q", ctx.ComposeFile())
	}
}

func TestResolveRecommendedLayoutPaths(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	root := t.TempDir()
	m := &manifest.Manifest{
		Brand:       manifest.BrandInfo{DisplayName: "isos", Command: "isos"},
		GitHost:     manifest.GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain: "isos.local",
		Layout: manifest.Layout{
			ComposeFile:    manifest.RecommendedComposeFile,
			ComposeProfile: "standard",
			ConfigDir:      manifest.RecommendedConfigDir,
			ClonesDir:      manifest.DefaultClonesDir,
		},
	}
	if err := m.Save(filepath.Join(root, manifest.FileName)); err != nil {
		t.Fatal(err)
	}

	dirs := paths.Resolver{GOOS: "linux", Home: home, ConfigHome: filepath.Join(home, ".config")}
	reg := registry.New()
	if _, err := reg.Register("isos", registry.ProjectEntry{
		Path:       root,
		GitHostURL: "https://gitlab.example.com",
		GitRemote:  "git@gitlab.example.com:sos/meta.git",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dirs.RegistryFile()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := reg.Save(dirs.RegistryFile()); err != nil {
		t.Fatal(err)
	}

	ctx, err := Resolve("isos", dirs)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.ComposeFile() != filepath.Join(root, "compose", "docker-compose.yml") {
		t.Fatalf("compose = %q", ctx.ComposeFile())
	}
	if ctx.ConfigDir() != filepath.Join(root, "config") {
		t.Fatalf("config = %q", ctx.ConfigDir())
	}
	if ctx.ClonesDir() != filepath.Join(root, "local") {
		t.Fatalf("clones = %q", ctx.ClonesDir())
	}
}
