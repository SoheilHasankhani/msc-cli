package syncsvc

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SoheilHasankhani/msc-cli/internal/gitops"
	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/progress"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
)

type fakeGit struct {
	mu       sync.Mutex
	access   map[string]bool
	cloned   []string
	pulled   []string
	lsErr    error
	cloneErr error
	pullErr  map[string]error
}

func (f *fakeGit) LsRemote(_ context.Context, url string) (bool, error) {
	if f.lsErr != nil {
		return false, f.lsErr
	}
	return f.access[url], nil
}
func (f *fakeGit) Clone(_ context.Context, url, dest string, _ io.Writer) error {
	if f.cloneErr != nil {
		return f.cloneErr
	}
	f.mu.Lock()
	f.cloned = append(f.cloned, url)
	f.mu.Unlock()
	if err := os.MkdirAll(filepath.Join(dest, ".git"), 0o755); err != nil {
		return err
	}
	return nil
}
func (f *fakeGit) Pull(_ context.Context, dest string, _ io.Writer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.pullErr[dest]; ok {
		return err
	}
	f.pulled = append(f.pulled, dest)
	return nil
}

func syncProject(t *testing.T) *project.Context {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
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
		},
	}
	m.ApplyDefaults()
	return &project.Context{Name: "isos", Root: root, Manifest: m}
}

