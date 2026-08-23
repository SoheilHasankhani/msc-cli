//go:build windows

package shim

// RefreshWindowsShellHooks upserts PATH and PowerShell brand-command blocks into profile files.
func RefreshWindowsShellHooks(home, binDir string) ([]string, bool, error) {
	return ensureWindowsPathHook(home, binDir)
}
