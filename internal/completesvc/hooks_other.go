//go:build !windows

package completesvc

import "path/filepath"

func windowsProfileTargets(home string) []string {
	return []string{
		filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
	}
}
