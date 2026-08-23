//go:build windows

package elevate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Process re-invokes the current binary with the UAC runas verb and waits.
type Process struct {
	Executable func() (string, error)
	Cmd        func(ctx context.Context, name string, args []string) error
}

// NewProcess returns the production UAC elevator.
func NewProcess() Elevator {
	return Process{}
}

func (p Process) exe() (string, error) {
	if p.Executable != nil {
		return p.Executable()
	}
	return os.Executable()
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
	ps := fmt.Sprintf(
		"Start-Process -FilePath %s -ArgumentList %s -Verb RunAs -Wait",
		psQuote(exe),
		psQuote(strings.Join(args, " ")),
	)
	if err := p.cmd(ctx, "powershell", []string{"-NoProfile", "-Command", ps}); err != nil {
		return fmt.Errorf("UAC %s: %w", description, err)
	}
	return nil
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
