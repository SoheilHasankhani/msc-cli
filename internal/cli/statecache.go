package cli

import (
	"context"
	"time"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/nginxcfg"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/state"
)

func inferLive(ctx context.Context, p *project.Context, c dockerapi.Client) (map[string]state.ServiceState, error) {
	ports := map[string]int{}
	if p != nil && p.Manifest != nil {
		for _, repo := range p.Manifest.Repos {
			for _, svc := range repo.Services {
				ports[svc.ComposeService] = svc.SourcePort
			}
		}
	}
	return state.Inferer{
		Containers: dockerapi.Lookup{Client: c},
		Nginx: nginxcfg.FilesLookup{
			Path:  nginxcfg.GeneratedFile(p.ConfigDir()),
			Ports: ports,
		},
	}.Infer(ctx, p.Manifest)
}

func syncProjectCache(ctx context.Context, p *project.Context, c dockerapi.Client, switched string) error {
	live, err := inferLive(ctx, p, c)
	if err != nil {
		return err
	}
	return saveLiveCache(p.Name, live, switched)
}

func refreshProjectCache(ctx context.Context, p *project.Context) error {
	engine, err := dockerapi.NewEngine()
	if err != nil {
		return err
	}
	defer engine.Close()
	return syncProjectCache(ctx, p, engine, "")
}

func saveLiveCache(projectName string, live map[string]state.ServiceState, switched string) error {
	cacheFile := state.CachePath(paths.Default().StateDir(), projectName)
	snap, err := state.LoadCache(cacheFile)
	if err != nil {
		return err
	}
	next := snap.SyncLive(live)
	if switched != "" {
		if s, ok := next.Services[switched]; ok {
			now := time.Now()
			s.LastSwitchedAt = &now
			next.Services[switched] = s
		}
	}
	return state.SaveCache(cacheFile, next)
}
