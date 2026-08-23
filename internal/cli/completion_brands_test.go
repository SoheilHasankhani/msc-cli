package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/completesvc"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/registry"
	"github.com/SoheilHasankhani/msc-cli/internal/testenv"
)

func TestWriteZshBrandCompleters(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	if err := completesvc.WriteZshBrandCompleters(&out, []string{"myproject"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, `compdef _msc "myproject"`) {
		t.Fatalf("missing zsh compdef:\n%s", got)
	}
}

func TestBashCompletionOutputRegistersRegistryBrands(t *testing.T) {
	root := testenv.IsolateUserConfig(t)

	dirs := paths.Default()
	reg := registry.New()
	if _, err := reg.Register("myproject", registry.ProjectEntry{Path: filepath.Join(root, "meta")}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Save(dirs.RegistryFile()); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"completion", "bash"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, `complete -o default -F __start_msc "myproject"`) {
		t.Fatalf("expected brand shim registration in bash completion:\n%s", got)
	}
}

func TestCompletionInstallCommandExists(t *testing.T) {
	cmd, _, err := NewRootCmd().Find([]string{"completion", "install"})
	if err != nil || cmd == nil {
		t.Fatalf("completion install: %v", err)
	}
}
