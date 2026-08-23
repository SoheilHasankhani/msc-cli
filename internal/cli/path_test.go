package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/testenv"
)

func TestPathInstallCommandExists(t *testing.T) {
	t.Parallel()
	root := NewRootCmd()
	cmd, _, err := root.Find([]string{"path", "install"})
	if err != nil || cmd == nil || cmd.Name() != "install" {
		t.Fatalf("path install: %v %#v", err, cmd)
	}
}

func TestPathInstallWritesPlatformHook(t *testing.T) {
	home := testenv.IsolateUserConfig(t)
	t.Setenv("MSC_SKIP_USER_PATH", "1")
	docs := filepath.Join(home, "Documents")
	t.Setenv("MSC_WINDOWS_DOCUMENTS", docs)

	binDir := filepath.Join(home, "bin")
	cmd := newPathInstallCmd()
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.SetArgs([]string{"--dir", binDir})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	hook := pathHookFile(home, docs)
	data, err := os.ReadFile(hook)
	if err != nil {
		t.Fatalf("read %s: %v", hook, err)
	}
	text := string(data)
	if !strings.Contains(text, "# msc-begin path") || !strings.Contains(text, binDir) {
		t.Fatalf("%s missing PATH hook:\n%s", hook, text)
	}
}

func pathHookFile(home, docs string) string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(docs, "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	case "darwin":
		return filepath.Join(home, ".zshrc")
	default:
		return filepath.Join(home, ".bashrc")
	}
}

func TestResolveInstallBinDir(t *testing.T) {
	home := t.TempDir()
	dirs := paths.Resolver{Home: home, GOOS: "linux"}
	if got := resolveInstallBinDir("", dirs); got != dirs.BinDir() {
		t.Fatalf("default = %q, want %q", got, dirs.BinDir())
	}
	t.Setenv("MSC_INSTALL_DIR", filepath.Join(home, "opt", "bin"))
	wantEnv := filepath.Join(home, "opt", "bin")
	if got := resolveInstallBinDir("", dirs); got != wantEnv {
		t.Fatalf("MSC_INSTALL_DIR = %q, want %q", got, wantEnv)
	}
	wantFlag := filepath.Join(home, "custom")
	if got := resolveInstallBinDir(wantFlag, dirs); got != wantFlag {
		t.Fatalf("--dir = %q, want %q", got, wantFlag)
	}
}
