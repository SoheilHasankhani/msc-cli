package progress

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/gitops"
)

type fakePuller struct {
	err error
}

func (f fakePuller) Pull(_ context.Context, _ string, w io.Writer) error {
	if w != nil {
		_, _ = io.WriteString(w, "Receiving objects: 100% (10/10)\n")
	}
	return f.err
}

func TestGitPullSourceEmitsProgress(t *testing.T) {
	t.Parallel()

	src := GitPullSource{ID: "identity-api", Label: "identity-api", Path: "/tmp/x", Puller: fakePuller{}}
	updates := make(chan Update, 16)
	if err := src.Run(context.Background(), updates); err != nil {
		t.Fatal(err)
	}
	close(updates)
	var sawPartial, sawDone bool
	for u := range updates {
		if u.Total == 100 && u.Current == 80 {
			sawPartial = true
		}
		if u.Done && u.Err == nil {
			sawDone = true
		}
	}
	if !sawPartial || !sawDone {
		t.Fatal("expected progress and successful done")
	}
}

func TestGitPullSourceReportsErrorWithoutFailingRun(t *testing.T) {
	t.Parallel()

	conflict := gitops.WrapPullConflict("fatal: Not possible to fast-forward, aborting.\n")
	src := GitPullSource{ID: "doctor-api", Label: "doctor-api", Puller: fakePuller{err: conflict}}
	updates := make(chan Update, 8)
	if err := src.Run(context.Background(), updates); err != nil {
		t.Fatalf("Run should continue batch: %v", err)
	}
	close(updates)
	var last Update
	for u := range updates {
		last = u
	}
	if !last.Done || !errors.Is(last.Err, gitops.ErrPullConflict) {
		t.Fatalf("last = %#v", last)
	}
}
