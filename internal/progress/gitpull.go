package progress

import (
	"context"
	"io"
)

// GitPuller fast-forwards one existing clone with optional progress output.
type GitPuller interface {
	Pull(ctx context.Context, repoPath string, progress io.Writer) error
}

// GitPullSource pulls one repo and emits progress from git stderr.
// Pull failures are reported on the final Update but Run returns nil so batch work continues.
type GitPullSource struct {
	ID     string
	Label  string
	Path   string
	Puller GitPuller
}

// Run implements Source.
func (s GitPullSource) Run(ctx context.Context, updates chan<- Update) error {
	updates <- Update{ID: s.ID, Label: s.Label, Current: 0, Total: 1}
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Puller.Pull(ctx, s.Path, pw)
		_ = pw.Close()
	}()
	_ = ParseCloneProgress(pr, s.ID, s.Label, func(u Update) {
		if u.Done {
			return
		}
		updates <- u
	})
	if err := <-errCh; err != nil {
		updates <- Update{ID: s.ID, Label: s.Label, Current: 1, Total: 1, Done: true, Err: err}
		return nil
	}
	updates <- Update{ID: s.ID, Label: s.Label, Current: 1, Total: 1, Done: true}
	return nil
}
