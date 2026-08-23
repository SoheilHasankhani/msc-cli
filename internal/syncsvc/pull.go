package syncsvc

import (
	"context"
	"sync"

	"github.com/SoheilHasankhani/msc-cli/internal/progress"
)

// PullWarning is a per-repo pull failure that did not stop other repos.
type PullWarning = RepoWarning

// PullResult summarizes a sync --pull-only run.
type PullResult struct {
	Pulled   int
	Warnings []PullWarning
}

// Pull fast-forwards selected cloned repos in parallel. Conflicts warn per repo only.
func Pull(ctx context.Context, opt Options) (PullResult, error) {
	plan, err := Plan(ctx, opt)
	if err != nil {
		return PullResult{}, err
	}
	pullAll := opt.All || len(opt.Names) == 0
	selected, err := plan.SelectPull(opt.Names, pullAll)
	if err != nil {
		return PullResult{}, err
	}
	sources := make([]progress.Source, 0, len(selected))
	for _, r := range selected {
		r := r
		sources = append(sources, progress.GitPullSource{
			ID:     r.Name,
			Label:  r.Name,
			Path:   r.Dest,
			Puller: opt.Git,
		})
	}
	var mu sync.Mutex
	res := PullResult{}
	err = progress.RunLenient(ctx, sources, opt.Progress, func(u progress.Update) {
		if !u.Done {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if u.Err != nil {
			res.Warnings = append(res.Warnings, PullWarning{Name: u.ID, Message: repoWarningMessage(u.Err), Op: "pull"})
			return
		}
		res.Pulled++
	})
	if err != nil {
		return res, err
	}
	return res, nil
}
