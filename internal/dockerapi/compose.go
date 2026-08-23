package dockerapi

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/SoheilHasankhani/msc-cli/internal/usererr"
)

// overlayFileName matches nginxcfg.OverlayFileName (kept here to avoid an import cycle).
const overlayFileName = "docker-compose.msc.yml"

// ExecCompose shells out to the system `docker compose` binary.
type ExecCompose struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (e ExecCompose) cmd(ctx context.Context, workDir, composeFile string, profiles []string, args ...string) *exec.Cmd {
	all := []string{"compose", "-f", composeFile}
	overlayRel := path.Join(filepath.Dir(composeFile), overlayFileName)
	overlayAbs := overlayRel
	if !filepath.IsAbs(overlayAbs) {
		overlayAbs = filepath.Join(workDir, filepath.FromSlash(overlayRel))
	}
	if _, err := os.Stat(overlayAbs); err == nil {
		all = append(all, "-f", overlayRel)
	}
	for _, profile := range profiles {
		all = append(all, "--profile", profile)
	}
	all = append(all, args...)
	cmd := exec.CommandContext(ctx, "docker", all...)
	cmd.Dir = workDir
	return cmd
}

// Pull pulls compose images for the selected profile(s).
func (e ExecCompose) Pull(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts, onStatus StatusFn) error {
	return e.run(ctx, workDir, composeFile, opts, onStatus, "pull", "--ignore-pull-failures")
}

// Up starts the stack without pulling (images are pulled separately via compose pull).
func (e ExecCompose) Up(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts, onStatus StatusFn) error {
	return e.run(ctx, workDir, composeFile, opts, onStatus, composeUpArgs(opts)...)
}

func composeUpArgs(opts ComposeRunOpts) []string {
	args := []string{"up", "-d", "--pull", "never", "--remove-orphans"}
	if opts.NoDeps {
		args = append(args, "--no-deps")
	}
	return append(args, opts.Services...)
}

// Down stops the stack.
func (e ExecCompose) Down(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts, onStatus StatusFn) error {
	return e.run(ctx, workDir, composeFile, opts, onStatus, "down")
}

func (e ExecCompose) run(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts, onStatus StatusFn, args ...string) error {
	cmd := e.cmd(ctx, workDir, composeFile, opts.Profiles, args...)
	if onStatus == nil {
		var buf bytes.Buffer
		cmd.Stdout = io.Discard
		if e.Stderr != nil {
			cmd.Stderr = io.MultiWriter(e.Stderr, &buf)
		} else {
			cmd.Stderr = &buf
		}
		if err := cmd.Run(); err != nil {
			return usererr.Compose(fmt.Errorf("docker compose %s: %w", args[0], err), buf.String())
		}
		return nil
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("docker compose %s stdout pipe: %w", args[0], err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("docker compose %s stderr pipe: %w", args[0], err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("docker compose %s: %w", args[0], err)
	}

	var wg sync.WaitGroup
	scan := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			if ctx.Err() != nil {
				return
			}
			DispatchComposeLine(sc.Text(), onStatus)
		}
	}
	wg.Add(2)
	go scan(stdout)
	go scan(stderr)
	wg.Wait()

	if ctx.Err() != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return ctx.Err()
	}
	waitErr := cmd.Wait()
	if waitErr != nil {
		var buf bytes.Buffer
		return usererr.Compose(fmt.Errorf("docker compose %s: %w", args[0], waitErr), buf.String())
	}
	return nil
}

// Images lists image refs from `docker compose config --images`.
func (e ExecCompose) Images(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts) ([]string, error) {
	var out bytes.Buffer
	cmd := e.cmd(ctx, workDir, composeFile, opts.Profiles, "config", "--images")
	cmd.Stdout = &out
	var buf bytes.Buffer
	if e.Stderr != nil {
		cmd.Stderr = io.MultiWriter(e.Stderr, &buf)
	} else {
		cmd.Stderr = &buf
	}
	if err := cmd.Run(); err != nil {
		return nil, usererr.Compose(fmt.Errorf("docker compose config --images: %w", err), buf.String())
	}
	var images []string
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			images = append(images, line)
		}
	}
	return images, nil
}
