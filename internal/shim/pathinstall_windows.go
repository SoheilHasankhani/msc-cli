//go:build windows

package shim

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func windowsProfileTargets(home string) []string {
	docs := windowsDocumentsDir()
	if docs == "" {
		docs = filepath.Join(home, "Documents")
	}
	legacy := filepath.Join(home, "Documents")
	candidates := []string{
		filepath.Join(docs, "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(docs, "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(legacy, "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(legacy, "PowerShell", "Microsoft.PowerShell_profile.ps1"),
	}
	return uniquePaths(candidates)
}

func windowsDocumentsDir() string {
	if v := strings.TrimSpace(os.Getenv("MSC_WINDOWS_DOCUMENTS")); v != "" {
		return v
	}
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		"[Environment]::GetFolderPath('MyDocuments')").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func ensureWindowsUserPath(binDir string) error {
	if os.Getenv("MSC_SKIP_USER_PATH") != "" {
		return nil
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.READ|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open user Environment key: %w", err)
	}
	defer func() { _ = key.Close() }()

	path, _, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("read user Path: %w", err)
	}
	if pathContainsDir(path, binDir) {
		return nil
	}
	next := binDir
	if path != "" {
		next = binDir + ";" + path
	}
	if err := key.SetStringValue("Path", next); err != nil {
		return fmt.Errorf("write user Path: %w", err)
	}
	return nil
}

func pathContainsDir(path, dir string) bool {
	dir = strings.TrimRight(filepath.Clean(dir), `\`)
	for _, part := range strings.Split(path, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.EqualFold(filepath.Clean(part), dir) {
			return true
		}
	}
	return false
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = filepath.Clean(p)
		if p == "" || p == "." {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}
