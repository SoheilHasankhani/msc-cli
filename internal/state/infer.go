package state

import (
	"context"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
)

// ContainerLookup reports whether a compose service's container is running.
type ContainerLookup interface {
	Running(ctx context.Context, composeService string) (bool, error)
}

// NginxLookup reports where nginx currently sends a service.
type NginxLookup interface {
	Target(composeService string) string
}

// Inferer builds live ServiceState from Docker + nginx observations.
type Inferer struct {
	Containers ContainerLookup
	Nginx      NginxLookup
}

// Infer returns live state for every service in the manifest.
func (in Inferer) Infer(ctx context.Context, m *manifest.Manifest) (map[string]ServiceState, error) {
	out := map[string]ServiceState{}
	if m == nil {
		return out, nil
	}
	for _, repo := range m.Repos {
		for _, svc := range repo.Services {
			up, err := in.Containers.Running(ctx, svc.ComposeService)
			if err != nil {
				return nil, err
			}
			target := NginxTargetContainer
			if in.Nginx != nil {
				if t := in.Nginx.Target(svc.ComposeService); t != "" {
					target = t
				}
			}
			out[svc.ComposeService] = ServiceState{
				ComposeService: svc.ComposeService,
				Mode:           InferMode(up, target),
				ContainerUp:    up,
				NginxTarget:    target,
			}
		}
	}
	return out, nil
}
