package gitops

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"
)

type slowOK struct{ delay time.Duration }

func (s *slowOK) LsRemote(ctx context.Context, url string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-time.After(s.delay):
		return true, nil
	}
}
func (s *slowOK) Clone(context.Context, string, string, io.Writer) error { return nil }
func (s *slowOK) Pull(context.Context, string, io.Writer) error          { return nil }

type fakeRunner struct {
	access map[string]bool
	err    error
}

func (f fakeRunner) LsRemote(_ context.Context, url string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.access[url], nil
}
func (f fakeRunner) Clone(context.Context, string, string, io.Writer) error { return nil }
func (f fakeRunner) Pull(context.Context, string, io.Writer) error          { return nil }

func TestProbeAccessRunsInParallel(t *testing.T) {
	t.Parallel()

	r := &slowOK{delay: 120 * time.Millisecond}
	start := time.Now()
	got, err := ProbeAccess(context.Background(), r, []string{"a", "b", "c", "d"})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("%v", got)
	}
	if elapsed > 350*time.Millisecond {
		t.Fatalf("probes look sequential: %s", elapsed)
	}
}

func TestProbeAccessStopsOnUnreachable(t *testing.T) {
	t.Parallel()

	r := fakeRunner{err: fmt.Errorf("%s", Message(KindUnreachable, "git@host:group/repo.git"))}
	_, err := ProbeAccess(context.Background(), r, []string{"a", "b"})
	if err == nil || !IsUnreachable(err) {
		t.Fatalf("%v", err)
	}
}

func TestProbeAccess(t *testing.T) {
	t.Parallel()

	r := fakeRunner{access: map[string]bool{"a": true, "b": false}}
	got, err := ProbeAccess(context.Background(), r, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if !got["a"] || got["b"] {
		t.Fatalf("%#v", got)
	}
}

func TestProbeAccessStopsOnAgentError(t *testing.T) {
	t.Parallel()

	r := fakeRunner{err: errNoAgent()}
	if _, err := ProbeAccess(context.Background(), r, []string{"a"}); err == nil {
		t.Fatal("expected agent error")
	}
}

func errNoAgent() error {
	return &agentError{}
}

type agentError struct{}

func (agentError) Error() string { return Message(KindNoAgent, "x") }
