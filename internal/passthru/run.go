package passthru

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Runner executes a planned spec. Tests inject a fake.
type Runner func(ctx context.Context, spec Spec) error

// Exec runs the spec with the given stdio (defaults to discarding if nil).
func Exec(ctx context.Context, spec Spec, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", spec.Name, err)
	}
	return nil
}
