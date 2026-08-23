package switchsvc

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/nginxcfg"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/state"
	"github.com/SoheilHasankhani/msc-cli/internal/testenv"
)

type fakeDocker struct {
	list      []dockerapi.Container
	stopped   []string
	started   []string
	signaled  string
	signal    string
	stopErr   error
	startErr  error
	signalErr error
}

func (f *fakeDocker) ListContainers(context.Context) ([]dockerapi.Container, error) {
	return f.list, nil
}
func (f *fakeDocker) StopService(_ context.Context, name string) error {
	f.stopped = append(f.stopped, name)
	return f.stopErr
}
func (f *fakeDocker) StartService(_ context.Context, name string) error {
	f.started = append(f.started, name)
	return f.startErr
}
func (f *fakeDocker) SignalContainer(_ context.Context, name, signal string) error {
	f.signaled = name
	f.signal = signal
	return f.signalErr
}
func (f *fakeDocker) Pull(context.Context, string) (io.ReadCloser, error) { return nil, nil }

type fakeCompose struct {
	upCalled bool
	lastOpts dockerapi.ComposeRunOpts
}

func (f *fakeCompose) Pull(context.Context, string, string, dockerapi.ComposeRunOpts, dockerapi.StatusFn) error {
	return nil
}
func (f *fakeCompose) Up(_ context.Context, _, _ string, opts dockerapi.ComposeRunOpts, _ dockerapi.StatusFn) error {
	f.upCalled = true
	f.lastOpts = opts
	return nil
}
func (f *fakeCompose) Down(context.Context, string, string, dockerapi.ComposeRunOpts, dockerapi.StatusFn) error {
	return nil
}
func (f *fakeCompose) Images(context.Context, string, string, dockerapi.ComposeRunOpts) ([]string, error) {
	return nil, nil
}
func (f *fakeCompose) Services(context.Context, string, string, dockerapi.ComposeRunOpts) ([]dockerapi.ComposeService, error) {
	return nil, nil
}

