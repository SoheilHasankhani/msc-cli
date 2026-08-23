package state

import (
	"fmt"
	"time"
)

// Mode is how a service is currently routed.
type Mode string

const (
	ModeDocker Mode = "docker"
	ModeSource Mode = "source"
)

// ServiceState is the observed or cached status of one compose service.
type ServiceState struct {
	ComposeService string     `json:"compose_service"`
	Mode           Mode       `json:"mode"`
	ContainerUp    bool       `json:"container_up"`
	NginxTarget    string     `json:"nginx_target"`
	LastSwitchedAt *time.Time `json:"last_switched_at,omitempty"`
}

// NginxTargetContainer is the live-inference value when nginx points at the compose service.
const NginxTargetContainer = "container"

// NginxSourceTarget formats a Source Mode nginx target.
func NginxSourceTarget(port int) string {
	return fmt.Sprintf("source_port:%d", port)
}
