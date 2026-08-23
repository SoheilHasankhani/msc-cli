package update

import (
	"context"
	"fmt"
	"runtime"

	"github.com/SoheilHasankhani/msc-cli/internal/progress"
)

// Releaser returns the latest published release.
type Releaser interface {
	Latest(ctx context.Context) (Release, error)
}

// Options controls a self-update or --check run.
type Options struct {
	Current   string
	Dest      string
	GOOS      string
	GOARCH    string
	Force     bool
	CheckOnly bool
	Client    Releaser
	Fetch     Fetcher
	Progress  progress.Options
}

// Result is the user-visible outcome of Run.
type Result struct {
	Current string
	Latest  string
	Updated bool
	Skipped bool
	Message string
}

// Run checks GitHub Releases and optionally replaces dest.
func Run(ctx context.Context, opt Options) (Result, error) {
	if opt.GOOS == "" {
		opt.GOOS = runtime.GOOS
	}
	if opt.GOARCH == "" {
		opt.GOARCH = runtime.GOARCH
	}
	if opt.Client == nil {
		return Result{}, fmt.Errorf("no release client")
	}
	rel, err := opt.Client.Latest(ctx)
	if err != nil {
		return Result{}, err
	}
	p, err := PlanUpdate(opt.Current, rel, opt.GOOS, opt.GOARCH)
	if err != nil {
		return Result{}, err
	}
	if p.AlreadyLatest && !opt.Force {
		return Result{
			Current: p.Current,
			Latest:  p.Latest,
			Skipped: true,
			Message: fmt.Sprintf("msc %s is already the latest", p.Latest),
		}, nil
	}
	if p.AlreadyLatest && opt.Force {
		p, err = PlanAssets(opt.Current, rel, opt.GOOS, opt.GOARCH)
		if err != nil {
			return Result{}, err
		}
	}
	if opt.CheckOnly {
		return Result{
			Current: p.Current,
			Latest:  p.Latest,
			Message: fmt.Sprintf("update available: %s → %s", displayVersion(p.Current), p.Latest),
		}, nil
	}
	if opt.Fetch == nil {
		return Result{}, fmt.Errorf("no downloader")
	}
	apply := func(ctx context.Context) error {
		return Apply(ctx, p, opt.Dest, opt.GOOS, opt.Fetch)
	}
	if opt.Progress.Output != nil {
		src := progress.FuncSource{
			ID:    "self-update",
			Label: "msc " + p.Latest,
			Fn: func(ctx context.Context, emit func(progress.Update)) error {
				steps := []string{"checksums", "download", "verify", "install"}
				for i, label := range steps {
					emit(progress.Update{ID: "self-update", Label: label, Current: int64(i), Total: int64(len(steps))})
				}
				err := apply(ctx)
				emit(progress.Update{ID: "self-update", Label: "msc " + p.Latest, Current: int64(len(steps)), Total: int64(len(steps)), Done: true, Err: err})
				return err
			},
		}
		if err := progress.Run(ctx, []progress.Source{src}, opt.Progress); err != nil {
			return Result{}, err
		}
	} else if err := apply(ctx); err != nil {
		return Result{}, err
	}
	return Result{
		Current: p.Current,
		Latest:  p.Latest,
		Updated: true,
		Message: fmt.Sprintf("updated msc to %s", p.Latest),
	}, nil
}

func displayVersion(v string) string {
	if v == "" {
		return "dev"
	}
	return v
}
