//go:build !windows

package shim

func RefreshWindowsShellHooks(home, binDir string) ([]string, bool, error) {
	return nil, false, nil
}
