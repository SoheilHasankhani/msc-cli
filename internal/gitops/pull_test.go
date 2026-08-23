package gitops

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestIsPullConflict(t *testing.T) {
	t.Parallel()
	cases := []struct {
		stderr string
		want   bool
	}{
		{"fatal: Not possible to fast-forward, aborting.\n", true},
		{"error: Your local changes to the following files would be overwritten by merge:\n", true},
		{"fatal: repository not found\n", false},
	}
	for _, tc := range cases {
		if got := IsPullConflict(tc.stderr); got != tc.want {
			t.Fatalf("IsPullConflict(%q) = %v", tc.stderr, got)
		}
	}
}

func TestIsRetryablePullStderr(t *testing.T) {
	t.Parallel()
	stderr := "kex_exchange_identification: read: Connection reset by peer\nConnection reset by 87.247.175.28 port 22\nfatal: Could not read from remote repository.\n"
	if !IsRetryablePullStderr(stderr) {
		t.Fatal("expected retryable")
	}
	if IsRetryablePullStderr("fatal: Not possible to fast-forward, aborting.\n") {
		t.Fatal("conflict should not retry")
	}
	if !IsRetryablePullStderr("error: RPC failed; curl 56 Recv failure: Connection reset by peer\nfatal: early EOF\n") {
		t.Fatal("expected clone RPC/EOF to retry")
	}
}

func TestCloneErrorRetriesTransient(t *testing.T) {
	t.Parallel()
	err := cloneError("Connection reset by peer\nfatal: Could not read from remote repository.\n", "git@host:g/r.git")
	if !IsRetryablePullError(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestCloneErrorDoesNotRetryDenied(t *testing.T) {
	t.Parallel()
	err := cloneError("ERROR: The project you were looking for could not be found.\n", "git@host:g/r.git")
	if IsRetryablePullError(err) {
		t.Fatalf("denied should not retry: %v", err)
	}
}

func TestPullNetworkMessageIsOneLine(t *testing.T) {
	t.Parallel()
	msg := PullNetworkMessage("Connection reset by 87.247.175.28 port 22\nfatal: Could not read from remote repository.\n")
	if msg == "" || strings.Contains(msg, "\n") {
		t.Fatalf("msg = %q", msg)
	}
}

func TestFormatPullErrorStripsSentinel(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("%s: %w", PullNetworkMessage("connection reset"), ErrPullTransient)
	got := FormatPullError(err)
	if strings.Contains(got, "pull transient") || strings.Contains(got, "\n") {
		t.Fatalf("got = %q", got)
	}
}

func TestPullWithRetry(t *testing.T) {
	t.Parallel()
	var calls int
	err := pullWithRetry(context.Background(), func() error {
		calls++
		if calls < 3 {
			return fmt.Errorf("%s: %w", PullNetworkMessage("connection reset"), ErrPullTransient)
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestPullWithRetryStopsOnConflict(t *testing.T) {
	t.Parallel()
	var calls int
	err := pullWithRetry(context.Background(), func() error {
		calls++
		return WrapPullConflict("fatal: Not possible to fast-forward, aborting.\n")
	})
	if err == nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	if !errors.Is(err, ErrPullConflict) {
		t.Fatalf("err = %v", err)
	}
}
