package progress

import (
	"context"
	"io"
	"strings"
	"testing"
)

type fakeCloner struct {
	err error
}

func (f fakeCloner) Clone(context.Context, string, string, io.Writer) error {
	return f.err
}

type writingCloner struct{}

func (writingCloner) Clone(_ context.Context, _, _ string, w io.Writer) error {
	_, _ = io.WriteString(w, "Receiving objects:  50% (50/100)\n")
	return nil
}

func TestGitCloneSourceEmitsProgress(t *testing.T) {
	t.Parallel()

	src := GitCloneSource{ID: "doctor-api", Label: "doctor-api", URL: "file://x", Dest: "/tmp/x", Cloner: writingCloner{}}
	updates := make(chan Update, 16)
	if err := src.Run(context.Background(), updates); err != nil {
		t.Fatal(err)
	}
	close(updates)
	var sawPartial, sawDone bool
	for u := range updates {
		if u.Current == 40 && u.Total == 100 {
			sawPartial = true
		}
		if u.Done && u.Err == nil {
			sawDone = true
		}
	}
	if !sawPartial || !sawDone {
		t.Fatal("expected partial and done updates")
	}
}

func TestGitCloneSourceSurfacesCloneError(t *testing.T) {
	t.Parallel()

	src := GitCloneSource{ID: "r", Label: "r", Cloner: fakeCloner{err: io.ErrUnexpectedEOF}}
	updates := make(chan Update, 8)
	err := src.Run(context.Background(), updates)
	if err == nil {
		t.Fatal("expected clone error")
	}
	close(updates)
	var last Update
	for u := range updates {
		last = u
	}
	if last.Err == nil || !last.Done {
		t.Fatalf("last = %#v", last)
	}
	if !strings.Contains(err.Error(), "EOF") {
		t.Fatalf("err = %v", err)
	}
}
