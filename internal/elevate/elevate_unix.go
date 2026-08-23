//go:build unix

package elevate

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/term"
)

// Process re-invokes the current binary under sudo.
type Process struct {
	Executable func() (string, error)
	UID        func() int
	TTY        func() bool
	Cmd        func(ctx context.Context, name string, args []string) error
}

// NewProcess returns the production sudo elevator.
func NewProcess() Elevator {
	return Process{}
}

func (p Process) exe() (string, error) {
	if p.Executable != nil {
		return p.Executable()
	}
	return os.Executable()
}

func (p Process) uid() int {
	if p.UID != nil {
		return p.UID()
	}
	return os.Geteuid()
}

func (p Process) tty() bool {
	if p.TTY != nil {
		return p.TTY()
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func (p Process) cmd(ctx context.Context, name string, args []string) error {
	if p.Cmd != nil {
		return p.Cmd(ctx, name, args)
	}
	c := exec.CommandContext(ctx, name, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// RunElevated implements Elevator.
func (p Process) RunElevated(ctx context.Context, description string, args []string) error {
	exe, err := p.exe()
	if err != nil {
		return err
	}
	if p.uid() == 0 {
		return p.cmd(ctx, exe, args)
	}
	name, sudoArgs := SudoArgs(exe, description, args, true)
	if err := p.cmd(ctx, name, sudoArgs); err == nil {
		return nil
	}
	if !p.tty() {
		return fmt.Errorf("%w", ErrNeedTTY)
	}
	name, sudoArgs = SudoArgs(exe, description, args, false)
	if err := p.cmd(ctx, name, sudoArgs); err != nil {
		return fmt.Errorf("sudo %s: %w", description, err)
	}
	return nil
}
