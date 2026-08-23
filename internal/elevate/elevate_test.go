package elevate

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSudoArgsInteractive(t *testing.T) {
	t.Parallel()

	name, args := SudoArgs("/usr/local/bin/msc", "update the hosts file", []string{"__elevated-do", "--payload", "/tmp/p.json"}, false)
	if name != "sudo" {
		t.Fatal(name)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "/usr/local/bin/msc") || !strings.Contains(joined, "__elevated-do") {
		t.Fatal(joined)
	}
	if strings.Contains(joined, " -n ") || args[0] == "-n" {
		t.Fatalf("interactive sudo must not use -n: %v", args)
	}
}

func TestSudoArgsNonInteractive(t *testing.T) {
	t.Parallel()

	_, args := SudoArgs("/bin/msc", "install CA", []string{"__elevated-do", "--payload", "p"}, true)
	if args[0] != "-n" {
		t.Fatalf("%v", args)
	}
}

func TestDirectRunsArgs(t *testing.T) {
	t.Parallel()

	var got []string
	el := Direct{Handle: func(_ context.Context, args []string) error {
		got = args
		return nil
	}}
	if err := el.RunElevated(context.Background(), "test", []string{"__elevated-do", "--payload", "x"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "__elevated-do --payload x" {
		t.Fatalf("%v", got)
	}
}

func TestDirectPropagatesError(t *testing.T) {
	t.Parallel()
	el := Direct{Handle: func(context.Context, []string) error { return errors.New("nope") }}
	if err := el.RunElevated(context.Background(), "x", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestNeedTTYError(t *testing.T) {
	t.Parallel()
	if !IsNeedTTY(ErrNeedTTY) {
		t.Fatal("IsNeedTTY")
	}
}
