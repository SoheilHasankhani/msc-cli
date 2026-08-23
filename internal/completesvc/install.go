package completesvc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/spf13/cobra"
)

// Options drives a persistent shell completion install.
type Options struct {
	Root *cobra.Command
	Dirs paths.Resolver
}

// Result describes written completion assets.
type Result struct {
	BashScript       string
	ZshScript        string
	PowerShellScript string
	RCFiles          []string
}

// Install writes completion scripts under the config dir and upserts shell startup hooks.
func Install(opt Options) (Result, error) {
	if opt.Root == nil {
		return Result{}, fmt.Errorf("completion root command is required")
	}
	dirs := opt.Dirs
	if dirs.Home == "" {
		dirs = paths.Default()
	}
	if err := os.MkdirAll(dirs.ConfigDir(), 0o755); err != nil {
		return Result{}, err
	}

	names, err := BrandNames(dirs)
	if err != nil {
		return Result{}, err
	}

	bashPath := dirs.CompletionBash()
	if err := writeBashScript(bashPath, opt.Root, names); err != nil {
		return Result{}, err
	}
	zshPath := dirs.CompletionZsh()
	if err := writeZshScript(zshPath, opt.Root, names); err != nil {
		return Result{}, err
	}
	psPath := dirs.CompletionPowerShell()
	if err := writePowerShellScript(psPath, opt.Root, names); err != nil {
		return Result{}, err
	}

	goos := dirs.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	rcFiles, err := EnsureShellHooks(dirs.Home, goos, bashPath, zshPath, psPath)
	if err != nil {
		return Result{}, err
	}
	return Result{BashScript: bashPath, ZshScript: zshPath, PowerShellScript: psPath, RCFiles: rcFiles}, nil
}

func writeBashScript(path string, root *cobra.Command, brands []string) error {
	var buf bytes.Buffer
	if err := root.GenBashCompletionV2(&buf, false); err != nil {
		return err
	}
	if err := WriteBashBrandCompleters(&buf, brands); err != nil {
		return err
	}
	return atomicWrite(path, buf.Bytes())
}

func writeZshScript(path string, root *cobra.Command, brands []string) error {
	var buf bytes.Buffer
	if err := root.GenZshCompletionNoDesc(&buf); err != nil {
		return err
	}
	if err := WriteZshBrandCompleters(&buf, brands); err != nil {
		return err
	}
	return atomicWrite(path, buf.Bytes())
}

func writePowerShellScript(path string, root *cobra.Command, brands []string) error {
	var buf bytes.Buffer
	if err := root.GenPowerShellCompletion(&buf); err != nil {
		return err
	}
	if err := WritePowerShellBrandCompleters(&buf, brands); err != nil {
		return err
	}
	return atomicWrite(path, buf.Bytes())
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".msc-completion-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}
