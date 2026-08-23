package progress

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/SoheilHasankhani/msc-cli/internal/gitops"
)

type scriptedSource struct {
	id     string
	events []Update
	err    error
	delay  time.Duration
}

func (s scriptedSource) Run(ctx context.Context, updates chan<- Update) error {
	for _, ev := range s.events {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case updates <- ev:
		}
		if s.delay > 0 {
			time.Sleep(s.delay)
		}
	}
	return s.err
}

func TestRunBatchFansInConcurrentSources(t *testing.T) {
	t.Parallel()

	a := scriptedSource{id: "a", events: []Update{{ID: "a", Current: 1, Total: 2}, {ID: "a", Current: 2, Total: 2, Done: true}}}
	b := scriptedSource{id: "b", events: []Update{{ID: "b", Current: 5, Total: 5, Done: true}}}

	var mu sync.Mutex
	var got []Update
	err := RunBatch(context.Background(), []Source{a, b}, func(u Update) {
		mu.Lock()
		got = append(got, u)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d updates, want 3: %#v", len(got), got)
	}
	ids := map[string]bool{}
	for _, u := range got {
		ids[u.ID] = true
	}
	if !ids["a"] || !ids["b"] {
		t.Fatalf("missing source ids in %#v", got)
	}
}

func TestRunBatchWarnDoesNotFailBatch(t *testing.T) {
	t.Parallel()

	src := scriptedSource{id: "x", events: []Update{{ID: "x", Done: true, Warn: errSentinel{}}}}
	err := RunBatch(context.Background(), []Source{src}, func(Update) {})
	if err != nil {
		t.Fatalf("warn should not fail batch: %v", err)
	}
}

func TestRunBatchReturnsSourceError(t *testing.T) {
	t.Parallel()

	boom := errors.New("pull failed")
	src := scriptedSource{id: "x", events: []Update{{ID: "x", Done: true, Err: boom}}, err: boom}
	err := RunBatch(context.Background(), []Source{src}, func(Update) {})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want pull failed", err)
	}
}

func TestRunLenientIgnoresSourceErrors(t *testing.T) {
	t.Parallel()

	boom := gitops.WrapPullConflict("fatal: Not possible to fast-forward, aborting.\n")
	ok := scriptedSource{id: "ok", events: []Update{{ID: "ok", Current: 1, Total: 1, Done: true}}}
	fail := scriptedSource{id: "bad", events: []Update{{ID: "bad", Done: true, Err: boom}}}
	tty := false
	var done int
	err := RunLenient(context.Background(), []Source{ok, fail}, Options{Output: io.Discard, IsTTY: &tty}, func(u Update) {
		if u.Done && u.Err == nil {
			done++
		}
	})
	if err != nil {
		t.Fatalf("lenient batch: %v", err)
	}
	if done != 1 {
		t.Fatalf("done = %d", done)
	}
}
