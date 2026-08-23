package gitops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// Runner is the git surface used by sync (and later init).
//
//go:generate go run go.uber.org/mock/mockgen -source=runner.go -destination=mock_runner.go -package=gitops
type Runner interface {
	LsRemote(ctx context.Context, repoURL string) (accessible bool, err error)
	Clone(ctx context.Context, repoURL, destPath string, progress io.Writer) error
	Pull(ctx context.Context, repoPath string, progress io.Writer) error
}

// Exec shells out to the system git binary (SSH agent, config, and keys as-is).
type Exec struct {
	Git     string
	Timeout time.Duration // 0 = DefaultLsRemoteTimeout
}

func (e Exec) bin() string {
	if e.Git != "" {
		return e.Git
	}
	return "git"
}

// LsRemote reports whether the current SSH identity can read repoURL.
// Access denial is accessible=false with a nil error so sync can omit the repo.
// A missing SSH agent is a hard error.
func (e Exec) LsRemote(ctx context.Context, repoURL string) (bool, error) {
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = DefaultLsRemoteTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, e.bin(), "ls-remote", repoURL)
	cmd.Env = NonInteractiveGitEnv(os.Environ())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		return true, nil
	}
	if ctx.Err() != nil {
		return false, fmt.Errorf("%s", Message(KindUnreachable, repoURL))
	}
	kind := Classify(stderr.String())
	switch kind {
	case KindDenied:
		return false, nil
	case KindNoAgent, KindUnreachable:
		return false, fmt.Errorf("%s", Message(kind, repoURL))
	default:
		if kind == KindOther && stderr.Len() == 0 {
			return false, fmt.Errorf("git ls-remote %s failed", repoURL)
		}
		return false, fmt.Errorf("%s: %s", Message(kind, repoURL), stderr.String())
	}
}

// Clone runs `git clone --progress` so stderr can feed GitCloneSource.
// Transient SSH/network failures are retried like image pull (DefaultPullRetries).
func (e Exec) Clone(ctx context.Context, repoURL, destPath string, progress io.Writer) error {
	_, statErr := os.Stat(destPath)
	existed := statErr == nil || !os.IsNotExist(statErr)
	env := NonInteractiveGitEnv(os.Environ())
	bin := e.bin()
	return pullWithRetry(ctx, func() error {
		if !existed {
			_ = os.RemoveAll(destPath)
		}
		cmd := exec.CommandContext(ctx, bin, "clone", "--progress", repoURL, destPath)
		cmd.Env = env
		var captured bytes.Buffer
		if progress != nil {
			w := io.MultiWriter(progress, &captured)
			cmd.Stdout = w
			cmd.Stderr = w
		} else {
			cmd.Stderr = &captured
		}
		if err := cmd.Run(); err != nil {
			if !existed {
				_ = os.RemoveAll(destPath)
			}
			return cloneError(captured.String(), repoURL)
		}
		return nil
	})
}

// Pull fast-forwards an existing clone with retries on transient SSH/network errors.
func (e Exec) Pull(ctx context.Context, repoPath string, progress io.Writer) error {
	env := NonInteractiveGitEnv(os.Environ())
	bin := e.bin()
	return pullWithRetry(ctx, func() error {
		_, err := e.pullOnce(ctx, bin, repoPath, env, progress)
		return err
	})
}

func (e Exec) pullOnce(ctx context.Context, bin, repoPath string, env []string, progress io.Writer) (string, error) {
	cmd := exec.CommandContext(ctx, bin, "-C", repoPath, "pull", "--ff-only", "--progress")
	cmd.Env = env
	var stderr bytes.Buffer
	if progress != nil {
		w := io.MultiWriter(progress, &stderr)
		cmd.Stdout = w
		cmd.Stderr = w
	} else {
		cmd.Stderr = &stderr
	}
	if err := cmd.Run(); err != nil {
		out := stderr.String()
		return out, pullError(out, repoPath)
	}
	return stderr.String(), nil
}

func pullError(stderr, repoPath string) error {
	if IsPullConflict(stderr) {
		return WrapPullConflict(stderr)
	}
	kind := Classify(stderr)
	switch kind {
	case KindDenied, KindNoAgent:
		return fmt.Errorf("%s", Message(kind, repoPath))
	case KindUnreachable:
		return fmt.Errorf("%s: %w", PullNetworkMessage(stderr), ErrPullTransient)
	}
	if IsRetryablePullStderr(stderr) {
		return fmt.Errorf("%s: %w", PullNetworkMessage(stderr), ErrPullTransient)
	}
	if stderr != "" {
		return fmt.Errorf("%s", PullUserMessage(stderr))
	}
	return fmt.Errorf("pull failed")
}

func pullWithRetry(ctx context.Context, fn func() error) error {
	attempts := DefaultPullRetries
	var lastErr error
	for i := 1; i <= attempts; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if i == attempts || !IsRetryablePullError(lastErr) {
			return lastErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i) * 500 * time.Millisecond):
		}
	}
	return lastErr
}

// IsRetryablePullError reports whether another pull attempt may succeed.
func IsRetryablePullError(err error) bool {
	return errors.Is(err, ErrPullTransient)
}
