package progress

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/gitops"
)

var httpPostErrRE = regexp.MustCompile(`(?s)^Post "http[^"]*":\s*`)

// ShortImageRef returns a compact label for progress output.
func ShortImageRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ref
	}
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		ref = ref[i+1:]
	}
	return Truncate(ref, 32)
}

// Truncate shortens s to at most max runes with an ellipsis.
func Truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// FormatDockerPullError returns a single-line message for image pull failures.
func FormatDockerPullError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	msg = httpPostErrRE.ReplaceAllString(msg, "")
	msg = strings.TrimPrefix(msg, "docker pull: ")
	if extracted := extractDockerMessage(msg); extracted != "" {
		msg = extracted
	}
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "pull access denied"),
		strings.Contains(lower, "authorization failed"),
		strings.Contains(lower, "no basic auth credentials"):
		return "pull access denied — run: docker login <registry>"
	case strings.Contains(lower, "failed to resolve reference"):
		return Truncate(msg, 100)
	}
	return Truncate(firstProgressLine(msg), 100)
}

func extractDockerMessage(s string) string {
	i := strings.Index(s, "{")
	if i < 0 {
		return ""
	}
	var v struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(s[i:]), &v); err != nil || v.Message == "" {
		return ""
	}
	return strings.TrimSpace(v.Message)
}

func firstProgressLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return text
}

func shortProgressError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.Contains(msg, "docker pull") ||
		strings.Contains(msg, `Post "http://docker`) ||
		strings.Contains(msg, `"message"`) {
		return FormatDockerPullError(err)
	}
	return gitops.FormatPullError(err)
}
