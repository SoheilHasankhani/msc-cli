// Package testenv provides helpers for hermetic unit tests.
//
// Tests must not depend on the developer machine registry, shell profile, or
// real ~/.config/msc paths. Call IsolateUserConfig (or a fixture built on top
// of it) at the start of any test that touches paths.Default(), registry files,
// or brand-mode commands (MSC_PROJECT).
package testenv

import (
	"path/filepath"
	"testing"
)

// IsolateUserConfig redirects per-user config paths into a fresh temp directory
// for the duration of t. It sets the env vars read by internal/paths.Default().
// The temp root is returned for writing fixtures (manifest, registry, etc.).
func IsolateUserConfig(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	t.Setenv("APPDATA", filepath.Join(root, "AppData", "Roaming"))
	t.Setenv("USERPROFILE", root)
	t.Setenv("LOCALAPPDATA", filepath.Join(root, "AppData", "Local"))
	return root
}
