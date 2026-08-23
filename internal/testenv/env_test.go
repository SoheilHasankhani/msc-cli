package testenv

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
)

func TestIsolateUserConfigRedirectsRegistryFile(t *testing.T) {
	root := IsolateUserConfig(t)
	got := paths.Default().RegistryFile()

	rel, err := filepath.Rel(root, got)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.Fatalf("RegistryFile() = %q is not under isolated root %q", got, root)
	}
	if filepath.Base(got) != "projects.yml" {
		t.Fatalf("RegistryFile() = %q, want basename projects.yml", got)
	}
	if filepath.Base(filepath.Dir(got)) != paths.AppName {
		t.Fatalf("RegistryFile() parent = %q, want %q", filepath.Base(filepath.Dir(got)), paths.AppName)
	}
}
