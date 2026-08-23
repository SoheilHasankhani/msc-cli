package stack

import (
	"context"
	"io"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/nginxcfg"
	"github.com/SoheilHasankhani/msc-cli/internal/progress"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/state"
)

type fakeCompose struct {
	images     []string
	services   []dockerapi.ComposeService
	pullCalled bool
	upCalled   bool
	downCalled bool
	workDir    string
	file       string
	lastOpts   dockerapi.ComposeRunOpts
	pullStatus []dockerapi.ComposeStatus
	upStatus   []dockerapi.ComposeStatus
}

func (f *fakeCompose) Pull(_ context.Context, workDir, composeFile string, opts dockerapi.ComposeRunOpts, onStatus dockerapi.StatusFn) error {
	f.pullCalled = true
	f.workDir = workDir
	f.file = composeFile
	f.lastOpts = opts
	for _, st := range f.pullStatus {
		if onStatus != nil {
			onStatus(st)
		}
	}
	return nil
}
func (f *fakeCompose) Up(_ context.Context, workDir, composeFile string, opts dockerapi.ComposeRunOpts, onStatus dockerapi.StatusFn) error {
	f.upCalled = true
	f.workDir = workDir
	f.file = composeFile
	f.lastOpts = opts
	for _, st := range f.upStatus {
		if onStatus != nil {
			onStatus(st)
		}
	}
	return nil
}
func (f *fakeCompose) Down(_ context.Context, workDir, composeFile string, opts dockerapi.ComposeRunOpts, onStatus dockerapi.StatusFn) error {
	f.downCalled = true
	f.workDir = workDir
	f.file = composeFile
	f.lastOpts = opts
	return nil
}
func (f *fakeCompose) Images(_ context.Context, _, _ string, _ dockerapi.ComposeRunOpts) ([]string, error) {
	return f.images, nil
}
func (f *fakeCompose) Services(_ context.Context, _, _ string, _ dockerapi.ComposeRunOpts) ([]dockerapi.ComposeService, error) {
	if len(f.services) > 0 {
		return f.services, nil
	}
	return []dockerapi.ComposeService{
		{Name: "patient", ContainerName: "isos-patient", Image: "patient:latest"},
	}, nil
}

func testProject() *project.Context {
	p := &project.Context{
		Root: "/work/isos",
		Manifest: &manifest.Manifest{Layout: manifest.Layout{
			ComposeFile:    "local/docker-compose.yml",
			ComposeProfile: "standard",
		}},
	}
	p.Manifest.ApplyDefaults()
	return p
}

func TestUpRunsComposeUp(t *testing.T) {
	t.Parallel()

	p := testProject()
	fc := &fakeCompose{
		services: []dockerapi.ComposeService{
			{Name: "patient", ContainerName: "isos-patient", Image: "patient:latest"},
		},
		upStatus: []dockerapi.ComposeStatus{
			{Container: "isos-patient", State: "Starting"},
			{Container: "isos-patient", State: "Started"},
		},
	}
	tty := false
	if err := Up(context.Background(), p, fc, progress.Options{Output: io.Discard, IsTTY: &tty}, UpOpts{SkipPull: true}); err != nil {
		t.Fatal(err)
	}
	if !fc.upCalled || fc.workDir != "/work/isos" {
		t.Fatalf("compose up not called correctly: %#v", fc)
	}
}

func TestUpSkipsPull(t *testing.T) {
	t.Parallel()

	p := testProject()
	fc := &fakeCompose{images: []string{"alpine:latest"}}
	tty := false
	if err := Up(context.Background(), p, fc, progress.Options{Output: io.Discard, IsTTY: &tty}, UpOpts{SkipPull: true}); err != nil {
		t.Fatal(err)
	}
	if fc.upCalled == false {
		t.Fatal("compose up not called")
	}
}

