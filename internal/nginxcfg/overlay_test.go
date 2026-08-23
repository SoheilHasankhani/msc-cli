package nginxcfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/state"
)

func testServices() []OverlayService {
	return []OverlayService{
		{ComposeService: "doctor", SourcePort: 5010},
		{ComposeService: "identity.api", SourcePort: 2000},
		{ComposeService: "schedule", SourcePort: 3000},
	}
}

func TestUpstreamVarAndDockerHost(t *testing.T) {
	t.Parallel()

	if got := UpstreamVar("identity.api"); got != "identity_api_upstream" {
		t.Fatalf("var = %q", got)
	}
	if got := UpstreamVar("admin_panel"); got != "admin_panel_upstream" {
		t.Fatalf("var = %q", got)
	}
	if got := DockerHost("identity.api"); got != "identity-api" {
		t.Fatalf("host = %q", got)
	}
	if got := DockerUpstream("doctor"); got != "http://doctor" {
		t.Fatalf("docker = %q", got)
	}
	if got := SourceUpstream(7000); got != "http://host.docker.internal:7000" {
		t.Fatalf("source = %q", got)
	}
}

func TestRenderOverlayDockerDefaults(t *testing.T) {
	t.Parallel()

	got := RenderOverlay(testServices(), nil)
	if !strings.Contains(got, "map \"\" $doctor_upstream") || !strings.Contains(got, "default http://doctor;") {
		t.Fatalf("doctor:\n%s", got)
	}
	if !strings.Contains(got, "map \"\" $identity_api_upstream") || !strings.Contains(got, "default http://identity-api;") {
		t.Fatalf("identity:\n%s", got)
	}
}

func TestWriteOverlaySwitchPreservesOthers(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "upstreams.conf")
	if err := WriteOverlay(path, testServices(), "doctor", state.ModeSource); err != nil {
		t.Fatal(err)
	}
	if err := WriteOverlay(path, testServices(), "schedule", state.ModeSource); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if ReadOverlayTarget(body, "doctor", 5010) != state.NginxSourceTarget(5010) {
		t.Fatalf("doctor should stay source:\n%s", body)
	}
	if ReadOverlayTarget(body, "schedule", 3000) != state.NginxSourceTarget(3000) {
		t.Fatalf("schedule:\n%s", body)
	}
	if ReadOverlayTarget(body, "identity.api", 2000) != state.NginxTargetContainer {
		t.Fatalf("identity should stay docker:\n%s", body)
	}

	if err := WriteOverlay(path, testServices(), "doctor", state.ModeDocker); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if ReadOverlayTarget(string(data), "doctor", 5010) != state.NginxTargetContainer {
		t.Fatalf("doctor back to docker:\n%s", data)
	}
	if ReadOverlayTarget(string(data), "schedule", 3000) != state.NginxSourceTarget(3000) {
		t.Fatalf("schedule must be untouched:\n%s", data)
	}
}

func TestWriteOverlayUnknownService(t *testing.T) {
	t.Parallel()

	err := WriteOverlay(filepath.Join(t.TempDir(), "upstreams.conf"), testServices(), "wallet", state.ModeSource)
	if err == nil {
		t.Fatal("expected missing-service error")
	}
}

func TestOverlayServicesFromManifest(t *testing.T) {
	t.Parallel()

	m := &manifest.Manifest{Repos: []manifest.RepoDef{
		{Services: []manifest.ServiceDef{{ComposeService: "wallet", SourcePort: 4006}}},
		{Services: []manifest.ServiceDef{{ComposeService: "doctor", SourcePort: 7000}}},
	}}
	got := OverlayServices(m)
	if len(got) != 2 || got[0].ComposeService != "doctor" || got[1].ComposeService != "wallet" {
		t.Fatalf("%#v", got)
	}
}

func TestFilesLookupMissingFileIsDocker(t *testing.T) {
	t.Parallel()

	lu := FilesLookup{Path: filepath.Join(t.TempDir(), "missing.conf"), Ports: map[string]int{"doctor": 5010}}
	if got := lu.Target("doctor"); got != state.NginxTargetContainer {
		t.Fatalf("got %q", got)
	}
}

func TestFilesLookupReadsGenerated(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "upstreams.conf")
	if err := WriteOverlay(path, testServices(), "schedule", state.ModeSource); err != nil {
		t.Fatal(err)
	}
	lu := FilesLookup{Path: path, Ports: map[string]int{"doctor": 5010, "schedule": 3000}}
	if got := lu.Target("doctor"); got != state.NginxTargetContainer {
		t.Fatalf("doctor = %q", got)
	}
	if got := lu.Target("schedule"); got != state.NginxSourceTarget(3000) {
		t.Fatalf("schedule = %q", got)
	}
}

func TestSourceModeNames(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "upstreams.conf")
	if err := WriteOverlay(path, testServices(), "schedule", state.ModeSource); err != nil {
		t.Fatal(err)
	}
	got := SourceModeNames(path, testServices())
	if len(got) != 1 || got[0] != "schedule" {
		t.Fatalf("got %#v", got)
	}
	if SourceModeNames(filepath.Join(t.TempDir(), "missing.conf"), testServices()) != nil {
		t.Fatal("missing overlay should be empty")
	}
}

func TestGeneratedPaths(t *testing.T) {
	t.Parallel()
	cfg := "/work/isos/local/config"
	if got := ComponentsDir(cfg); got != filepath.Join(cfg, "nginx", "components") {
		t.Fatalf("components = %q", got)
	}
	if got := GeneratedFile(cfg); got != filepath.Join(cfg, "nginx", "generated", "upstreams.conf") {
		t.Fatalf("generated = %q", got)
	}
}
