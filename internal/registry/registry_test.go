package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func sampleEntry(path string) ProjectEntry {
	return ProjectEntry{
		Path:       path,
		GitHostURL: "https://gitlab.example.com",
		GitRemote:  "git@gitlab.example.com:sos/meta.git",
		LastUsed:   time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC),
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "projects.yml")
	if err := os.WriteFile(path, []byte("projects: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load(invalid yaml) = nil, want error")
	}
}

func TestLoadNullProjectsBecomesEmptyMap(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "projects.yml")
	if err := os.WriteFile(path, []byte("projects: null\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if r.Projects == nil {
		t.Fatal("Projects map must be non-nil after load")
	}
}

func TestRegisterInitializesNilMap(t *testing.T) {
	t.Parallel()

	r := &Registry{}
	if _, err := r.Register("isos", sampleEntry("/work/isos")); err != nil {
		t.Fatalf("Register on nil map: %v", err)
	}
}

func TestLoadMissingFileReturnsEmptyRegistry(t *testing.T) {
	t.Parallel()

	r, err := Load(filepath.Join(t.TempDir(), "projects.yml"))
	if err != nil {
		t.Fatalf("Load(missing): %v", err)
	}
	if r == nil || len(r.Projects) != 0 {
		t.Fatalf("expected empty registry, got %#v", r)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "projects.yml")
	r := New()
	r.Projects["isos"] = sampleEntry("/work/isos")

	if err := r.Save(path); err != nil {
		t.Fatalf("Save(): %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	got, ok := loaded.Projects["isos"]
	if !ok {
		t.Fatal("missing isos entry after round-trip")
	}
	if got.Path != "/work/isos" || got.GitRemote != "git@gitlab.example.com:sos/meta.git" {
		t.Fatalf("entry = %#v", got)
	}
}

func TestRegisterNewName(t *testing.T) {
	t.Parallel()

	r := New()
	res, err := r.Register("isos", sampleEntry("/work/isos"))
	if err != nil {
		t.Fatalf("Register(): %v", err)
	}
	if res.Kind != RegisterNew {
		t.Fatalf("Kind = %v, want RegisterNew", res.Kind)
	}
}

func TestRegisterSameProjectUpdatesPath(t *testing.T) {
	t.Parallel()

	r := New()
	_, _ = r.Register("isos", sampleEntry("/old/isos"))

	moved := sampleEntry("/new/isos")
	res, err := r.Register("isos", moved)
	if err != nil {
		t.Fatalf("Register(same project): %v", err)
	}
	if res.Kind != RegisterPathUpdated {
		t.Fatalf("Kind = %v, want RegisterPathUpdated", res.Kind)
	}
	if r.Projects["isos"].Path != "/new/isos" {
		t.Fatalf("path = %q, want /new/isos", r.Projects["isos"].Path)
	}
}

func TestRegisterDifferentProjectIsBlocked(t *testing.T) {
	t.Parallel()

	r := New()
	_, _ = r.Register("isos", sampleEntry("/work/isos"))

	other := sampleEntry("/work/other")
	other.GitRemote = "git@gitlab.other.com:acme/meta.git"
	other.GitHostURL = "https://gitlab.other.com"

	res, err := r.Register("isos", other)
	if err == nil {
		t.Fatal("Register(different project) = nil, want error")
	}
	if res.Kind != RegisterBlocked {
		t.Fatalf("Kind = %v, want RegisterBlocked", res.Kind)
	}
	if r.Projects["isos"].Path != "/work/isos" {
		t.Fatal("blocked register must not overwrite the existing entry")
	}
}

func TestRegisterRejectsEmptyName(t *testing.T) {
	t.Parallel()

	r := New()
	if _, err := r.Register("", sampleEntry("/work/isos")); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestResolveUnknownName(t *testing.T) {
	t.Parallel()

	r := New()
	if _, err := r.Resolve("missing"); err == nil {
		t.Fatal("Resolve(missing) = nil, want error")
	}
}

func TestResolveKnownName(t *testing.T) {
	t.Parallel()

	r := New()
	_, _ = r.Register("isos", sampleEntry("/work/isos"))
	got, err := r.Resolve("isos")
	if err != nil {
		t.Fatalf("Resolve(): %v", err)
	}
	if got.Path != "/work/isos" {
		t.Fatalf("path = %q", got.Path)
	}
}

func TestRelinkUpdatesPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ok := filepath.Join(root, "moved")
	if err := os.Mkdir(ok, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ok, "msc.manifest.yml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New()
	_, _ = r.Register("isos", sampleEntry("/old/isos"))
	if err := r.Relink("isos", ok); err != nil {
		t.Fatal(err)
	}
	if r.Projects["isos"].Path != ok {
		t.Fatalf("path = %q", r.Projects["isos"].Path)
	}
	if err := r.Relink("isos", filepath.Join(root, "missing")); err == nil {
		t.Fatal("relink to missing path")
	}
	if err := r.Relink("nope", ok); err == nil {
		t.Fatal("relink unknown")
	}
}

func TestRemove(t *testing.T) {
	t.Parallel()

	r := New()
	_, _ = r.Register("isos", sampleEntry("/work/isos"))
	if err := r.Remove("isos"); err != nil {
		t.Fatalf("Remove(): %v", err)
	}
	if _, err := r.Resolve("isos"); err == nil {
		t.Fatal("expected missing after Remove")
	}
	if err := r.Remove("isos"); err == nil {
		t.Fatal("Remove(missing) = nil, want error")
	}
}

func TestCheckPathStatuses(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	missing := filepath.Join(root, "gone")
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	okDir := filepath.Join(root, "project")
	if err := os.Mkdir(okDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(okDir, "msc.manifest.yml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	noManifest := filepath.Join(root, "empty")
	if err := os.Mkdir(noManifest, 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		path string
		want PathStatus
	}{
		{missing, PathMissing},
		{file, PathNotDir},
		{noManifest, PathInvalid},
		{okDir, PathOK},
	}
	for _, tc := range cases {
		got := CheckPath(tc.path)
		if got != tc.want {
			t.Fatalf("CheckPath(%s) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestSuggestRepair(t *testing.T) {
	t.Parallel()

	if acts := SuggestRepair(PathOK); len(acts) != 0 {
		t.Fatalf("PathOK suggestions = %v, want none", acts)
	}
	for _, status := range []PathStatus{PathMissing, PathNotDir, PathInvalid} {
		acts := SuggestRepair(status)
		if !containsAction(acts, RepairRelink) || !containsAction(acts, RepairRemove) {
			t.Fatalf("SuggestRepair(%v) = %v, want relink+remove", status, acts)
		}
	}
}

func containsAction(acts []RepairAction, want RepairAction) bool {
	for _, a := range acts {
		if a == want {
			return true
		}
	}
	return false
}
