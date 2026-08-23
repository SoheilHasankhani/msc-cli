//go:build windows

package completesvc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func windowsProfileTargets(home string) []string {
	docs := windowsDocumentsDir()
	if docs == "" {
		docs = filepath.Join(home, "Documents")
	}
	legacy := filepath.Join(home, "Documents")
	return uniquePaths([]string{
		filepath.Join(docs, "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(docs, "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(legacy, "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(legacy, "PowerShell", "Microsoft.PowerShell_profile.ps1"),
	})
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
