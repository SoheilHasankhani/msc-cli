package progress

import (
	"context"
	"errors"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Options control how Run renders a batch.
type Options struct {
	Output io.Writer
	IsTTY  *bool
	Limit  int // max concurrent sources; 0 means no limit
}

// Run executes sources concurrently and renders progress.
// Returns ErrInterrupted when the user presses Ctrl+C on a TTY.
func Run(ctx context.Context, sources []Source, opt Options) error {
	err := run(ctx, sources, opt, false, nil)
	if errors.Is(err, ErrInterrupted) {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// RunLenient is like Run but per-source failures do not fail the batch.
// onDone is called for every finished source (Update.Done).
// Returns context.Canceled or ErrInterrupted if the batch was aborted.
func RunLenient(ctx context.Context, sources []Source, opt Options, onDone func(Update)) error {
	err := run(ctx, sources, opt, true, onDone)
	if aborted(err) {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func aborted(err error) bool {
	return errors.Is(err, ErrInterrupted) || errors.Is(err, context.Canceled)
}

func run(ctx context.Context, sources []Source, opt Options, lenient bool, onDone func(Update)) error {
	out := opt.Output
	if out == nil {
		out = os.Stderr
	}
	tty := false
	if opt.IsTTY != nil {
		tty = *opt.IsTTY
	} else {
		out, tty = ResolveOutput(out)
	}
	if !tty {
		return runFallback(ctx, sources, out, opt.Limit, lenient, onDone)
	}
	return runCharm(ctx, sources, out, opt.Limit, lenient, onDone)
}

func runFallback(ctx context.Context, sources []Source, out io.Writer, limit int, lenient bool, onDone func(Update)) error {
	last := map[string]string{}
	return runBatch(ctx, sources, func(u Update) {
		line := FormatFallback(u)
		if u.Status != "" {
			if last[u.ID] == line {
				if onDone != nil && u.Done {
					onDone(u)
				}
				return
			}
			last[u.ID] = line
		}
		WriteFallback(out, u)
		if onDone != nil && u.Done {
			onDone(u)
		}
	}, limit, lenient)
}

func runCharm(ctx context.Context, sources []Source, out io.Writer, limit int, lenient bool, onDone func(Update)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	opts := []tea.ProgramOption{tea.WithOutput(out)}
	if isFileTTY(out) {
		opts = append(opts, tea.WithInput(os.Stdin))
	} else {
		opts = append(opts, tea.WithInput(nil), tea.WithoutRenderer())
	}

	p := tea.NewProgram(newTeaModel(80, cancel), opts...)
	var batchErr error
	go func() {
		batchErr = runBatch(ctx, sources, func(u Update) {
			p.Send(updateMsg(u))
			if onDone != nil && u.Done {
				onDone(u)
			}
		}, limit, lenient)
		p.Send(quitMsg{err: batchErr})
	}()

	finalModel, runErr := p.Run()
	if batchErr != nil {
		return batchErr
	}
	if runErr != nil {
		return runErr
	}
	if m, ok := finalModel.(teaModel); ok && m.interrupted {
		return ErrInterrupted
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}
