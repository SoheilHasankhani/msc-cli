package progress

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
)

type readerPuller struct{ body string }

func (p readerPuller) Pull(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(p.body)), nil
}

type flakyPuller struct {
	calls atomic.Int32
	fail  int
	body  string
}

func (p *flakyPuller) Pull(context.Context, string) (io.ReadCloser, error) {
	n := int(p.calls.Add(1))
	if n <= p.fail {
		return nil, fmt.Errorf("connection reset by peer: %w", ErrDockerPullTransient)
	}
	return io.NopCloser(strings.NewReader(p.body)), nil
}

func TestDockerPullSourceEmitsDone(t *testing.T) {
	t.Parallel()

	src := DockerPullSource{
		Ref:    "alpine:latest",
		Puller: readerPuller{body: `{"status":"Already exists","id":"latest"}` + "\n"},
	}
	ch := make(chan Update, 8)
	if err := src.Run(context.Background(), ch); err != nil {
		t.Fatal(err)
	}
	close(ch)
	var last Update
	for u := range ch {
		last = u
	}
	if !last.Done || last.ID != "alpine:latest" {
		t.Fatalf("last = %#v", last)
	}
}

func TestDockerPullSourceRetriesTransient(t *testing.T) {
	t.Parallel()

	src := DockerPullSource{
		Ref:    "alpine:latest",
		Puller: &flakyPuller{fail: 2, body: `{"status":"Already exists","id":"latest"}` + "\n"},
	}
	ch := make(chan Update, 32)
	if err := src.Run(context.Background(), ch); err != nil {
		t.Fatal(err)
	}
	close(ch)
	if src.Puller.(*flakyPuller).calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", src.Puller.(*flakyPuller).calls.Load())
	}
}

func TestDockerPullSourceFanOutIDs(t *testing.T) {
	t.Parallel()

	src := DockerPullSource{
		Ref:       "alpine:latest",
		Puller:    readerPuller{body: `{"status":"Already exists","id":"latest"}` + "\n"},
		FanOutIDs: []string{"wallet", "patient"},
	}
	ch := make(chan Update, 16)
	if err := src.Run(context.Background(), ch); err != nil {
		t.Fatal(err)
	}
	close(ch)
	seen := map[string]bool{}
	for u := range ch {
		if u.Done {
			seen[u.ID] = true
		}
		if u.Label != u.ID {
			t.Fatalf("label = %q id = %q", u.Label, u.ID)
		}
	}
	if !seen["wallet"] || !seen["patient"] {
		t.Fatalf("done rows = %#v", seen)
	}
}

func TestDockerPullSourceDoesNotRetryAuthFailure(t *testing.T) {
	t.Parallel()

	var calls int
	src := DockerPullSource{
		Ref: "registry.example.com/app:latest",
		Puller: pullerFunc(func(context.Context, string) (io.ReadCloser, error) {
			calls++
			return nil, errors.New(`docker pull: pull access denied`)
		}),
	}
	ch := make(chan Update, 8)
	err := src.Run(context.Background(), ch)
	close(ch)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

type pullerFunc func(context.Context, string) (io.ReadCloser, error)

func (f pullerFunc) Pull(ctx context.Context, ref string) (io.ReadCloser, error) {
	return f(ctx, ref)
}

func TestIsRetryableDockerPullError(t *testing.T) {
	t.Parallel()

	if !IsRetryableDockerPullError(fmt.Errorf("connection reset: %w", ErrDockerPullTransient)) {
		t.Fatal("transient wrapper")
	}
	if IsRetryableDockerPullError(errors.New("pull access denied")) {
		t.Fatal("auth should not retry")
	}
	if IsRetryableDockerPullError(context.Canceled) {
		t.Fatal("cancel should not retry")
	}
}
