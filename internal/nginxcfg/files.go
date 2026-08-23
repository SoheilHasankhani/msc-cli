package nginxcfg

import (
	"os"
	"path/filepath"

	"github.com/SoheilHasankhani/msc-cli/internal/state"
)

// ComponentsDir is the bind-mounted nginx server-block directory.
func ComponentsDir(configDir string) string {
	return filepath.Join(configDir, "nginx", "components")
}

// GeneratedDir is the CLI-owned nginx overlay directory.
func GeneratedDir(configDir string) string {
	return filepath.Join(configDir, "nginx", "generated")
}

// GeneratedFile is the docker/source upstream maps file.
func GeneratedFile(configDir string) string {
	return filepath.Join(GeneratedDir(configDir), "upstreams.conf")
}

// FilesLookup implements state.NginxLookup from the generated overlay.
type FilesLookup struct {
	Path  string
	Ports map[string]int
}

// Target reports the live nginx target for a compose service.
// A missing overlay or variable is treated as Docker Mode.
func (l FilesLookup) Target(composeService string) string {
	data, err := os.ReadFile(l.Path)
	if err != nil {
		return state.NginxTargetContainer
	}
	if t := ReadOverlayTarget(string(data), composeService, l.Ports[composeService]); t != "" {
		return t
	}
	return state.NginxTargetContainer
}
