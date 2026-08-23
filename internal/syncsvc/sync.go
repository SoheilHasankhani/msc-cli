package syncsvc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/SoheilHasankhani/msc-cli/internal/gitops"
	"github.com/SoheilHasankhani/msc-cli/internal/progress"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
)

// Options drives plan, list, clone, pull, and update runs.
type Options struct {
	Project   *project.Context
	Git       gitops.Runner
	CachePath string
	Now       time.Time
	Refresh   bool
	ListOnly  bool
	CloneOnly bool
	PullOnly  bool
	All       bool // legacy alias for CloneOnly when updating all available repos
	Names     []string
	Progress  progress.Options
}

// Plan builds the sync plan, using the access cache unless Refresh is set.
func Plan(ctx context.Context, opt Options) (gitops.Plan, error) {
	if opt.Project == nil || opt.Project.Manifest == nil {
		return gitops.Plan{}, fmt.Errorf("project is required")
	}
	if opt.Now.IsZero() {
		opt.Now = time.Now()
	}
	access, err := loadOrProbe(ctx, opt)
	if err != nil {
		return gitops.Plan{}, err
	}
	return gitops.BuildPlan(opt.Project.Manifest, opt.Project.ClonesDir(), access)
}

func loadOrProbe(ctx context.Context, opt Options) (map[string]bool, error) {
	if !opt.Refresh && opt.CachePath != "" {
		if cached, err := gitops.LoadAccessCache(opt.CachePath); err == nil && cached.Valid(opt.Now, gitops.DefaultAccessTTL) {
			return cached.Repos, nil
		}
	}
	if opt.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	remotes, err := gitops.ManifestRemotes(opt.Project.Manifest)
	if err != nil {
		return nil, err
	}
	prog := opt.Progress
	// Access probes are silent; clone/pull/update show per-repo progress.
	prog.Output = io.Discard
	off := false
	if prog.IsTTY == nil {
		prog.IsTTY = &off
	}
	if prog.Limit == 0 {
		prog.Limit = gitops.DefaultProbeParallel
	}
	var mu sync.Mutex
	access := make(map[string]bool, len(remotes))
	sources := make([]progress.Source, 0, len(remotes))
	for _, remote := range remotes {
		remote := remote
		sources = append(sources, progress.FuncSource{
			ID:    remote.Name,
			Label: remote.Name,
			Fn: func(ctx context.Context, emit func(progress.Update)) error {
				emit(progress.Update{Current: 0, Total: 1})
				ok, err := opt.Git.LsRemote(ctx, remote.URL)
				if err != nil {
					emit(progress.Update{Done: true, Err: err})
					return err
				}
				mu.Lock()
				access[remote.URL] = ok
				mu.Unlock()
				emit(progress.Update{Current: 1, Total: 1, Done: true})
				return nil
			},
		})
	}
	if err := progress.Run(ctx, sources, prog); err != nil {
		return nil, err
	}
	if opt.CachePath != "" {
		_ = gitops.SaveAccessCache(opt.CachePath, &gitops.AccessCache{CheckedAt: opt.Now, Repos: access})
	}
	return access, nil
}

// Clone plans, selects, and clones repos in parallel with progress.
func Clone(ctx context.Context, opt Options) (int, error) {
	plan, err := Plan(ctx, opt)
	if err != nil {
		return 0, err
	}
	selected, err := plan.Select(opt.Names, opt.All || (opt.CloneOnly && len(opt.Names) == 0))
	if err != nil {
		return 0, err
	}
	sources := make([]progress.Source, 0, len(selected))
	for _, r := range selected {
		sources = append(sources, progress.GitCloneSource{
			ID:     r.Name,
			Label:  r.Name,
			URL:    r.URL,
			Dest:   r.Dest,
			Cloner: opt.Git,
		})
	}
	if err := progress.Run(ctx, sources, opt.Progress); err != nil {
		return 0, err
	}
	return len(selected), nil
}
