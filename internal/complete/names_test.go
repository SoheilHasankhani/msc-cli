package complete

import (
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/state"
)

func TestServicesPrefersCacheThenManifest(t *testing.T) {
	t.Parallel()

	m := &manifest.Manifest{Repos: []manifest.RepoDef{
		{Name: "doctor", Services: []manifest.ServiceDef{{ComposeService: "doctor"}}},
		{Name: "wallet", Services: []manifest.ServiceDef{{ComposeService: "wallet"}}},
	}}
	cache := &state.Snapshot{Services: map[string]state.ServiceState{
		"doctor": {ComposeService: "doctor"},
		"extra":  {ComposeService: "extra"},
	}}
	got := Services(m, cache)
	if strings.Join(got, ",") != "doctor,extra,wallet" {
		t.Fatal(got)
	}
}

func TestReposFromManifest(t *testing.T) {
	t.Parallel()
	m := &manifest.Manifest{Repos: []manifest.RepoDef{{Name: "identity-api"}, {Name: "doctor"}}}
	got := Repos(m)
	if strings.Join(got, ",") != "doctor,identity-api" {
		t.Fatal(got)
	}
}

func TestFilterPrefix(t *testing.T) {
	t.Parallel()
	got := FilterPrefix([]string{"doctor", "wallet", "identity.api"}, "do")
	if strings.Join(got, ",") != "doctor" {
		t.Fatal(got)
	}
}
