package manifest

import (
	"strings"
	"testing"
)

const sampleCompose = `
services:
  nginx:
    image: nginx
  redis:
    image: redis
  doctor:
    image: doctor
  identity.api:
    image: identity
  wallet:
    image: wallet
`

func TestSuggestMapsCloneDirsToComposeServices(t *testing.T) {
	t.Parallel()

	got := Suggest(SuggestInput{
		DisplayName:  "isos",
		Command:      "isos",
		GitHostBase:  "https://gitlab.example.com",
		LocalDomain:  "isos.local",
		ComposeYAML:  sampleCompose,
		CloneDirs:    []string{"doctor", "identity-api", "wallet", "config"},
		DefaultGroup: "sos",
	})
	if err := got.Validate(); err != nil {
		t.Fatalf("suggested manifest invalid: %v", err)
	}
	if got.Brand.Command != "isos" || got.LocalDomain != "isos.local" {
		t.Fatalf("%#v", got.Brand)
	}
	by := map[string]RepoDef{}
	for _, r := range got.Repos {
		by[r.Name] = r
	}
	if _, ok := by["config"]; ok {
		t.Fatal("config dir must not become a repo")
	}
	if by["doctor"].Services[0].ComposeService != "doctor" {
		t.Fatalf("doctor = %#v", by["doctor"])
	}
	if by["identity-api"].Services[0].ComposeService != "identity.api" {
		t.Fatalf("identity-api = %#v", by["identity-api"])
	}
	if by["wallet"].Git != "sos/wallet" {
		t.Fatalf("git = %q", by["wallet"].Git)
	}
}

func TestSuggestSkipsInfrastructureServices(t *testing.T) {
	t.Parallel()

	got := Suggest(SuggestInput{
		DisplayName:  "x",
		Command:      "x",
		GitHostBase:  "https://gitlab.example.com",
		LocalDomain:  "x.local",
		ComposeYAML:  sampleCompose,
		CloneDirs:    nil,
		DefaultGroup: "acme",
	})
	for _, r := range got.Repos {
		if r.Name == "nginx" || r.Name == "redis" {
			t.Fatalf("infra leaked: %#v", r)
		}
	}
}

func TestCommitReminder(t *testing.T) {
	t.Parallel()

	got := CommitReminder("msc.manifest.yml")
	if !strings.Contains(got, "git add") || !strings.Contains(got, "msc.manifest.yml") || !strings.Contains(got, "commit") {
		t.Fatalf("%s", got)
	}
}
