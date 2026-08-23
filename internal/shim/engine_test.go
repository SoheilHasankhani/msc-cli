package shim

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveEnginePathSkipsTestBinary(t *testing.T) {
	dir := t.TempDir()
	engineName := "msc"
	if runtime.GOOS == "windows" {
		engineName = "msc.exe"
	}
	testBin := filepath.Join(dir, "doctor.test")
	if runtime.GOOS == "windows" {
		testBin += ".exe"
	}
	if err := os.WriteFile(testBin, []byte{'M', 'Z'}, 0o644); err != nil {
		t.Fatal(err)
	}
	realEngine := filepath.Join(dir, engineName)
	engineMode := os.FileMode(0o644)
	if runtime.GOOS != "windows" {
		engineMode = 0o755
	}
	if err := os.WriteFile(realEngine, []byte{'M', 'Z'}, engineMode); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	// Simulate running from a go test binary path.
	got := ResolveEnginePath()
	if isTestBinary(got) {
		t.Fatalf("ResolveEnginePath() = %q, want non-test engine", got)
	}
	if filepath.Base(got) != engineName {
		t.Fatalf("ResolveEnginePath() = %q, want %q on PATH", got, engineName)
	}
}

func TestIsTestBinary(t *testing.T) {
	t.Parallel()
	if !isTestBinary(`C:\tmp\doctor.test.exe`) {
		t.Fatal("expected doctor.test.exe to be a test binary")
	}
	if isTestBinary(`C:\tmp\msc.exe`) {
		t.Fatal("msc.exe is not a test binary")
	}
}
