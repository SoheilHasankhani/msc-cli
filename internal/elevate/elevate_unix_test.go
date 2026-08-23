//go:build unix

package elevate

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestProcessRootRunsBinaryDirectly(t *testing.T) {
	t.Parallel()

	var name string
	var args []string
	p := Process{
		Executable: func() (string, error) { return "/bin/msc", nil },
		UID:        func() int { return 0 },
		TTY:        func() bool { return true },
		Cmd: func(_ context.Context, n string, a []string) error {
			name, args = n, a
			return nil
		},
	}
	if err := p.RunElevated(context.Background(), "hosts", []string{"__elevated-do", "--payload", "p"}); err != nil {
		t.Fatal(err)
	}
	if name != "/bin/msc" || strings.Join(args, " ") != "__elevated-do --payload p" {
		t.Fatalf("%s %v", name, args)
	}
}

func TestProcessNonTTYReturnsNeedTTY(t *testing.T) {
	t.Parallel()

	var calls int
	p := Process{
		Executable: func() (string, error) { return "/bin/msc", nil },
		UID:        func() int { return 1000 },
		TTY:        func() bool { return false },
		Cmd: func(context.Context, string, []string) error {
			calls++
			return errors.New("sudo: a password is required")
		},
	}
	err := p.RunElevated(context.Background(), "hosts", []string{"__elevated-do"})
	if !IsNeedTTY(err) {
		t.Fatalf("%v", err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestProcessTriesPasswordlessSudoFirst(t *testing.T) {
	t.Parallel()

	var first []string
	var second []string
	p := Process{
		Executable: func() (string, error) { return "/bin/msc", nil },
		UID:        func() int { return 1000 },
		TTY:        func() bool { return true },
		Cmd: func(_ context.Context, _ string, a []string) error {
			if first == nil {
				first = append([]string(nil), a...)
				return errors.New("sudo: a password is required")
			}
			second = append([]string(nil), a...)
			return nil
		},
	}
	if err := p.RunElevated(context.Background(), "hosts", []string{"__elevated-do"}); err != nil {
		t.Fatal(err)
	}
	if len(first) == 0 || first[0] != "-n" {
		t.Fatalf("first=%v", first)
	}
	if len(second) == 0 || second[0] == "-n" {
		t.Fatalf("second=%v", second)
	}
}
