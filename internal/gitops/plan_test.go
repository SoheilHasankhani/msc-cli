package gitops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
)

func testManifest() *manifest.Manifest {
	m := &manifest.Manifest{
		Brand:       manifest.BrandInfo{DisplayName: "isos", Command: "isos"},
		GitHost:     manifest.GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain: "isos.local",
		Repos: []manifest.RepoDef{
			{Name: "identity-api", Git: "sos/identity-api", Services: []manifest.ServiceDef{
				{ComposeService: "identity.api", Path: ".", SourcePort: 5000},
			}},
			{Name: "doctor-api", Git: "sos/doctor-api", Services: []manifest.ServiceDef{
				{ComposeService: "doctor", Path: ".", SourcePort: 5010},
			}},
			{Name: "wallet-api", Git: "sos/wallet-api", Services: []manifest.ServiceDef{
				{ComposeService: "wallet", Path: ".", SourcePort: 4006},
			}},
		},
	}
	m.ApplyDefaults()
	return m
}

func TestBuildPlanFiltersByAccessAndClone(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	clones := filepath.Join(root, "local")
	if err := os.MkdirAll(filepath.Join(clones, "doctor-api", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	access := map[string]bool{
		"git@gitlab.example.com:sos/identity-api.git": true,
		"git@gitlab.example.com:sos/doctor-api.git":   false, // revoked after clone
		"git@gitlab.example.com:sos/wallet-api.git":   false, // never cloned
	}

	plan, err := BuildPlan(testManifest(), clones, access)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Available) != 1 || plan.Available[0].Name != "identity-api" {
		t.Fatalf("available = %#v", plan.Available)
	}
	if len(plan.Denied) != 1 || plan.Denied[0].Name != "wallet-api" {
		t.Fatalf("denied = %#v", plan.Denied)
	}
	if len(plan.Revoked) != 1 || plan.Revoked[0].Name != "doctor-api" {
		t.Fatalf("revoked = %#v", plan.Revoked)
	}
	if !strings.Contains(plan.Revoked[0].Warning, "never") || !strings.Contains(plan.Revoked[0].Warning, "switch") {
		t.Fatalf("revoked warning = %q", plan.Revoked[0].Warning)
	}
}

func TestBuildPlanOmitsDeniedFromAvailable(t *testing.T) {
	t.Parallel()

	plan, err := BuildPlan(testManifest(), t.TempDir(), map[string]bool{
		"git@gitlab.example.com:sos/identity-api.git": true,
		"git@gitlab.example.com:sos/doctor-api.git":   false,
		"git@gitlab.example.com:sos/wallet-api.git":   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range plan.Available {
		if r.Name == "wallet-api" || r.Name == "doctor-api" {
			t.Fatalf("denied repo leaked into available: %#v", r)
		}
	}
}

func TestSelectRepos(t *testing.T) {
	t.Parallel()

	plan := Plan{Available: []RepoStatus{
		{Name: "identity-api", URL: "u1", Dest: "d1"},
		{Name: "doctor-api", URL: "u2", Dest: "d2"},
	}}
	got, err := plan.Select(nil, true)
	if err != nil || len(got) != 2 {
		t.Fatalf("all = %#v err=%v", got, err)
	}
	got, err = plan.Select([]string{"doctor-api"}, false)
	if err != nil || len(got) != 1 || got[0].Name != "doctor-api" {
		t.Fatalf("named = %#v err=%v", got, err)
	}
	if _, err := plan.Select([]string{"wallet-api"}, false); err == nil {
		t.Fatal("unknown name")
	}
	if _, err := plan.Select(nil, false); err == nil {
		t.Fatal("empty selection")
	}
}

func TestSelectUpdate(t *testing.T) {
	t.Parallel()

	plan := Plan{
		Available: []RepoStatus{{Name: "wallet-api", Accessible: true}},
		Cloned:    []RepoStatus{{Name: "identity-api", Accessible: true}},
	}
	toClone, toPull, err := plan.SelectUpdate(nil)
	if err != nil || len(toClone) != 1 || len(toPull) != 1 {
		t.Fatalf("all = clone %#v pull %#v err=%v", toClone, toPull, err)
	}
	toClone, toPull, err = plan.SelectUpdate([]string{"identity-api"})
	if err != nil || len(toClone) != 0 || len(toPull) != 1 {
		t.Fatalf("named cloned = %#v %#v err=%v", toClone, toPull, err)
	}
	toClone, toPull, err = plan.SelectUpdate([]string{"wallet-api"})
	if err != nil || len(toClone) != 1 || len(toPull) != 0 {
		t.Fatalf("named available = %#v %#v err=%v", toClone, toPull, err)
	}
}

func TestSelectPull(t *testing.T) {
	t.Parallel()

	plan := Plan{
		Cloned: []RepoStatus{
			{Name: "identity-api", Accessible: true, Dest: "d1"},
			{Name: "doctor-api", Accessible: false, Dest: "d2", Warning: "access was revoked"},
		},
	}
	got, err := plan.SelectPull(nil, true)
	if err != nil || len(got) != 1 || got[0].Name != "identity-api" {
		t.Fatalf("all pullable = %#v err=%v", got, err)
	}
	if _, err := plan.SelectPull([]string{"doctor-api"}, false); err == nil {
		t.Fatal("expected revoked repo to fail")
	}
	if _, err := plan.SelectPull([]string{"wallet-api"}, false); err == nil {
		t.Fatal("expected missing clone to fail")
	}
}
