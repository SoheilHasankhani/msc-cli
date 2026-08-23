package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
)

func TestBrandShimNeedsRefresh(t *testing.T) {
	root := t.TempDir()
	dirs := paths.Resolver{GOOS: "windows", Home: root}

	if !BrandShimNeedsRefresh("isos", dirs) {
		t.Fatal("expected refresh when cmd shim is missing")
	}

	if _, err := Write(dirs.ShimDir(), "isos", `C:\tools\msc.exe`, "windows"); err != nil {
		t.Fatal(err)
	}
	if BrandShimNeedsRefresh("isos", dirs) {
		t.Fatal("expected no refresh when cmd launcher is present")
	}
}

func TestEnsureWindowsEngineCommandWritesLauncher(t *testing.T) {
	root := t.TempDir()
	engineDir := filepath.Join(root, "repo", "bin")
	if err := os.MkdirAll(engineDir, 0o755); err != nil {
		t.Fatal(err)
	}
	engine := filepath.Join(engineDir, "msc.exe")
	if err := os.WriteFile(engine, []byte{'M', 'Z'}, 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "AppData", "Local", "msc")
	changed, err := EnsureWindowsEngineCommand(binDir, engine)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected first install to report a change")
	}
	exe := filepath.Join(binDir, "msc.exe")
	cmd := filepath.Join(binDir, "msc.cmd")
	if _, err := os.Lstat(exe); err == nil {
		if WindowsEngineLaunchPath(binDir, engine) != exe {
			t.Fatal("launch path should prefer linked msc.exe")
		}
		return
	}
	data, err := os.ReadFile(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), engine) || !strings.Contains(string(data), "set MSC_PROJECT=") {
		t.Fatalf("msc.cmd should launch the engine with MSC_PROJECT cleared:\n%s", data)
	}
	changed, err = EnsureWindowsEngineCommand(binDir, engine)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("second install should be a no-op")
	}
}
