package initsvc

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/registry"
)

type fakeGit struct {
	seed func(dest string) error
}

func (f fakeGit) LsRemote(context.Context, string) (bool, error) { return true, nil }
func (f fakeGit) Clone(_ context.Context, _, dest string, _ io.Writer) error {
	if f.seed != nil {
		return f.seed(dest)
	}
	return os.MkdirAll(dest, 0o755)
}
func (f fakeGit) Pull(context.Context, string, io.Writer) error { return nil }

func seedWithManifest(t *testing.T) func(string) error {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "manifests", "valid.yaml")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	return func(dest string) error {
		if err := os.MkdirAll(dest, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, manifest.FileName), data, 0o644)
	}
}

func testPathDirs(root string) paths.Resolver {
	return paths.Resolver{GOOS: "linux", Home: root}
}

func TestInitRegistersAndWritesShim(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dest := filepath.Join(root, "isos")
	regFile := filepath.Join(root, "projects.yml")
	shimDir := filepath.Join(root, "shims")

	res, err := Run(context.Background(), Options{
		RepoURL:      "git@gitlab.example.com:sos/meta.git",
		Path:         dest,
		Git:          fakeGit{seed: seedWithManifest(t)},
		RegistryFile: regFile,
		ShimDir:      shimDir,
		PathDirs:     testPathDirs(root),
		EnginePath:   "/opt/msc",
		GOOS:         "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Name != "isos" || res.WroteManifest {
		t.Fatalf("%#v", res)
	}
	wantCmd := filepath.Join(root, ".local", "bin", "isos")
	if res.CommandPath != wantCmd {
		t.Fatalf("CommandPath = %q, want %q", res.CommandPath, wantCmd)
	}
	reg, err := registry.Load(regFile)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := reg.Resolve("isos")
	if err != nil || entry.Path != dest {
		t.Fatalf("entry = %#v err=%v", entry, err)
	}
	if entry.GitRemote != "git@gitlab.example.com:sos/meta.git" {
		t.Fatalf("remote = %q", entry.GitRemote)
	}
	data, err := os.ReadFile(filepath.Join(shimDir, "isos"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "MSC_PROJECT=") {
		t.Fatalf("%s", data)
	}
}

func TestInitAsOverridesBlockedName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	regFile := filepath.Join(root, "projects.yml")
	reg := registry.New()
	_, _ = reg.Register("isos", registry.ProjectEntry{
		Path:       "/old",
		GitHostURL: "https://gitlab.other.com",
		GitRemote:  "git@gitlab.other.com:acme/meta.git",
	})
	if err := reg.Save(regFile); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(root, "isos2")
	res, err := Run(context.Background(), Options{
		RepoURL:      "git@gitlab.example.com:sos/meta.git",
		Path:         dest,
		As:           "isos-task",
		Git:          fakeGit{seed: seedWithManifest(t)},
		RegistryFile: regFile,
		ShimDir:      filepath.Join(root, "shims"),
		PathDirs:     testPathDirs(root),
		EnginePath:   "/opt/msc",
		GOOS:         "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Name != "isos-task" {
		t.Fatalf("name = %q", res.Name)
	}
}

func TestInitBlockedWithoutAs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	regFile := filepath.Join(root, "projects.yml")
	reg := registry.New()
	_, _ = reg.Register("isos", registry.ProjectEntry{
		Path:       "/old",
		GitHostURL: "https://gitlab.other.com",
		GitRemote:  "git@gitlab.other.com:acme/meta.git",
	})
	if err := reg.Save(regFile); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		RepoURL:      "git@gitlab.example.com:sos/meta.git",
		Path:         filepath.Join(root, "new"),
		Git:          fakeGit{seed: seedWithManifest(t)},
		RegistryFile: regFile,
		ShimDir:      filepath.Join(root, "shims"),
		PathDirs:     testPathDirs(root),
		EnginePath:   "/opt/msc",
		GOOS:         "linux",
	})
	if err == nil || !strings.Contains(err.Error(), "--as") {
		t.Fatalf("err = %v", err)
	}
}

