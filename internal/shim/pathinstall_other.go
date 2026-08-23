//go:build !windows

package shim

import "path/filepath"

func windowsProfileTargets(home string) []string {
	return []string{
		filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
	}
}

func ensureWindowsUserPath(binDir string) error { return nil }
