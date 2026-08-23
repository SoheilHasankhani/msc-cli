package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
)

func TestUnixScriptSetsProjectEnv(t *testing.T) {
	t.Parallel()

	got := UnixScript("/opt/msc", "isos")
	if !strings.HasPrefix(got, "#!/bin/sh\n") {
		t.Fatal("missing shebang")
	}
	want := fmt.Sprintf(`exec env %s="isos" "/opt/msc" "$@"`, project.EnvVar)
	if !strings.Contains(got, want) {
		t.Fatalf("unix shim:\n%s", got)
	}
	if strings.Contains(got, "--project") {
		t.Fatalf("shim must not pass --project:\n%s", got)
	}
}

func TestCmdScriptSetsProjectEnv(t *testing.T) {
	t.Parallel()

	got := CmdScript(`C:\tools\msc.exe`, "isos")
	if !strings.Contains(got, `@echo off`) {
		t.Fatal("missing echo off")
	}
	if !strings.Contains(got, `set MSC_PROJECT=isos`) || !strings.Contains(got, `"C:\tools\msc.exe" %*`) {
		t.Fatalf("cmd shim:\n%s", got)
	}
}

func TestValidUnix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "isos")
	if err := os.WriteFile(path, []byte(UnixScript("/opt/msc", "isos")), 0o755); err != nil {
		t.Fatal(err)
	}
	if !ValidUnix(path) {
		t.Fatal("expected valid shim")
	}
	if err := os.WriteFile(path, []byte{0x7f, 'E', 'L', 'F'}, 0o755); err != nil {
		t.Fatal(err)
	}
	if ValidUnix(path) {
		t.Fatal("binary should not be valid shim")
	}
}

func TestWriteUnixShim(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path, err := Write(dir, "isos", "/opt/msc", "linux")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "isos" {
		t.Fatalf("path = %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows host filesystems do not represent Unix execute bits.
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		t.Fatalf("shim is not executable: %v", info.Mode())
	}
	data, _ := os.ReadFile(path)
	text := string(data)
	if !strings.Contains(text, "MSC_PROJECT=") || !strings.Contains(text, `"/opt/msc"`) {
		t.Fatalf("%s", data)
	}
}

func TestWriteWindowsShims(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path, err := Write(dir, "isos", `C:\tools\msc.exe`, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "isos.cmd" {
		t.Fatalf("primary windows shim = %s", path)
	}
	if !ValidCmd(path) {
		t.Fatal("expected cmd launcher")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `set MSC_PROJECT=isos`) || !strings.Contains(string(data), `"C:\tools\msc.exe" %*`) {
		t.Fatalf("cmd shim:\n%s", data)
	}
}

func TestValidCmdBrandShim(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "isos.cmd")
	if err := os.WriteFile(path, []byte(CmdScript(`C:\tools\msc.exe`, "isos")), 0o644); err != nil {
		t.Fatal(err)
	}
	if !ValidCmd(path) {
		t.Fatal("expected valid cmd shim")
	}
	if err := os.WriteFile(path, []byte{'M', 'Z'}, 0o644); err != nil {
		t.Fatal(err)
	}
	if ValidCmd(path) {
		t.Fatal("PE binary must not count as cmd shim")
	}
	if err := os.WriteFile(path, []byte(CmdScript(`C:\tools\msc.exe`, "isos")), 0o644); err != nil {
		t.Fatal(err)
	}
	if !ValidBrandShim(dir, "isos", "windows") {
		t.Fatal("expected valid brand shim on windows")
	}
	if ValidBrandShim(dir, "isos", "linux") {
		t.Fatal("linux should not accept .cmd shim")
	}
}

func TestInstallOnPATHWindows(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "Documents")
	t.Setenv("MSC_WINDOWS_DOCUMENTS", docs)
	t.Setenv("MSC_SKIP_USER_PATH", "1")
	shimDir := filepath.Join(root, ".msc", "shims")
	shimPath, err := Write(shimDir, "myproject", `C:\tools\msc.exe`, "windows")
	if err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(root, "AppData", "Local")
	t.Setenv("LOCALAPPDATA", local)
	res, err := InstallOnPATH("myproject", shimPath, paths.Resolver{GOOS: "windows", Home: root})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(local, "msc", "myproject.cmd")
	if res.CommandPath != want {
		t.Fatalf("CommandPath = %q, want %q", res.CommandPath, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(local, "msc", "myproject.exe")); err == nil {
		t.Fatal("legacy brand .exe should not be installed")
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "set MSC_PROJECT=myproject") {
		t.Fatalf("installed cmd:\n%s", data)
	}
	if len(res.ShellFiles) != 2 {
		t.Fatalf("ShellFiles = %v, want PS5 + PS7 under test Documents", res.ShellFiles)
	}
	for _, profile := range res.ShellFiles {
		data, err := os.ReadFile(profile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), pathHookBegin) {
			t.Fatalf("profile %s missing path hook", profile)
		}
		if !strings.Contains(string(data), brandHookBegin) {
			t.Fatalf("profile %s missing brand-commands hook", profile)
		}
		text := string(data)
		if !strings.Contains(text, "Filter '*.cmd'") || !strings.Contains(text, ".cmd") {
			t.Fatalf("profile %s should invoke brand .cmd launchers:\n%s", profile, text)
		}
		if strings.Contains(text, "Filter '*.exe'") || strings.Contains(text, "Get-Command msc.exe") {
			t.Fatalf("profile %s still looks up a copied engine:\n%s", profile, text)
		}
	}
}

func TestInstallOnPATHLinksBrandCommand(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	shimDir := filepath.Join(root, ".msc", "shims")
	shimPath, err := Write(shimDir, "myproject", "/opt/msc", "linux")
	if err != nil {
		t.Fatal(err)
	}
	res, err := InstallOnPATH("myproject", shimPath, paths.Resolver{GOOS: "linux", Home: root})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".local", "bin", "myproject")
	if res.CommandPath != want {
		t.Fatalf("CommandPath = %q, want %q", res.CommandPath, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
}
