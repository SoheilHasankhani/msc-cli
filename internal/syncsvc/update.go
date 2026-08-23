package syncsvc

import (
	"context"
	"sync"

	"github.com/SoheilHasankhani/msc-cli/internal/gitops"
	"github.com/SoheilHasankhani/msc-cli/internal/progress"
)

// UpdateResult summarizes a default sync run (clone missing + pull cloned).
type UpdateResult struct {
	Cloned   int
	Pulled   int
	Warnings []RepoWarning
}

// RepoWarning is a per-repo failure that did not stop other repos.
type RepoWarning struct {
	Name    string
	Message string
	Op      string // "clone" or "pull"
}

// Update syncs accessible repos: clone missing, pull existing. Denied repos are skipped.
func Update(ctx context.Context, opt Options) (gitops.Plan, UpdateResult, error) {
	plan, err := Plan(ctx, opt)
	if err != nil {
		return gitops.Plan{}, UpdateResult{}, err
	}
	toClone, toPull, err := plan.SelectUpdate(opt.Names)
	if err != nil {
		return plan, UpdateResult{}, err
	}
	sources := make([]progress.Source, 0, len(toClone)+len(toPull))
	ops := make(map[string]string, len(toClone)+len(toPull))
	for _, r := range toClone {
		r := r
		ops[r.Name] = "clone"
		sources = append(sources, progress.GitCloneSource{
			ID:     r.Name,
			Label:  r.Name,
			URL:    r.URL,
			Dest:   r.Dest,
			Cloner: opt.Git,
		})
	}
	for _, r := range toPull {
		r := r
		ops[r.Name] = "pull"
		sources = append(sources, progress.GitPullSource{
			ID:     r.Name,
			Label:  r.Name,
			Path:   r.Dest,
			Puller: opt.Git,
		})
	}
	var mu sync.Mutex
	res := UpdateResult{}
	err = progress.RunLenient(ctx, sources, opt.Progress, func(u progress.Update) {
		if !u.Done {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if u.Err != nil {
			res.Warnings = append(res.Warnings, RepoWarning{
				Name:    u.ID,
				Message: repoWarningMessage(u.Err),
				Op:      ops[u.ID],
			})
			return
		}
		switch ops[u.ID] {
		case "clone":
			res.Cloned++
		case "pull":
			res.Pulled++
		}
	})
	if err != nil {
		return plan, res, err
	}
	return plan, res, nil
}

func repoWarningMessage(err error) string {
	return gitops.FormatPullError(err)
}
