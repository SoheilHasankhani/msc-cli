package dockerapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DefaultHost is the native Linux Engine socket.
const DefaultHost = "unix:///var/run/docker.sock"

type dockerConfigJSON struct {
	CurrentContext string `json:"currentContext"`
}

type dockerContextMeta struct {
	Name      string `json:"Name"`
	Endpoints struct {
		Docker struct {
			Host string `json:"Host"`
		} `json:"docker"`
	} `json:"Endpoints"`
}

// ResolveHost picks DOCKER_HOST, then the active Docker CLI context, then the Engine socket.
// Docker Desktop on Linux uses ~/.docker/desktop/docker.sock via the desktop-linux context,
// not /var/run/docker.sock.
func ResolveHost(getenv func(string) string, home string) string {
	if getenv == nil {
		getenv = os.Getenv
	}
	if h := strings.TrimSpace(getenv("DOCKER_HOST")); h != "" {
		return h
	}
	name := strings.TrimSpace(getenv("DOCKER_CONTEXT"))
	if name == "" {
		name = currentContextName(home)
	}
	if name != "" && name != "default" {
		if h := hostFromContext(home, name); h != "" {
			return h
		}
	}
	return platformDefaultHost
}

func currentContextName(home string) string {
	data, err := os.ReadFile(filepath.Join(home, ".docker", "config.json"))
	if err != nil {
		return ""
	}
	var cfg dockerConfigJSON
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.CurrentContext)
}

func hostFromContext(home, name string) string {
	matches, err := filepath.Glob(filepath.Join(home, ".docker", "contexts", "meta", "*", "meta.json"))
	if err != nil {
		return ""
	}
	for _, p := range matches {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var meta dockerContextMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		if meta.Name == name {
			return strings.TrimSpace(meta.Endpoints.Docker.Host)
		}
	}
	return ""
}