func TestPlanUsesCacheWhenFresh(t *testing.T) {
	t.Parallel()

	p := syncProject(t)
	cachePath := filepath.Join(t.TempDir(), "access.json")
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	cached := &gitops.AccessCache{
		CheckedAt: now.Add(-time.Hour),
		Repos: map[string]bool{
			"git@gitlab.example.com:sos/identity-api.git": true,
			"git@gitlab.example.com:sos/doctor-api.git":   false,
		},
	}
	if err := gitops.SaveAccessCache(cachePath, cached); err != nil {
		t.Fatal(err)
	}

	g := &fakeGit{lsErr: io.ErrUnexpectedEOF} // must not be called
	plan, err := Plan(context.Background(), Options{
		Project:   p,
		Git:       g,
		CachePath: cachePath,
		Now:       now,
		Refresh:   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Available) != 1 || plan.Available[0].Name != "identity-api" {
		t.Fatalf("%#v", plan.Available)
	}
}

func TestPlanRefreshProbesAndSaves(t *testing.T) {
	t.Parallel()

	p := syncProject(t)
	cachePath := filepath.Join(t.TempDir(), "access.json")
	g := &fakeGit{access: map[string]bool{
		"git@gitlab.example.com:sos/identity-api.git": true,
		"git@gitlab.example.com:sos/doctor-api.git":   true,
	}}
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	plan, err := Plan(context.Background(), Options{
		Project:   p,
		Git:       g,
		CachePath: cachePath,
		Now:       now,
		Refresh:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Available) != 2 {
		t.Fatalf("available = %#v", plan.Available)
	}
	saved, err := gitops.LoadAccessCache(cachePath)
	if err != nil || !saved.Repos["git@gitlab.example.com:sos/identity-api.git"] {
		t.Fatalf("cache not saved: %#v %v", saved, err)
	}
}

func TestCloneAllRunsInParallelViaProgress(t *testing.T) {
	t.Parallel()

	p := syncProject(t)
	g := &fakeGit{access: map[string]bool{
		"git@gitlab.example.com:sos/identity-api.git": true,
		"git@gitlab.example.com:sos/doctor-api.git":   true,
	}}
	tty := false
	n, err := Clone(context.Background(), Options{
		Project:  p,
		Git:      g,
		Now:      time.Now(),
		Refresh:  true,
		All:      true,
		Progress: progress.Options{Output: io.Discard, IsTTY: &tty},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || len(g.cloned) != 2 {
		t.Fatalf("cloned %d urls=%v", n, g.cloned)
	}
}

func TestFormatPlanListsAvailableAndRevoked(t *testing.T) {
	t.Parallel()

	plan := gitops.Plan{
		Available: []gitops.RepoStatus{{Name: "identity-api"}},
		Cloned:    []gitops.RepoStatus{{Name: "wallet-api", Accessible: true}},
		Revoked:   []gitops.RepoStatus{{Name: "doctor-api", Warning: "access was revoked; the local clone is kept (never deleted)"}},
	}
	got := FormatPlan(plan, ui.Render{})
	if !strings.Contains(got, "identity-api") {
		t.Fatalf("missing available:\n%s", got)
	}
	if !strings.Contains(got, "wallet-api") || !strings.Contains(got, "available to clone") {
		t.Fatalf("missing cloned/list sections:\n%s", got)
	}
	if !strings.Contains(got, "warning:") || !strings.Contains(got, "doctor-api") {
		t.Fatalf("missing revoked warning:\n%s", got)
	}
	if !strings.Contains(got, "--list") {
		t.Fatalf("missing --list hint:\n%s", got)
	}
}

func TestUpdateClonesAndPulls(t *testing.T) {
	t.Parallel()

	p := syncProject(t)
	dest := filepath.Join(p.ClonesDir(), "identity-api")
	if err := os.MkdirAll(filepath.Join(dest, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	g := &fakeGit{access: map[string]bool{
		"git@gitlab.example.com:sos/identity-api.git": true,
		"git@gitlab.example.com:sos/doctor-api.git":   true,
	}}
	tty := false
	_, res, err := Update(context.Background(), Options{
		Project:  p,
		Git:      g,
		Now:      time.Now(),
		Refresh:  true,
		Progress: progress.Options{Output: io.Discard, IsTTY: &tty},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Cloned != 1 || res.Pulled != 1 {
		t.Fatalf("res = %#v cloned=%v pulled=%v", res, g.cloned, g.pulled)
	}
}

func TestUpdateRefreshDoesNotDuplicateProbeProgress(t *testing.T) {
	t.Parallel()

	p := syncProject(t)
	for _, name := range []string{"identity-api", "doctor-api"} {
		dest := filepath.Join(p.ClonesDir(), name)
		if err := os.MkdirAll(filepath.Join(dest, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	g := &fakeGit{access: map[string]bool{
		"git@gitlab.example.com:sos/identity-api.git": true,
		"git@gitlab.example.com:sos/doctor-api.git":   true,
	}}
	var buf bytes.Buffer
	tty := false
	_, _, err := Update(context.Background(), Options{
		Project:  p,
		Git:      g,
		Now:      time.Now(),
		Refresh:  true,
		Progress: progress.Options{Output: &buf, IsTTY: &tty},
	})
	if err != nil {
		t.Fatal(err)
	}
	doneCount := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if !strings.HasSuffix(line, ": done") {
			continue
		}
		name := strings.TrimSuffix(line, ": done")
		doneCount[name]++
	}
	if len(doneCount) != 2 {
		t.Fatalf("expected done line per repo, got %v:\n%s", doneCount, buf.String())
	}
	for name, n := range doneCount {
		if n != 1 {
			t.Fatalf("repo %q appeared %d times in done output:\n%s", name, n, buf.String())
		}
	}
}

func TestPullContinuesOnConflict(t *testing.T) {
	t.Parallel()

	p := syncProject(t)
	dest := filepath.Join(p.ClonesDir(), "identity-api")
	if err := os.MkdirAll(filepath.Join(dest, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	conflictDest := filepath.Join(p.ClonesDir(), "doctor-api")
	if err := os.MkdirAll(filepath.Join(conflictDest, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	g := &fakeGit{
		access: map[string]bool{
			"git@gitlab.example.com:sos/identity-api.git": true,
			"git@gitlab.example.com:sos/doctor-api.git":   true,
		},
		pullErr: map[string]error{
			conflictDest: gitops.WrapPullConflict("fatal: Not possible to fast-forward, aborting.\n"),
		},
	}
	tty := false
	res, err := Pull(context.Background(), Options{
		Project:  p,
		Git:      g,
		Now:      time.Now(),
		Refresh:  true,
		All:      true,
		Progress: progress.Options{Output: io.Discard, IsTTY: &tty},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pulled != 1 || len(res.Warnings) != 1 || res.Warnings[0].Name != "doctor-api" {
		t.Fatalf("res = %#v pulled=%v", res, g.pulled)
	}
}

func TestFormatPullResult(t *testing.T) {
	t.Parallel()

	got := FormatPullResult(PullResult{
		Pulled:   2,
		Warnings: []PullWarning{{Name: "doctor-api", Message: "cannot fast-forward"}},
	}, ui.Render{})
	if !strings.Contains(got, "warning: doctor-api") || !strings.Contains(got, "pulled 2 repo(s)") {
		t.Fatalf("%s", got)
	}
}
