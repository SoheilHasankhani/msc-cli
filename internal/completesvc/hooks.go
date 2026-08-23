package completesvc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	hookBegin = "# msc-begin shell-completion"
	hookEnd   = "# msc-end shell-completion"
)

func hookBlock(bashScript, zshScript string) string {
	bashScript = filepath.ToSlash(bashScript)
	zshScript = filepath.ToSlash(zshScript)
	return fmt.Sprintf(`%s
# Managed by msc — regenerate with: msc completion install
if [ -n "${BASH_VERSION:-}" ] && [ -f %q ]; then
  . %q
elif [ -n "${ZSH_VERSION:-}" ] && [ -f %q ]; then
  . %q
fi
%s
`, hookBegin, bashScript, bashScript, zshScript, zshScript, hookEnd)
}

func rcCandidates(home, goos string) []string {
	if goos == "darwin" {
		return []string{
			filepath.Join(home, ".zshrc"),
			filepath.Join(home, ".bash_profile"),
			filepath.Join(home, ".bashrc"),
		}
	}
	return []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
	}
}

func defaultRCCreate(home, goos string) string {
	if goos == "darwin" {
		return filepath.Join(home, ".zshrc")
	}
	return filepath.Join(home, ".bashrc")
}

func powershellHookBlock(script string) string {
	escaped := strings.ReplaceAll(script, `'`, `''`)
	return fmt.Sprintf(`%s
# Managed by msc — regenerate with: msc completion install
$__mscCompletion = '%s'
if (Test-Path -LiteralPath $__mscCompletion) { . $__mscCompletion }
%s
`, hookBegin, escaped, hookEnd)
}

// EnsureShellHooks upserts the msc completion source block into shell startup files.
func EnsureShellHooks(home, goos, bashScript, zshScript, psScript string) ([]string, error) {
	if goos == "windows" {
		return ensureWindowsCompletionHooks(home, bashScript, zshScript, psScript)
	}
	return ensureUnixCompletionHooks(home, goos, bashScript, zshScript)
}

func ensureUnixCompletionHooks(home, goos, bashScript, zshScript string) ([]string, error) {
	block := hookBlock(bashScript, zshScript)
	var targets []string
	for _, rc := range rcCandidates(home, goos) {
		if _, err := os.Stat(rc); err == nil {
			targets = append(targets, rc)
		}
	}
	if len(targets) == 0 {
		targets = []string{defaultRCCreate(home, goos)}
	}
	var updated []string
	for _, rc := range targets {
		if err := upsertHookFile(rc, block); err != nil {
			return nil, err
		}
		updated = append(updated, rc)
	}
	return updated, nil
}

func ensureWindowsCompletionHooks(home, bashScript, zshScript, psScript string) ([]string, error) {
	var updated []string
	if psScript != "" {
		block := powershellHookBlock(psScript)
		for _, profile := range windowsProfileTargets(home) {
			if err := upsertHookFile(profile, block); err != nil {
				return nil, err
			}
			updated = append(updated, profile)
		}
	}
	unix, err := ensureExistingUnixHooks(home, bashScript, zshScript)
	if err != nil {
		return updated, err
	}
	return append(updated, unix...), nil
}

func ensureExistingUnixHooks(home, bashScript, zshScript string) ([]string, error) {
	if bashScript == "" || zshScript == "" {
		return nil, nil
	}
	block := hookBlock(bashScript, zshScript)
	var updated []string
	for _, rc := range []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
	} {
		if _, err := os.Stat(rc); err != nil {
			continue
		}
		if err := upsertHookFile(rc, block); err != nil {
			return updated, err
		}
		updated = append(updated, rc)
	}
	return updated, nil
}

func upsertHookFile(path, block string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	content := string(data)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		content = ""
	}
	next := upsertBlock(content, block)
	if next == content {
		return nil
	}
	return os.WriteFile(path, []byte(next), 0o644)
}

func upsertBlock(content, block string) string {
	start := strings.Index(content, hookBegin)
	if start < 0 {
		body := strings.TrimRight(content, "\n")
		if body != "" {
			body += "\n\n"
		}
		return body + block + "\n"
	}
	rest := content[start:]
	relEnd := strings.Index(rest, hookEnd)
	if relEnd < 0 {
		return content[:start] + block + "\n"
	}
	stop := start + relEnd + len(hookEnd)
	for stop < len(content) && (content[stop] == '\n' || content[stop] == '\r') {
		stop++
	}
	return content[:start] + block + "\n" + strings.TrimLeft(content[stop:], "\n")
}
