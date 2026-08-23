package switchsvc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/nginxcfg"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/state"
)

// Options is the input for a Source/Docker mode switch.
type Options struct {
	Project *project.Context
	Name    string
	To      state.Mode
	Docker  dockerapi.Client
	Compose dockerapi.ComposeRunner // creates a missing container when the stack is already up
}

// Result is the outcome of a successful switch.
type Result struct {
	Service  manifest.ServiceDef
	From     state.Mode
	Mode     state.Mode
	Reminder string
}

// ParseMode accepts the --to flag value. An empty string means "toggle".
func ParseMode(s string) (state.Mode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return "", nil
	case string(state.ModeSource):
		return state.ModeSource, nil
	case string(state.ModeDocker):
		return state.ModeDocker, nil
	default:
		return "", fmt.Errorf("invalid --to %q; use source or docker", s)
	}
}

// Toggle flips Docker ↔ Source.
func Toggle(m state.Mode) state.Mode {
	if m == state.ModeSource {
		return state.ModeDocker
	}
	return state.ModeSource
}

// ListenReminder tells the developer to bind the IDE process on all interfaces.
func ListenReminder(port int) string {
	return fmt.Sprintf("Source Mode: start the process yourself and listen on 0.0.0.0:%d (not 127.0.0.1). nginx reaches the host via host.docker.internal.", port)
}

// Run rewrites nginx, stops or starts the compose service, and reloads nginx.
func Run(ctx context.Context, opt Options) (Result, error) {
	if opt.Project == nil || opt.Project.Manifest == nil {
		return Result{}, fmt.Errorf("project is required")
	}
	if opt.Docker == nil {
		return Result{}, fmt.Errorf("docker client is required")
	}
	if opt.To != "" && opt.To != state.ModeSource && opt.To != state.ModeDocker {
		return Result{}, fmt.Errorf("unsupported mode %q; use --to source or --to docker", opt.To)
	}

	svc, err := opt.Project.Manifest.LookupService(opt.Name)
	if err != nil {
		return Result{}, err
	}

	overlay := nginxcfg.GeneratedFile(opt.Project.ConfigDir())
	services := nginxcfg.OverlayServices(opt.Project.Manifest)
	if err := nginxcfg.EnsureOverlay(overlay, services); err != nil {
		return Result{}, err
	}

	var from state.Mode
	if opt.To == "" {
		from, err = inferLiveMode(ctx, opt, svc)
		if err != nil {
			return Result{}, err
		}
		opt.To = Toggle(from)
	}

	if err := nginxcfg.WriteOverlay(overlay, services, svc.ComposeService, opt.To); err != nil {
		return Result{}, err
	}

	if opt.To == state.ModeSource {
		if err := stopIfPresent(ctx, opt.Docker, svc.ComposeService); err != nil {
			return Result{}, fmt.Errorf("stop %s: %w", svc.ComposeService, err)
		}
	} else if err := startDockerService(ctx, opt, svc.ComposeService); err != nil {
		return Result{}, fmt.Errorf("start %s: %w", svc.ComposeService, err)
	}

	if _, _, err := nginxcfg.EnsureHostGateway(opt.Project.Root, opt.Project.Manifest.Layout.ComposeFile); err != nil {
		return Result{}, err
	}
	if err := nginxcfg.Reload(ctx, opt.Docker, nginxcfg.DefaultNginxService); err != nil {
		return Result{}, err
	}

	res := Result{Service: svc, From: from, Mode: opt.To}
	if opt.To == state.ModeSource {
		res.Reminder = ListenReminder(svc.SourcePort)
	}
	return res, nil
}

func stopIfPresent(ctx context.Context, c dockerapi.Client, name string) error {
	ok, err := composeServicePresent(ctx, c, name)
	if err != nil || !ok {
		return err
	}
	if err := c.StopService(ctx, name); err != nil && !errors.Is(err, dockerapi.ErrNoContainer) {
		return err
	}
	return nil
}

func startDockerService(ctx context.Context, opt Options, name string) error {
	present, err := composeServicePresent(ctx, opt.Docker, name)
	if err != nil {
		return err
	}
	if present {
		err := opt.Docker.StartService(ctx, name)
		if err == nil {
			return nil
		}
		if !errors.Is(err, dockerapi.ErrNoContainer) {
			return err
		}
	}
	up, err := nginxRunning(ctx, opt.Docker)
	if err != nil || !up {
		return err
	}
	if opt.Compose == nil {
		return fmt.Errorf("no container; compose runner is required to create it")
	}
	return opt.Compose.Up(ctx, opt.Project.Root, opt.Project.Manifest.Layout.ComposeFile, dockerapi.ComposeRunOpts{
		Profiles: opt.Project.Manifest.ComposeProfiles(nil),
		Services: []string{name},
		NoDeps:   true,
	}, nil)
}

func nginxRunning(ctx context.Context, c dockerapi.Client) (bool, error) {
	list, err := c.ListContainers(ctx)
	if err != nil {
		return false, err
	}
	for _, ctr := range list {
		if ctr.ComposeService == nginxcfg.DefaultNginxService && ctr.Running {
			return true, nil
		}
	}
	return false, nil
}

func composeServicePresent(ctx context.Context, c dockerapi.Client, name string) (bool, error) {
	list, err := c.ListContainers(ctx)
	if err != nil {
		return false, err
	}
	for _, ctr := range list {
		if ctr.ComposeService == name {
			return true, nil
		}
	}
	return false, nil
}

func inferLiveMode(ctx context.Context, opt Options, svc manifest.ServiceDef) (state.Mode, error) {
	ports := map[string]int{}
	for _, repo := range opt.Project.Manifest.Repos {
		for _, s := range repo.Services {
			ports[s.ComposeService] = s.SourcePort
		}
	}
	live, err := state.Inferer{
		Containers: dockerapi.Lookup{Client: opt.Docker},
		Nginx: nginxcfg.FilesLookup{
			Path:  nginxcfg.GeneratedFile(opt.Project.ConfigDir()),
			Ports: ports,
		},
	}.Infer(ctx, opt.Project.Manifest)
	if err != nil {
		return "", err
	}
	st, ok := live[svc.ComposeService]
	if !ok || st.Mode == "" {
		return "", fmt.Errorf("cannot infer current mode for %s; pass --to source or --to docker", svc.ComposeService)
	}
	return st.Mode, nil
}
