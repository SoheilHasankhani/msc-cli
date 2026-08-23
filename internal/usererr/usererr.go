// Package usererr turns raw Docker/git/nginx failures into actionable messages.
package usererr

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// MsgDockerDown is shown when the Engine socket/API is unreachable.
	MsgDockerDown = "Docker Engine is not running — start Docker Desktop or the docker service, then retry (or set DOCKER_HOST)"
	// MsgDockerPerm is shown when the socket exists but this user cannot use it.
	MsgDockerPerm = "cannot talk to Docker — add your user to the docker group or set DOCKER_HOST, then retry"
	// MsgPortBusy is shown when host ports (typically 80/443) are already taken.
	MsgPortBusy = "another project or process is already using this stack's host ports — run `down` on the other project first; only one stack can be up at a time"
	// MsgNginxReload is prepended when SIGHUP fails so the user knows routing did not change.
	MsgNginxReload = "nginx reload failed; previous routing is unchanged"
)

// Docker rewrites connection failures to MsgDockerDown.
func Docker(err error, detail string) error {
	if err == nil {
		return nil
	}
	if classified := classify(err, detail); classified != nil {
		return classified
	}
	return err
}

// Compose inspects docker compose stderr for a down daemon or a port clash.
func Compose(err error, stderr string) error {
	if err == nil {
		return nil
	}
	if classified := classify(err, stderr); classified != nil {
		return classified
	}
	if snippet := firstLine(stderr); snippet != "" {
		return fmt.Errorf("%w: %s", err, snippet)
	}
	return err
}

// NginxReload states that the previous nginx routing is still in place.
func NginxReload(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), MsgNginxReload) {
		return err
	}
	return fmt.Errorf("%s — %v", MsgNginxReload, err)
}

// IsDockerDown reports whether err is (or wraps) a down-daemon message.
func IsDockerDown(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Docker Engine is not running")
}

func classify(err error, detail string) error {
	text := strings.ToLower(err.Error() + " " + detail)
	switch {
	case isPortBusy(text):
		return errors.New(MsgPortBusy)
	case isDockerPerm(text):
		return errors.New(MsgDockerPerm)
	case isDockerDown(text):
		return errors.New(MsgDockerDown)
	default:
		return nil
	}
}

func isDockerDown(s string) bool {
	return strings.Contains(s, "cannot connect to the docker daemon") ||
		strings.Contains(s, "is the docker daemon running") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "no such file or directory") && strings.Contains(s, "docker") ||
		strings.Contains(s, "docker desktop is not running") ||
		strings.Contains(s, "error during connect")
}

func isDockerPerm(s string) bool {
	return strings.Contains(s, "permission denied") && strings.Contains(s, "docker")
}

func isPortBusy(s string) bool {
	return strings.Contains(s, "port is already allocated") ||
		strings.Contains(s, "address already in use") ||
		strings.Contains(s, "bind: address already in use") ||
		strings.Contains(s, "failed to bind host port")
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
