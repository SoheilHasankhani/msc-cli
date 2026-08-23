package stack

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/nginxcfg"
	"github.com/SoheilHasankhani/msc-cli/internal/progress"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
)

const defaultPullParallel = 4

// RunOpts selects compose profiles for up/down. Empty Profiles uses the manifest default.
type RunOpts struct {
	Profiles []string
}

// UpOpts configures a stack up run.
type UpOpts struct {
	RunOpts
	SkipPull bool
	PullOnly bool
}

// DownOpts configures a stack down run.
type DownOpts struct {
	RunOpts
}

// Up pulls images (with progress bars) then starts the stack with live per-service status lines.
func Up(ctx context.Context, p *project.Context, compose dockerapi.ComposeRunner, prog progress.Options, opts UpOpts) error {
	run := composeRunOpts(p, opts.RunOpts)
	err := progress.Run(ctx, []progress.Source{StackUpSource{
		Compose:      compose,
		Root:         p.Root,
		File:         p.Manifest.Layout.ComposeFile,
		Opts:         run,
		SkipPull:     opts.SkipPull,
		PullOnly:     opts.PullOnly,
		SkipServices: nginxcfg.SourceModeNames(nginxcfg.GeneratedFile(p.ConfigDir()), nginxcfg.OverlayServices(p.Manifest)),
	}}, prog)
	if aborted(err) {
		return err
	}
	return err
}

func aborted(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, progress.ErrInterrupted)
}

// Down stops the stack with per-container status lines.
func Down(ctx context.Context, p *project.Context, compose dockerapi.ComposeRunner, prog progress.Options, opts DownOpts) error {
	run := composeRunOpts(p, opts.RunOpts)
	err := progress.Run(ctx, []progress.Source{ComposeDownSource{
		Compose: compose,
		Root:    p.Root,
		File:    p.Manifest.Layout.ComposeFile,
		Opts:    run,
	}}, prog)
	if aborted(err) {
		return err
	}
	return err
}

// ProfileLabel formats the effective compose profile(s) for status output.
func ProfileLabel(p *project.Context, opts RunOpts) string {
	if p == nil || p.Manifest == nil {
		return ""
	}
	return strings.Join(p.Manifest.ComposeProfiles(opts.Profiles), ", ")
}

func composeRunOpts(p *project.Context, opts RunOpts) dockerapi.ComposeRunOpts {
	return dockerapi.ComposeRunOpts{Profiles: p.Manifest.ComposeProfiles(opts.Profiles)}
}

// Discard is a helper so callers can pass a writer without importing io everywhere.
func Discard() io.Writer { return io.Discard }
