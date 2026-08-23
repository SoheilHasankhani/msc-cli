package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func validManifest() *Manifest {
	return &Manifest{
		Brand:       BrandInfo{DisplayName: "isos", Command: "isos"},
		GitHost:     GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain: "isos.local",
		Repos: []RepoDef{
			{
				Name: "identity-api",
				Git:  "sos/identity-api",
				Services: []ServiceDef{
					{ComposeService: "identity.api", Path: ".", SourcePort: 5000},
				},
			},
			{
				Name: "consultant-suite",
				Git:  "sos/consultant-suite",
				Services: []ServiceDef{
					{ComposeService: "consultant", Path: "src/Services/ConsultantService/Consultant.API", SourcePort: 5020},
					{ComposeService: "consultant_hub", Path: "src/Services/ConsultantHubService/Consultant.SignalrHub", SourcePort: 5021},
				},
			},
		},
	}
}

func TestValidateNilManifest(t *testing.T) {
	t.Parallel()
	var m *Manifest
	if err := m.Validate(); err == nil {
		t.Fatal("nil manifest must fail Validate")
	}
}

func TestValidateAcceptsValidManifest(t *testing.T) {
	t.Parallel()
	if err := validManifest().Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
}

func TestValidateRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		mut  func(*Manifest)
	}{
		{"empty command", func(m *Manifest) { m.Brand.Command = "" }},
		{"invalid command", func(m *Manifest) { m.Brand.Command = "iso s" }},
		{"empty display name", func(m *Manifest) { m.Brand.DisplayName = "" }},
		{"empty git host url", func(m *Manifest) { m.GitHost.BaseURL = "" }},
		{"invalid git host url", func(m *Manifest) { m.GitHost.BaseURL = "not-a-url" }},
		{"empty local domain", func(m *Manifest) { m.LocalDomain = "" }},
		{"empty repo name", func(m *Manifest) { m.Repos[0].Name = "" }},
		{"empty repo git", func(m *Manifest) { m.Repos[0].Git = "" }},
		{"empty compose service", func(m *Manifest) { m.Repos[0].Services[0].ComposeService = "" }},
		{"empty service path", func(m *Manifest) { m.Repos[0].Services[0].Path = "" }},
		{"zero source port", func(m *Manifest) { m.Repos[0].Services[0].SourcePort = 0 }},
		{"source port too high", func(m *Manifest) { m.Repos[0].Services[0].SourcePort = 70000 }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := validManifest()
			tc.mut(m)
			if err := m.Validate(); err == nil {
				t.Fatal("Validate() = nil, want error")
			}
		})
	}
}

func TestValidateRejectsDuplicateNames(t *testing.T) {
	t.Parallel()

	t.Run("duplicate repo name", func(t *testing.T) {
		t.Parallel()
		m := validManifest()
		m.Repos = append(m.Repos, RepoDef{Name: "identity-api", Git: "other/repo"})
		if err := m.Validate(); err == nil {
			t.Fatal("expected duplicate repo name error")
		}
	})

	t.Run("duplicate compose service", func(t *testing.T) {
		t.Parallel()
		m := validManifest()
		m.Repos[1].Services[0].ComposeService = "identity.api"
		if err := m.Validate(); err == nil {
			t.Fatal("expected duplicate compose_service error")
		}
	})

	t.Run("duplicate source port", func(t *testing.T) {
		t.Parallel()
		m := validManifest()
		m.Repos[1].Services[0].SourcePort = 5000
		if err := m.Validate(); err == nil {
			t.Fatal("expected duplicate source_port error")
		}
	})
}

func TestApplyDefaultsFillsPrerequisites(t *testing.T) {
	t.Parallel()

	m := validManifest()
	m.ApplyDefaults()
	if len(m.Prerequisites) != 3 {
		t.Fatalf("prerequisites = %v", m.Prerequisites)
	}
}

func TestApplyDefaultsMatchesCurrentIsosLayout(t *testing.T) {
	t.Parallel()

	m := validManifest()
	m.ApplyDefaults()
	if m.Layout.ComposeFile != DefaultComposeFile || m.Layout.ConfigDir != DefaultConfigDir || m.Layout.ClonesDir != DefaultClonesDir {
		t.Fatalf("defaults = %#v", m.Layout)
	}
	if m.Layout.ComposeProfile != DefaultComposeProfile {
		t.Fatalf("compose_profile = %q, want %q", m.Layout.ComposeProfile, DefaultComposeProfile)
	}
}

