package dockerapi

import (
	"context"
	"io"
)

// Container is a subset of Docker container state the engine needs.
type Container struct {
	Name           string
	ComposeService string
	Running        bool
}

// Client is the Engine API surface used by state, switch, and progress.
//
//go:generate go run go.uber.org/mock/mockgen -source=client.go -destination=mock_client.go -package=dockerapi
type Client interface {
	ListContainers(ctx context.Context) ([]Container, error)
	StopService(ctx context.Context, composeService string) error
	StartService(ctx context.Context, composeService string) error
	SignalContainer(ctx context.Context, containerName, signal string) error
	Pull(ctx context.Context, ref string) (io.ReadCloser, error)
}

// ComposeRunOpts selects which compose profiles and services to enable.
type ComposeRunOpts struct {
	Profiles []string
	Services []string // if set, only these services (compose up SERVICE…)
	NoDeps   bool
}

// StatusFn is called for each container status line emitted by compose.
type StatusFn func(ComposeStatus)

// ComposeRunner runs docker compose in a project directory.
type ComposeRunner interface {
	Pull(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts, onStatus StatusFn) error
	Up(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts, onStatus StatusFn) error
	Down(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts, onStatus StatusFn) error
	Images(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts) ([]string, error)
	Services(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts) ([]ComposeService, error)
}
