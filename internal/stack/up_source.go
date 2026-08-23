package stack

import (
	"context"
	"fmt"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/progress"
	"golang.org/x/sync/errgroup"
)

// StackUpSource pulls each service image then starts compose, reusing one row per service.
type StackUpSource struct {
	Compose      dockerapi.ComposeRunner
	Root         string
	File         string
	Opts         dockerapi.ComposeRunOpts
	SkipPull     bool
	PullOnly     bool
	SkipServices []string        // compose services already in Source Mode; do not start
	Puller       progress.Puller // optional; defaults to the Docker Engine API
}

// Run implements progress.Source.
func (s StackUpSource) Run(ctx context.Context, updates chan<- progress.Update) error {
	services, err := s.Compose.Services(ctx, s.Root, s.File, s.Opts)
	if err != nil {
		return err
	}

	skip := skipServiceSet(s.SkipServices)
	containerToService := make(map[string]string, len(services))
	imageToServices := map[string][]string{}
	pullWarns := map[string]error{}
	emit := func(u progress.Update) {
		select {
		case updates <- u:
		case <-ctx.Done():
		}
	}

	keep := make([]dockerapi.ComposeService, 0, len(services))
	for _, svc := range services {
		if _, skipped := skip[svc.Name]; skipped {
			emit(progress.Update{ID: svc.Name, Label: svc.Name, Status: "source", Done: true})
			continue
		}
		keep = append(keep, svc)
		containerToService[svc.ContainerName] = svc.Name
		emit(progress.Update{ID: svc.Name, Label: svc.Name})
		if svc.Image != "" {
			imageToServices[svc.Image] = append(imageToServices[svc.Image], svc.Name)
		}
	}
	services = keep

	if !s.SkipPull && len(imageToServices) > 0 {
		puller := s.Puller
		if puller == nil {
			engine, err := dockerapi.NewEngine()
			if err != nil {
				return err
			}
			defer engine.Close()
			puller = engine
		}
		if err := pullServiceImages(ctx, puller, imageToServices, s.PullOnly, pullWarns, emit); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.PullOnly {
		if len(pullWarns) > 0 {
			return fmt.Errorf("%d image(s) failed to pull", len(pullWarns))
		}
		return nil
	}
	if len(services) == 0 {
		return nil
	}

	upOpts := s.Opts
	if len(skip) > 0 {
		upOpts.NoDeps = true
		upOpts.Services = make([]string, 0, len(services))
		for _, svc := range services {
			upOpts.Services = append(upOpts.Services, svc.Name)
		}
	}

	failed := 0
	upErr := s.Compose.Up(ctx, s.Root, s.File, upOpts, func(st dockerapi.ComposeStatus) {
		u := serviceStatusUpdate(st, containerToService, pullWarns)
		if u.Err != nil {
			failed++
		}
		emit(u)
	})
	if upErr != nil {
		return upErr
	}
	if failed > 0 {
		return fmt.Errorf("%d service(s) failed to start", failed)
	}
	return nil
}

func pullServiceImages(ctx context.Context, puller progress.Puller, imageToServices map[string][]string, pullOnly bool, pullWarns map[string]error, emit func(progress.Update)) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(defaultPullParallel)

	for image, serviceNames := range imageToServices {
		image, serviceNames := image, serviceNames
		g.Go(func() error {
			ch := make(chan progress.Update, 16)
			done := make(chan error, 1)
			go func() {
				src := progress.DockerPullSource{
					Ref:       image,
					Puller:    puller,
					FanOutIDs: serviceNames,
				}
				done <- src.Run(ctx, ch)
				close(ch)
			}()
			for u := range ch {
				if u.Err != nil {
					if pullOnly {
						pullWarns[u.ID] = u.Err
					} else {
						u.Warn = u.Err
						u.Err = nil
						pullWarns[u.ID] = u.Warn
					}
				}
				emit(u)
			}
			<-done
			return nil
		})
	}
	return g.Wait()
}

func serviceStatusUpdate(st dockerapi.ComposeStatus, containerToService map[string]string, pullWarns map[string]error) progress.Update {
	id := containerToService[st.Container]
	if id == "" {
		id = st.Container
	}
	u := progress.Update{
		ID:     id,
		Label:  id,
		Status: st.State,
		Done:   dockerapi.IsTerminalComposeStatus(st.State),
	}
	if warn, ok := pullWarns[id]; ok {
		u.Warn = warn
	}
	if isComposeFailure(st.State) {
		u.Err = fmt.Errorf("%s", st.State)
		u.Warn = nil
	}
	return u
}

func isComposeFailure(state string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(state)), "error")
}

func skipServiceSet(names []string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name != "" {
			out[name] = struct{}{}
		}
	}
	return out
}