func TestApplyDefaultsDoesNotOverrideExplicitLayout(t *testing.T) {
	t.Parallel()

	m := validManifest()
	m.Layout = Layout{
		ComposeFile:    "compose/docker-compose.yml",
		ComposeProfile: "backend",
		ConfigDir:      "config",
		ClonesDir:      "local",
	}
	m.ApplyDefaults()
	if m.Layout.ComposeFile != "compose/docker-compose.yml" || m.Layout.ConfigDir != "config" {
		t.Fatalf("explicit layout was overwritten: %#v", m.Layout)
	}
	if m.Layout.ComposeProfile != "backend" {
		t.Fatalf("compose_profile = %q, want backend", m.Layout.ComposeProfile)
	}
}

func TestComposeProfilesUsesOverride(t *testing.T) {
	t.Parallel()

	m := validManifest()
	m.Layout.ComposeProfile = "standard"
	if got := m.ComposeProfiles([]string{"all"}); len(got) != 1 || got[0] != "all" {
		t.Fatalf("override = %#v", got)
	}
	if got := m.ComposeProfiles(nil); len(got) != 1 || got[0] != "standard" {
		t.Fatalf("default = %#v", got)
	}
}

func TestLoadRecommendedLayoutFixture(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "testdata", "manifests", "recommended-layout.yaml")
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("fixture failed Validate(): %v", err)
	}
	if m.Layout.ComposeFile != RecommendedComposeFile {
		t.Fatalf("compose_file = %q, want %q", m.Layout.ComposeFile, RecommendedComposeFile)
	}
	if m.Layout.ConfigDir != RecommendedConfigDir {
		t.Fatalf("config_dir = %q, want %q", m.Layout.ConfigDir, RecommendedConfigDir)
	}
	if m.Layout.ClonesDir != DefaultClonesDir {
		t.Fatalf("clones_dir = %q, want %q", m.Layout.ClonesDir, DefaultClonesDir)
	}
}

func TestLoadValidFixture(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "testdata", "manifests", "valid.yaml")
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("fixture failed Validate(): %v", err)
	}
	if m.Brand.Command != "isos" {
		t.Fatalf("command = %q, want isos", m.Brand.Command)
	}
	if len(m.Repos) != 2 {
		t.Fatalf("repos = %d, want 2", len(m.Repos))
	}
	if len(m.Repos[1].Services) != 2 {
		t.Fatalf("consultant-suite services = %d, want 2", len(m.Repos[1].Services))
	}
	if m.Layout.ComposeFile != DefaultComposeFile {
		t.Fatalf("Load() should apply layout defaults, compose_file = %q", m.Layout.ComposeFile)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte("brand: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load(invalid yaml) = nil, want error")
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()

	_, err := Load(filepath.Join(t.TempDir(), "missing.yml"))
	if err == nil {
		t.Fatal("Load(missing) = nil, want error")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	original := validManifest()

	if err := original.Save(path); err != nil {
		t.Fatalf("Save(): %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if err := loaded.Validate(); err != nil {
		t.Fatalf("round-trip Validate(): %v", err)
	}
	if loaded.Brand.Command != original.Brand.Command {
		t.Fatalf("command = %q, want %q", loaded.Brand.Command, original.Brand.Command)
	}
	if loaded.Repos[1].Services[1].SourcePort != 5021 {
		t.Fatalf("source_port = %d, want 5021", loaded.Repos[1].Services[1].SourcePort)
	}
}

func TestFindLooksForCanonicalName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := Find(root); err == nil {
		t.Fatal("Find(empty root) = nil, want error")
	}

	path := filepath.Join(root, FileName)
	if err := os.WriteFile(path, []byte("brand:\n  command: x\n  display_name: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Find(root)
	if err != nil {
		t.Fatalf("Find(): %v", err)
	}
	if got != path {
		t.Fatalf("Find() = %q, want %q", got, path)
	}
}

func TestFindRejectsDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, FileName), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Find(root); err == nil {
		t.Fatal("Find(dir named as manifest) = nil, want error")
	}
}
