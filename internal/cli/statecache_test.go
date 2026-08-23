package cli

import (
	"context"
	"io"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/nginxcfg"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/state"
)

type cacheDocker struct {
	list []dockerapi.Container
}

func (c cacheDocker) ListContainers(context.Context) ([]dockerapi.Container, error) {
	return c.list, nil
}
func (c cacheDocker) StopService(context.Context, string) error             { return nil }
func (c cacheDocker) StartService(context.Context, string) error            { return nil }
func (c cacheDocker) SignalContainer(context.Context, string, string) error { return nil }
func (c cacheDocker) Pull(context.Context, string) (io.ReadCloser, error)   { return nil, nil }

func TestSaveLiveCacheRecordsContainerUp(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	live := map[string]state.ServiceState{
		"wallet": {ComposeService: "wallet", Mode: state.ModeDocker, ContainerUp: true, NginxTarget: state.NginxTargetContainer},
	}
	if err := saveLiveCache("isos", live, ""); err != nil {
		t.Fatal(err)
	}
	snap, err := state.LoadCache(state.CachePath(paths.Default().StateDir(), "isos"))
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Services["wallet"].ContainerUp {
		t.Fatalf("cache = %#v", snap.Services["wallet"])
	}
}

func TestInferLiveUsesOverlayAndContainers(t *testing.T) {
	t.Parallel()

	p := &project.Context{
		Root: t.TempDir(),
		Manifest: &manifest.Manifest{
			Repos: []manifest.RepoDef{{
				Services: []manifest.ServiceDef{
					{ComposeService: "wallet", SourcePort: 4006},
					{ComposeService: "patient", SourcePort: 9000},
				},
			}},
		},
	}
	p.Manifest.ApplyDefaults()
	if err := nginxcfg.WriteOverlay(nginxcfg.GeneratedFile(p.ConfigDir()), nginxcfg.OverlayServices(p.Manifest), "patient", state.ModeSource); err != nil {
		t.Fatal(err)
	}

	live, err := inferLive(context.Background(), p, cacheDocker{list: []dockerapi.Container{
		{ComposeService: "wallet", Running: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !live["wallet"].ContainerUp || live["wallet"].Mode != state.ModeDocker {
		t.Fatalf("wallet = %#v", live["wallet"])
	}
	if live["patient"].ContainerUp || live["patient"].Mode != state.ModeSource {
		t.Fatalf("patient = %#v", live["patient"])
	}
}