func testProject(t *testing.T, nginxBody string) *project.Context {
	t.Helper()
	root := t.TempDir()
	compDir := filepath.Join(root, "local", "config", "nginx", "components")
	if err := os.MkdirAll(compDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(compDir, "default.conf"), []byte(nginxBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "local", "docker-compose.yml"), []byte("services:\n  nginx:\n    image: nginx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Brand:       manifest.BrandInfo{DisplayName: "isos", Command: "isos"},
		GitHost:     manifest.GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain: "isos.local",
		Repos: []manifest.RepoDef{{
			Name: "doctor-api",
			Git:  "sos/doctor-api",
			Services: []manifest.ServiceDef{{
				ComposeService: "doctor",
				Path:           ".",
				SourcePort:     5010,
			}},
		}},
	}
	m.ApplyDefaults()
	return &project.Context{Name: "isos", Root: root, Manifest: m}
}

func fixtureNginx(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(testenv.TestdataPath(t, "nginx", "default.conf"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSwitchToSourceStopsRewritesAndReloads(t *testing.T) {
	t.Parallel()

	p := testProject(t, fixtureNginx(t))
	d := &fakeDocker{list: []dockerapi.Container{
		{Name: "/isos-doctor", ComposeService: "doctor", Running: true},
		{Name: "/isos-nginx", ComposeService: "nginx", Running: true},
	}}

	res, err := Run(context.Background(), Options{Project: p, Name: "doctor", To: state.ModeSource, Docker: d})
	if err != nil {
		t.Fatal(err)
	}
	if res.Service.ComposeService != "doctor" || res.Mode != state.ModeSource {
		t.Fatalf("%#v", res)
	}
	if !strings.Contains(res.Reminder, "0.0.0.0") {
		t.Fatalf("missing listen reminder: %q", res.Reminder)
	}
	if len(d.stopped) != 1 || d.stopped[0] != "doctor" {
		t.Fatalf("stopped = %v", d.stopped)
	}
	if len(d.started) != 0 {
		t.Fatalf("started = %v", d.started)
	}
	if d.signaled != "isos-nginx" || d.signal != "HUP" {
		t.Fatalf("reload %q %q", d.signaled, d.signal)
	}

	body, err := os.ReadFile(nginxcfg.GeneratedFile(p.ConfigDir()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "$doctor_upstream") || !strings.Contains(string(body), "host.docker.internal:5010") {
		t.Fatalf("overlay not rewritten:\n%s", body)
	}
	static, err := os.ReadFile(filepath.Join(p.ConfigDir(), "nginx", "components", "default.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(static) != fixtureNginx(t) {
		t.Fatalf("static nginx file was rewritten:\n%s", static)
	}
}

func TestSwitchToDockerStartsContainer(t *testing.T) {
	t.Parallel()

	p := testProject(t, fixtureNginx(t))
	d := &fakeDocker{list: []dockerapi.Container{
		{Name: "/isos-doctor", ComposeService: "doctor", Running: false},
		{Name: "/isos-nginx", ComposeService: "nginx", Running: true},
	}}

	res, err := Run(context.Background(), Options{Project: p, Name: "doctor", To: state.ModeDocker, Docker: d})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != state.ModeDocker {
		t.Fatalf("mode = %s", res.Mode)
	}
	if res.Reminder != "" {
		t.Fatalf("docker mode should not remind: %q", res.Reminder)
	}
	if len(d.started) != 1 || d.started[0] != "doctor" {
		t.Fatalf("started = %v", d.started)
	}
	if len(d.stopped) != 0 {
		t.Fatalf("stopped = %v", d.stopped)
	}
}

func TestSwitchToSourceWhenStackIsDown(t *testing.T) {
	t.Parallel()

	p := testProject(t, fixtureNginx(t))
	d := &fakeDocker{}

	res, err := Run(context.Background(), Options{Project: p, Name: "doctor", To: state.ModeSource, Docker: d})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != state.ModeSource {
		t.Fatalf("mode = %s", res.Mode)
	}
	if len(d.stopped) != 0 {
		t.Fatalf("should not stop a missing container: %v", d.stopped)
	}
	if d.signaled != "" {
		t.Fatalf("should not reload missing nginx: %q", d.signaled)
	}

	body, err := os.ReadFile(nginxcfg.GeneratedFile(p.ConfigDir()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "host.docker.internal:5010") {
		t.Fatalf("overlay not rewritten:\n%s", body)
	}
}

func TestSwitchToDockerWhenStackIsDown(t *testing.T) {
	t.Parallel()

	p := testProject(t, fixtureNginx(t))
	d := &fakeDocker{}
	c := &fakeCompose{}

	res, err := Run(context.Background(), Options{Project: p, Name: "doctor", To: state.ModeDocker, Docker: d, Compose: c})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != state.ModeDocker {
		t.Fatalf("mode = %s", res.Mode)
	}
	if len(d.started) != 0 {
		t.Fatalf("should not start a missing container: %v", d.started)
	}
	if c.upCalled {
		t.Fatal("should not compose-up when the stack is down")
	}
}

func TestSwitchToDockerCreatesContainerWhenStackIsUp(t *testing.T) {
	t.Parallel()

	p := testProject(t, fixtureNginx(t))
	d := &fakeDocker{list: []dockerapi.Container{
		{Name: "/isos-nginx", ComposeService: "nginx", Running: true},
	}}
	c := &fakeCompose{}

	res, err := Run(context.Background(), Options{Project: p, Name: "doctor", To: state.ModeDocker, Docker: d, Compose: c})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != state.ModeDocker {
		t.Fatalf("mode = %s", res.Mode)
	}
	if len(d.started) != 0 {
		t.Fatalf("StartService should not run without a container: %v", d.started)
	}
	if !c.upCalled || !c.lastOpts.NoDeps || len(c.lastOpts.Services) != 1 || c.lastOpts.Services[0] != "doctor" {
		t.Fatalf("compose up = %#v", c.lastOpts)
	}
}

func TestSwitchUnknownService(t *testing.T) {
	t.Parallel()

	p := testProject(t, fixtureNginx(t))
	d := &fakeDocker{list: []dockerapi.Container{
		{Name: "/isos-nginx", ComposeService: "nginx", Running: true},
	}}
	if _, err := Run(context.Background(), Options{Project: p, Name: "wallet", To: state.ModeSource, Docker: d}); err == nil {
		t.Fatal("expected unknown service")
	}
}

func TestSwitchRejectsBadMode(t *testing.T) {
	t.Parallel()

	p := testProject(t, fixtureNginx(t))
	if _, err := Run(context.Background(), Options{Project: p, Name: "doctor", To: "both", Docker: &fakeDocker{}}); err == nil {
		t.Fatal("expected bad mode")
	}
}

func TestSwitchEmptyToTogglesFromLiveNginx(t *testing.T) {
	t.Parallel()

	p := testProject(t, fixtureNginx(t))
	d := &fakeDocker{list: []dockerapi.Container{
		{Name: "/isos-doctor", ComposeService: "doctor", Running: true},
		{Name: "/isos-nginx", ComposeService: "nginx", Running: true},
	}}

	res, err := Run(context.Background(), Options{Project: p, Name: "doctor", Docker: d})
	if err != nil {
		t.Fatal(err)
	}
	if res.From != state.ModeDocker || res.Mode != state.ModeSource {
		t.Fatalf("%#v", res)
	}
	if len(d.stopped) != 1 {
		t.Fatalf("stopped = %v", d.stopped)
	}

	d2 := &fakeDocker{list: []dockerapi.Container{
		{Name: "/isos-doctor", ComposeService: "doctor", Running: false},
		{Name: "/isos-nginx", ComposeService: "nginx", Running: true},
	}}
	res, err = Run(context.Background(), Options{Project: p, Name: "doctor", Docker: d2})
	if err != nil {
		t.Fatal(err)
	}
	if res.From != state.ModeSource || res.Mode != state.ModeDocker {
		t.Fatalf("%#v", res)
	}
	if len(d2.started) != 1 {
		t.Fatalf("started = %v", d2.started)
	}
}

func TestToggle(t *testing.T) {
	t.Parallel()
	if Toggle(state.ModeDocker) != state.ModeSource || Toggle(state.ModeSource) != state.ModeDocker {
		t.Fatal("toggle")
	}
}

func TestParseMode(t *testing.T) {
	t.Parallel()

	got, err := ParseMode("SOURCE")
	if err != nil || got != state.ModeSource {
		t.Fatalf("got %q err=%v", got, err)
	}
	got, err = ParseMode("docker")
	if err != nil || got != state.ModeDocker {
		t.Fatalf("got %q err=%v", got, err)
	}
	if _, err := ParseMode("both"); err == nil {
		t.Fatal("expected invalid mode")
	}
	got, err = ParseMode("")
	if err != nil || got != "" {
		t.Fatalf("empty means toggle, got %q err=%v", got, err)
	}
}

func TestListenReminder(t *testing.T) {
	t.Parallel()

	got := ListenReminder(5010)
	if !strings.Contains(got, "0.0.0.0:5010") || !strings.Contains(got, "127.0.0.1") {
		t.Fatalf("reminder = %q", got)
	}
}
