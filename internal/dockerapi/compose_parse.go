package dockerapi

import (
	"regexp"
	"strings"
)

var (
	ansiRE          = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	composeStatusRE = regexp.MustCompile(`(?i)^Container\s+(\S+)\s+(.+)$`)
	composePullRE   = regexp.MustCompile(`(?i)^Image\s+(\S+)\s+(Pulling|Pulled|Skipped|Interrupted|Error)\b(.*)$`)
)

// ComposeStatus is one container status line from docker compose output.
type ComposeStatus struct {
	Container string
	State     string
}

// ParseComposeStatusLine extracts a container status from a compose progress line.
func ParseComposeStatusLine(line string) (ComposeStatus, bool) {
	line = strings.TrimSpace(stripANSI(line))
	if line == "" {
		return ComposeStatus{}, false
	}
	m := composeStatusRE.FindStringSubmatch(line)
	if len(m) != 3 {
		return ComposeStatus{}, false
	}
	return ComposeStatus{Container: m[1], State: strings.TrimSpace(m[2])}, true
}

// ParseComposePullLine extracts an image pull status from compose pull output.
func ParseComposePullLine(line string) (ComposeStatus, bool) {
	line = strings.TrimSpace(stripANSI(line))
	if line == "" {
		return ComposeStatus{}, false
	}
	m := composePullRE.FindStringSubmatch(line)
	if len(m) < 3 {
		return ComposeStatus{}, false
	}
	state := strings.TrimSpace(m[2])
	if extra := strings.TrimSpace(m[3]); extra != "" && !strings.EqualFold(state, "Skipped") {
		state = state + " " + extra
	}
	return ComposeStatus{Container: m[1], State: state}, true
}

// DispatchComposeLine forwards compose stdout/stderr lines to onStatus when recognized.
func DispatchComposeLine(line string, onStatus StatusFn) {
	if onStatus == nil {
		return
	}
	if st, ok := ParseComposePullLine(line); ok {
		onStatus(st)
		return
	}
	if st, ok := ParseComposeStatusLine(line); ok {
		onStatus(st)
	}
}

// IsTerminalComposeStatus reports whether a compose state is a stable end state.
func IsTerminalComposeStatus(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "started", "running", "healthy", "exited", "recreated", "stopped", "removed",
		"pulled", "skipped", "interrupted", "error":
		return true
	default:
		return false
	}
}

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}
