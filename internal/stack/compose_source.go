package stack

import (
	"context"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/progress"
)

// ComposePullSource runs docker compose pull and emits one updatable line per image.
type ComposePullSource struct {
	Compose dockerapi.ComposeRunner
	Root    string
	File    string
	Opts    dockerapi.ComposeRunOpts
}

func (s ComposePullSource) Run(ctx context.Context, updates chan<- progress.Update) error {
	return s.Compose.Pull(ctx, s.Root, s.File, s.Opts, func(st dockerapi.ComposeStatus) {
		emitComposeStatus(ctx, updates, st)
	})
}

// ComposeUpSource runs docker compose up and emits one updatable line per container.
type ComposeUpSource struct {
	Compose dockerapi.ComposeRunner
	Root    string
	File    string
	Opts    dockerapi.ComposeRunOpts
}

func (s ComposeUpSource) Run(ctx context.Context, updates chan<- progress.Update) error {
	return s.Compose.Up(ctx, s.Root, s.File, s.Opts, func(st dockerapi.ComposeStatus) {
		emitComposeStatus(ctx, updates, st)
	})
}

// ComposeDownSource runs docker compose down and emits one updatable line per container.
type ComposeDownSource struct {
	Compose dockerapi.ComposeRunner
	Root    string
	File    string
	Opts    dockerapi.ComposeRunOpts
}

func (s ComposeDownSource) Run(ctx context.Context, updates chan<- progress.Update) error {
	return s.Compose.Down(ctx, s.Root, s.File, s.Opts, func(st dockerapi.ComposeStatus) {
		emitComposeStatus(ctx, updates, st)
	})
}

func emitComposeStatus(ctx context.Context, updates chan<- progress.Update, st dockerapi.ComposeStatus) {
	u := progress.Update{
		ID:     st.Container,
		Label:  progress.ShortImageRef(st.Container),
		Status: st.State,
		Done:   dockerapi.IsTerminalComposeStatus(st.State),
	}
	select {
	case updates <- u:
	case <-ctx.Done():
	}
}