func TestInitExistingProjectSkipsClone(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dest := filepath.Join(root, "isos")
	if err := seedWithManifest(t)(dest); err != nil {
		t.Fatal(err)
	}
	cloned := false
	_, err := Run(context.Background(), Options{
		RepoURL: "git@gitlab.example.com:sos/meta.git",
		Path:    dest,
		Git: fakeGit{seed: func(string) error {
			cloned = true
			return nil
		}},
		RegistryFile: filepath.Join(root, "projects.yml"),
		ShimDir:      filepath.Join(root, "shims"),
		PathDirs:     testPathDirs(root),
		EnginePath:   "/opt/msc",
		GOOS:         "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cloned {
		t.Fatal("should not clone when manifest already exists")
	}
}

func TestInitExistingTreeWithoutManifestDoesNotClone(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dest := filepath.Join(root, "existing-isos")
	if err := os.MkdirAll(filepath.Join(dest, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "local", "docker-compose.yml"), []byte("services:\n  doctor:\n    image: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "README.md"), []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cloned := false
	res, err := Run(context.Background(), Options{
		RepoURL: "git@gitlab.example.com:sos/meta.git",
		Path:    dest,
		Git: fakeGit{seed: func(string) error {
			cloned = true
			return nil
		}},
		RegistryFile: filepath.Join(root, "projects.yml"),
		ShimDir:      filepath.Join(root, "shims"),
		PathDirs:     testPathDirs(root),
		EnginePath:   "/opt/msc",
		GOOS:         "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cloned {
		t.Fatal("must not re-clone an existing checkout")
	}
	if !res.WroteManifest {
		t.Fatal("expected a drafted Manifest")
	}
	if _, err := os.Stat(filepath.Join(dest, manifest.FileName)); err != nil {
		t.Fatal(err)
	}
}

func TestInitWritesWizardManifestWhenMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dest := filepath.Join(root, "fresh")
	res, err := Run(context.Background(), Options{
		RepoURL: "git@gitlab.example.com:sos/meta.git",
		Path:    dest,
		Git: fakeGit{seed: func(d string) error {
			if err := os.MkdirAll(filepath.Join(d, "local"), 0o755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(d, "local", "docker-compose.yml"), []byte("services:\n  doctor:\n    image: x\n"), 0o644)
		}},
		RegistryFile: filepath.Join(root, "projects.yml"),
		ShimDir:      filepath.Join(root, "shims"),
		PathDirs:     testPathDirs(root),
		EnginePath:   "/opt/msc",
		GOOS:         "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.WroteManifest || res.CommitHint == "" {
		t.Fatalf("%#v", res)
	}
	if _, err := os.Stat(filepath.Join(dest, manifest.FileName)); err != nil {
		t.Fatal(err)
	}
}

func TestInitRequiresRepoWhenPathMissing(t *testing.T) {
	t.Parallel()
	if _, err := Run(context.Background(), Options{Path: filepath.Join(t.TempDir(), "new")}); err == nil {
		t.Fatal("repo required for new path")
	}
	if _, err := Run(context.Background(), Options{RepoURL: "git@x:y.git"}); err == nil {
		t.Fatal("path required")
	}
}

func TestInitResolvesRepoFromOrigin(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dest := filepath.Join(root, "meta")
	if err := seedWithManifest(t)(dest); err != nil {
		t.Fatal(err)
	}

	res, err := Run(context.Background(), Options{
		Path:         dest,
		OriginURL:    func(context.Context, string) (string, error) { return "git@gitlab.example.com:sos/meta.git", nil },
		RegistryFile: filepath.Join(root, "projects.yml"),
		ShimDir:      filepath.Join(root, "shims"),
		PathDirs:     testPathDirs(root),
		EnginePath:   "/opt/msc",
		GOOS:         "linux",
	})
	if err != nil {
		t.Fatal(err)
	}
	reg, err := registry.Load(filepath.Join(root, "projects.yml"))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := reg.Resolve(res.Name)
	if err != nil || entry.GitRemote != "git@gitlab.example.com:sos/meta.git" {
		t.Fatalf("entry = %#v err=%v", entry, err)
	}
}

func TestInitRequiresRepoWhenOriginUnknown(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dest := filepath.Join(root, "meta")
	if err := seedWithManifest(t)(dest); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		Path:         dest,
		OriginURL:    func(context.Context, string) (string, error) { return "", fmt.Errorf("no origin") },
		RegistryFile: filepath.Join(root, "projects.yml"),
		ShimDir:      filepath.Join(root, "shims"),
		PathDirs:     testPathDirs(root),
		GOOS:         "linux",
	})
	if err == nil || !strings.Contains(err.Error(), "--repo") {
		t.Fatalf("err = %v", err)
	}
}
