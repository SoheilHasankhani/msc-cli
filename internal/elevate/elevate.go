package elevate

import (
	"context"
	"errors"
	"fmt"
)

// ErrNeedTTY means the operation requires an interactive sudo/UAC prompt.
var ErrNeedTTY = errors.New("needs an interactive terminal to request administrator access (re-run doctor --fix in a terminal)")

// IsNeedTTY reports whether err is (or wraps) ErrNeedTTY.
func IsNeedTTY(err error) bool {
	return errors.Is(err, ErrNeedTTY)
}

// Elevator runs a privileged helper. Production re-invokes this binary
// as `msc __elevated-do ...` under sudo / UAC. Tests use Direct.
type Elevator interface {
	RunElevated(ctx context.Context, description string, args []string) error
}

// Direct runs the handler in-process (unit tests, or when already root).
type Direct struct {
	Handle func(ctx context.Context, args []string) error
}

// RunElevated implements Elevator.
func (d Direct) RunElevated(ctx context.Context, _ string, args []string) error {
	if d.Handle == nil {
		return fmt.Errorf("elevate: no handler")
	}
	return d.Handle(ctx, args)
}

// SudoArgs is the sudo argv (not including sudo itself as args[0] of the slice
// returned as `args`; `name` is "sudo").
func SudoArgs(exe, description string, inner []string, nonInteractive bool) (name string, args []string) {
	prompt := "msc needs administrator access to " + description + "\n[sudo] password: "
	args = nil
	if nonInteractive {
		args = append(args, "-n")
	}
	args = append(args, "-p", prompt, "--", exe)
	args = append(args, inner...)
	return "sudo", args
}
