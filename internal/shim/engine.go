package shim

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
)

// ResolveEnginePath returns the msc engine binary for shim installation.
// It avoids go test binaries, brand re-exec names, and go run temp paths when possible.
func ResolveEnginePath() string {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return EngineOnPATH()
	}
	exe = normalizeEnginePath(exe)
	base := strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
	if strings.EqualFold(base, "msc") && !isTestBinary(exe) {
		return exe
	}
	for _, name := range []string{"msc", "msc.exe"} {
		if path, err := exec.LookPath(name); err == nil && path != "" && !isTestBinary(path) {
			return normalizeEnginePath(path)
		}
	}
	if !isTestBinary(exe) {
		return exe
	}
	return EngineOnPATH()
}

func isTestBinary(path string) bool {
	return strings.Contains(strings.ToLower(filepath.Base(path)), ".test")
}

// EnsureWindowsEngineCommand puts an msc command into binDir, like the Linux
// symlink from ~/.local/bin/msc to the repo build. It prefers a symlink to
// msc.exe and falls back to msc.cmd when Windows blocks symlinks.
func EnsureWindowsEngineCommand(binDir, enginePath string) (bool, error) {
	enginePath = normalizeEnginePath(enginePath)
	if binDir == "" || enginePath == "" || strings.EqualFold(enginePath, "msc") || isTestBinary(enginePath) {
		return false, nil
	}
	if _, err := os.Stat(enginePath); err != nil {
		return false, nil
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return false, err
	}
	destExe := filepath.Join(binDir, "msc.exe")
	destCmd := filepath.Join(binDir, "msc.cmd")
	if samePath(enginePath, destExe) {
		return false, nil
	}
	if destInfo, err := os.Lstat(destExe); err == nil {
		if destInfo.Mode()&os.ModeSymlink != 0 {
			if target, err := os.Readlink(destExe); err == nil && samePath(target, enginePath) {
				return false, nil
			}
		} else if sameFile(enginePath, destExe) {
			return false, nil
		}
		if err := os.Remove(destExe); err != nil {
			return writeWindowsEngineCmd(destCmd, enginePath)
		}
	}
	if err := os.Symlink(enginePath, destExe); err == nil {
		_ = os.Remove(destCmd)
		return true, nil
	}
	return writeWindowsEngineCmd(destCmd, enginePath)
}

func writeWindowsEngineCmd(dest, enginePath string) (bool, error) {
	body := "@echo off\r\nset MSC_PROJECT=\r\n" + quotePath(enginePath) + " %*\r\n"
	if existing, err := os.ReadFile(dest); err == nil && string(existing) == body {
		return false, nil
	}
	return true, os.WriteFile(dest, []byte(body), 0o755)
}

// WindowsEngineLaunchPath is the engine path brand .cmd files should exec.
func WindowsEngineLaunchPath(binDir, enginePath string) string {
	exe := filepath.Join(binDir, "msc.exe")
	if _, err := os.Stat(exe); err == nil {
		return exe
	}
	return normalizeEnginePath(enginePath)
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	return strings.EqualFold(a, b)
}

func sameFile(a, b string) bool {
	ai, err1 := os.Stat(a)
	bi, err2 := os.Stat(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return os.SameFile(ai, bi)
}

func normalizeEnginePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if ext := filepath.Ext(path); ext != "" {
		return path
	}
	if _, err := os.Stat(path + ".exe"); err == nil {
		return path + ".exe"
	}
	return path
}

// BrandShimNeedsRefresh reports whether the brand command should be rewritten.
func BrandShimNeedsRefresh(brand string, dirs paths.Resolver) bool {
	if brand == "" {
		return false
	}
	return !ValidBrandShim(dirs.ShimDir(), brand, dirs.GOOS)
}