func TestUpOmitsSourceModeServices(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	p := &project.Context{
		Root: root,
		Manifest: &manifest.Manifest{
			Layout: manifest.Layout{ComposeFile: "local/docker-compose.yml"},
			Repos: []manifest.RepoDef{{
				Services: []manifest.ServiceDef{{ComposeService: "patient", SourcePort: 4001}},
			}},
		},
	}
	p.Manifest.ApplyDefaults()
	if err := nginxcfg.WriteOverlay(nginxcfg.GeneratedFile(p.ConfigDir()), nginxcfg.OverlayServices(p.Manifest), "patient", state.ModeSource); err != nil {
		t.Fatal(err)
	}

	fc := &fakeCompose{
		services: []dockerapi.ComposeService{
			{Name: "nginx", ContainerName: "isos-nginx", Image: "nginx"},
			{Name: "patient", ContainerName: "isos-patient", Image: "patient:latest"},
		},
		upStatus: []dockerapi.ComposeStatus{
			{Container: "isos-nginx", State: "Started"},
		},
	}
	tty := false
	if err := Up(context.Background(), p, fc, progress.Options{Output: io.Discard, IsTTY: &tty}, UpOpts{SkipPull: true}); err != nil {
		t.Fatal(err)
	}
	if !fc.lastOpts.NoDeps {
		t.Fatal("expected --no-deps so source services are not pulled in")
	}
	if len(fc.lastOpts.Services) != 1 || fc.lastOpts.Services[0] != "nginx" {
		t.Fatalf("services = %#v", fc.lastOpts.Services)
	}
}

func TestUpUsesCLIProfileOverride(t *testing.T) {
	t.Parallel()

	p := testProject()
	fc := &fakeCompose{}
	tty := false
	if err := Up(context.Background(), p, fc, progress.Options{Output: io.Discard, IsTTY: &tty}, UpOpts{
		RunOpts:  RunOpts{Profiles: []string{"all"}},
		SkipPull: true,
	}); err != nil {
		t.Fatal(err)
	}
	if len(fc.lastOpts.Profiles) != 1 || fc.lastOpts.Profiles[0] != "all" {
		t.Fatalf("profiles = %#v", fc.lastOpts.Profiles)
	}
}

func TestDownUsesManifestProfile(t *testing.T) {
	t.Parallel()

	p := testProject()
	fc := &fakeCompose{}
	tty := false
	if err := Down(context.Background(), p, fc, progress.Options{Output: io.Discard, IsTTY: &tty}, DownOpts{}); err != nil {
		t.Fatal(err)
	}
	if !fc.downCalled {
		t.Fatal("down not called")
	}
	if len(fc.lastOpts.Profiles) != 1 || fc.lastOpts.Profiles[0] != "standard" {
		t.Fatalf("profiles = %#v", fc.lastOpts.Profiles)
	}
}

func TestComposeUpSourceEmitsStatus(t *testing.T) {
	t.Parallel()

	fc := &fakeCompose{
		upStatus: []dockerapi.ComposeStatus{
			{Container: "isos-patient", State: "Starting"},
			{Container: "isos-patient", State: "Started"},
		},
	}
	src := ComposeUpSource{Compose: fc, Root: "/work", File: "local/docker-compose.yml"}
	updates := make(chan progress.Update, 4)
	done := make(chan error, 1)
	go func() {
		done <- src.Run(context.Background(), updates)
		close(updates)
	}()
	var got []progress.Update
	for u := range updates {
		got = append(got, u)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Status != "Started" || !got[1].Done {
		t.Fatalf("updates = %#v", got)
	}
}

func TestComposePullSourceEmitsStatus(t *testing.T) {
	t.Parallel()

	fc := &fakeCompose{
		pullStatus: []dockerapi.ComposeStatus{
			{Container: "registry.isos.clinic/isos/wallet:latest", State: "Pulling"},
			{Container: "registry.isos.clinic/isos/wallet:latest", State: "Pulled"},
		},
	}
	src := ComposePullSource{Compose: fc, Root: "/work", File: "local/docker-compose.yml"}
	updates := make(chan progress.Update, 4)
	done := make(chan error, 1)
	go func() {
		done <- src.Run(context.Background(), updates)
		close(updates)
	}()
	var got []progress.Update
	for u := range updates {
		got = append(got, u)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Status != "Pulled" || !got[1].Done {
		t.Fatalf("updates = %#v", got)
	}
	if got[0].Label != "wallet:latest" {
		t.Fatalf("label = %q", got[0].Label)
	}
}
