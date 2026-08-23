package hostcerts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	OpWriteHosts = "write-hosts"
	OpInstallCA  = "install-ca"
)

// Payload is the only input the hidden elevated command accepts.
// HostsPath and DestPath are for tests and for the non-elevated caller that
// already resolved destinations; the elevated process still validates them.
type Payload struct {
	Op        string   `json:"op"`
	Project   string   `json:"project"`
	Names     []string `json:"names,omitempty"`
	CAPath    string   `json:"ca_path,omitempty"`
	HostsPath string   `json:"hosts_path,omitempty"`
	DestPath  string   `json:"dest_path,omitempty"`
}

// WritePayloadFile writes a 0600 JSON payload.
func WritePayloadFile(path string, p Payload) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// ReadPayloadFile loads a payload from disk.
func ReadPayloadFile(path string) (Payload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Payload{}, err
	}
	var p Payload
	if err := json.Unmarshal(data, &p); err != nil {
		return Payload{}, err
	}
	return p, nil
}

// ApplyPayload performs one privileged (or test) operation. It never deletes files
// outside the explicit destination and never runs a shell.
func ApplyPayload(p Payload) error {
	switch p.Op {
	case OpWriteHosts:
		if strings.TrimSpace(p.Project) == "" {
			return fmt.Errorf("write-hosts: project is required")
		}
		if p.HostsPath == "" {
			return fmt.Errorf("write-hosts: hosts path is required")
		}
		if err := guardPath(p.HostsPath); err != nil {
			return err
		}
		return WriteFile(p.HostsPath, p.Project, p.Names)
	case OpInstallCA:
		if strings.TrimSpace(p.Project) == "" {
			return fmt.Errorf("install-ca: project is required")
		}
		if p.CAPath == "" || p.DestPath == "" {
			return fmt.Errorf("install-ca: ca_path and dest_path are required")
		}
		if err := guardPath(p.CAPath); err != nil {
			return err
		}
		if err := guardPath(p.DestPath); err != nil {
			return err
		}
		data, err := os.ReadFile(p.CAPath)
		if err != nil {
			return err
		}
		if _, err := ParseCertificatePEM(data); err != nil {
			return fmt.Errorf("install-ca: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(p.DestPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p.DestPath, data, 0o644)
	default:
		return fmt.Errorf("unknown elevated operation %q", p.Op)
	}
}

func guardPath(path string) error {
	if path == "" || strings.Contains(path, "\x00") {
		return fmt.Errorf("invalid path")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}
	return nil
}
