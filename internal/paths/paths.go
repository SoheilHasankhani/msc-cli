// Package paths resolves OS-conventional directories for the msc engine.
package paths

import (
	"path/filepath"
	"runtime"
)

// AppName is the directory name used under the platform config root.
const AppName = "msc"

// Resolver computes engine paths from explicit environment values so tests
// never depend on the real machine.
type Resolver struct {
	GOOS       string
	Home       string
	ConfigHome string // XDG_CONFIG_HOME (Linux)
	AppData    string // APPDATA (Windows)
}

// Default returns a Resolver for the current process environment.
func Default() Resolver {
	return Resolver{
		GOOS:       runtime.GOOS,
		Home:       userHome(),
		ConfigHome: getenv("XDG_CONFIG_HOME"),
		AppData:    getenv("APPDATA"),
	}
}

// ConfigDir is the per-user config root for this engine
// (XDG / Application Support / %APPDATA%/msc).
func (r Resolver) ConfigDir() string {
	switch r.GOOS {
	case "windows":
		if r.AppData != "" {
			return filepath.Join(r.AppData, AppName)
		}
		return filepath.Join(r.Home, "AppData", "Roaming", AppName)
	case "darwin":
		return filepath.Join(r.Home, "Library", "Application Support", AppName)
	default:
		if r.ConfigHome != "" {
			return filepath.Join(r.ConfigHome, AppName)
		}
		return filepath.Join(r.Home, ".config", AppName)
	}
}

// RegistryFile is the local Project Registry YAML path.
func (r Resolver) RegistryFile() string {
	return filepath.Join(r.ConfigDir(), "projects.yml")
}

// StateDir holds the live-inference cache (never committed).
func (r Resolver) StateDir() string {
	return filepath.Join(r.ConfigDir(), "state")
}

// CertsDir holds the machine-level local CA (local-ca.crt / local-ca.key).
func (r Resolver) CertsDir() string {
	return filepath.Join(r.ConfigDir(), "certs")
}

// LogDir holds local structured JSON logs.
func (r Resolver) LogDir() string {
	return filepath.Join(r.ConfigDir(), "logs")
}

// ShimDir is where brand command shims are written (~/.msc/shims).
func (r Resolver) ShimDir() string {
	return filepath.Join(r.Home, ".msc", "shims")
}

// BinDir is where user-facing commands (msc and brand shims) are linked.
func (r Resolver) BinDir() string {
	switch r.GOOS {
	case "windows":
		if local := getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, AppName)
		}
		return filepath.Join(r.Home, "AppData", "Local", AppName)
	default:
		return filepath.Join(r.Home, ".local", "bin")
	}
}

// CompletionBash is the cached bash completion script installed for new shells.
func (r Resolver) CompletionBash() string {
	return filepath.Join(r.ConfigDir(), "completion.bash")
}

// CompletionZsh is the cached zsh completion script installed for new shells.
func (r Resolver) CompletionZsh() string {
	return filepath.Join(r.ConfigDir(), "completion.zsh")
}

// CompletionPowerShell is the cached PowerShell completion script.
func (r Resolver) CompletionPowerShell() string {
	return filepath.Join(r.ConfigDir(), "completion.ps1")
}
