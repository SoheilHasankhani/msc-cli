package initsvc

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/registry"
)

func TestTwoProjectsResolveIndependently(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	shimDir := filepath.Join(root, "shims")
	dirs := paths.Resolver{GOOS: "linux", Home: root, ConfigHome: filepath.Join(root, "config")}
	regFile := dirs.RegistryFile()

	a := filepath.Join(root, "isos")
	b := filepath.Join(root, "mores")
	if _, err := Run(context.Background(), Options{
		RepoURL: "git@gitlab.example.com:sos/meta.git", Path: a,
		Git: fakeGit{seed: seedWithManifest(t)}, RegistryFile: regFile,
		ShimDir: shimDir, PathDirs: dirs, EnginePath: "/opt/msc", GOOS: "linux",
	}); err != nil {
		t.Fatal(err)
	}

	// Second project: change brand in a copy so command is mores
	if _, err := Run(context.Background(), Options{
		RepoURL: "git@gitlab.other.com:acme/meta.git", Path: b, As: "mores",
		Git: fakeGit{seed: seedWithManifest(t)}, RegistryFile: regFile,
		ShimDir: shimDir, PathDirs: dirs, EnginePath: "/opt/msc", GOOS: "linux",
	}); err != nil {
		t.Fatal(err)
	}

	pa, err := project.Resolve("isos", dirs)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := project.Resolve("mores", dirs)
	if err != nil {
		t.Fatal(err)
	}
	if pa.Root == pb.Root {
		t.Fatal("projects must not share a path")
	}
	if pa.Name != "isos" || pb.Name != "mores" {
		t.Fatalf("%q %q", pa.Name, pb.Name)
	}

	reg, err := registry.Load(regFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Projects) != 2 {
		t.Fatalf("projects = %d", len(reg.Projects))
	}
}
