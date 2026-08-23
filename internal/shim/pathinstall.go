package shim

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
)

const (
	pathHookBegin  = "# msc-begin path"
	pathHookEnd    = "# msc-end path"
	brandHookBegin = "# msc-begin brand-commands"
	brandHookEnd   = "# msc-end brand-commands"
)

// InstallResult describes a brand command linked onto the user's PATH.
type InstallResult struct {
	CommandPath string
	ShellFiles  []string
}

// InstallOnPATH links the brand shim into the conventional bin directory and
// upserts a managed PATH block into shell startup files when needed.
func InstallOnPATH(projectName, shimPath string, dirs paths.Resolver) (InstallResult, error) {
	if projectName == "" {
		return InstallResult{}, fmt.Errorf("project name is required")
	}
	if shimPath == "" {
		return InstallResult{}, fmt.Errorf("shim path is required")
	}
	binDir := dirs.BinDir()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return InstallResult{}, err
	}

	dest := filepath.Join(binDir, projectName)
	if dirs.GOOS == "windows" {
		dest += ".cmd"
	}
	if err := linkCommand(shimPath, dest); err != nil {
		return InstallResult{}, err
	}
	if dirs.GOOS == "windows" {
		if _, err := EnsureWindowsEngineCommand(binDir, ResolveEnginePath()); err != nil {
			return InstallResult{}, err
		}
	}

	files, err := ensurePathHooks(dirs.Home, dirs.GOOS, binDir)
	if err != nil {
		return InstallResult{}, err
	}
	return InstallResult{CommandPath: dest, ShellFiles: files}, nil
}

func linkCommand(src, dest string) error {
	_ = os.Remove(dest)
	if err := os.Symlink(src, dest); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func ensurePathHooks(home, goos, binDir string) ([]string, error) {
	switch goos {
	case "windows":
		files, _, err := ensureWindowsPathHook(home, binDir)
		return files, err
	default:
		return ensureUnixPathHooks(home, goos, binDir)
	}
}

// EnsureBinDirOnPATH upserts managed shell startup hooks so binDir is on PATH
// in new terminals. On Windows it also adds binDir to the user Path registry value.
func EnsureBinDirOnPATH(home, goos, binDir string) ([]string, error) {
	if binDir == "" {
		return nil, fmt.Errorf("bin directory is required")
	}
	return ensurePathHooks(home, goos, binDir)
}

func ensureUnixPathHooks(home, goos, binDir string) ([]string, error) {
	block := unixPathBlock(binDir)
	var targets []string
	for _, rc := range unixRCCandidates(home, goos) {
		if _, err := os.Stat(rc); err == nil {
			targets = append(targets, rc)
		}
	}
	if len(targets) == 0 {
		targets = []string{defaultUnixRC(home, goos)}
	}
	var updated []string
	for _, rc := range targets {
		if err := upsertPathHookFile(rc, block); err != nil {
			return nil, err
		}
		updated = append(updated, rc)
	}
	return updated, nil
}

func unixPathBlock(binDir string) string {
	return fmt.Sprintf(`%s
# Managed by msc — added on init
export PATH=%q:$PATH
%s
`, pathHookBegin, binDir, pathHookEnd)
}

func unixRCCandidates(home, goos string) []string {
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

func defaultUnixRC(home, goos string) string {
	if goos == "darwin" {
		return filepath.Join(home, ".zshrc")
	}
	return filepath.Join(home, ".bashrc")
}

func ensureWindowsPathHook(home, binDir string) ([]string, bool, error) {
	pathBlock := windowsPathBlock(binDir)
	brandBlock := windowsBrandCommandsBlock(binDir)
	targets := windowsProfileTargets(home)
	var updated []string
	changed := false
	for _, profile := range targets {
		c, err := upsertHookFile(profile, pathHookBegin, pathHookEnd, pathBlock)
		if err != nil {
			return nil, false, err
		}
		if c {
			changed = true
		}
		c, err = upsertHookFile(profile, brandHookBegin, brandHookEnd, brandBlock)
		if err != nil {
			return nil, false, err
		}
		if c {
			changed = true
		}
		updated = append(updated, profile)
	}
	if err := ensureWindowsUserPath(binDir); err != nil {
		return updated, changed, err
	}
	return updated, changed, nil
}

func windowsBrandCommandsBlock(binDir string) string {
	escaped := strings.ReplaceAll(binDir, `'`, `''`)
	return fmt.Sprintf(`%s
# Managed by msc — brand commands launch the .cmd shim (tab completion without .cmd)
$__mscBrandBin = '%s'
if (Test-Path -LiteralPath $__mscBrandBin) {
  Get-ChildItem -LiteralPath $__mscBrandBin -Filter '*.cmd' -ErrorAction SilentlyContinue | ForEach-Object {
    $__name = $_.BaseName
    if ($__name -eq 'msc') { return }
    $__cmd = $_.FullName.Replace("'", "''")
    Set-Item -Path "function:global:$__name" -Value ([scriptblock]::Create(('param([Parameter(ValueFromRemainingArguments=$true)][string[]]$args) & ''{0}'' @args' -f $__cmd)))
  }
}
%s
`, brandHookBegin, escaped, brandHookEnd)
}

func windowsPathBlock(binDir string) string {
	escaped := strings.ReplaceAll(binDir, `'`, `''`)
	return fmt.Sprintf(`%s
# Managed by msc — added on init
$__mscBin = '%s'
if ($env:PATH -notlike "*$__mscBin*") { $env:PATH = "$__mscBin;" + $env:PATH }
%s
`, pathHookBegin, escaped, pathHookEnd)
}

func upsertPathHookFile(path, block string) error {
	_, err := upsertHookFile(path, pathHookBegin, pathHookEnd, block)
	return err
}

func upsertHookFile(path, begin, end, block string) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	content := string(data)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
		content = ""
	}
	next := upsertHookBlock(content, begin, end, block)
	if next == content {
		return false, nil
	}
	return true, os.WriteFile(path, []byte(next), 0o644)
}

func upsertHookBlock(content, begin, end, block string) string {
	start := strings.Index(content, begin)
	if start < 0 {
		body := strings.TrimRight(content, "\n")
		if body != "" {
			body += "\n\n"
		}
		return body + block + "\n"
	}
	rest := content[start:]
	relEnd := strings.Index(rest, end)
	if relEnd < 0 {
		return content[:start] + block + "\n"
	}
	stop := start + relEnd + len(end)
	for stop < len(content) && (content[stop] == '\n' || content[stop] == '\r') {
		stop++
	}
	return content[:start] + block + "\n" + strings.TrimLeft(content[stop:], "\n")
}
