package passthru

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeInjectsManifestFileAndUserArgs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	compose := "local/docker-compose.yml"
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, compose), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	spec, err := Compose(root, compose, []string{"logs", "-f", "doctor"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name != "docker" || spec.Dir != root {
		t.Fatalf("%+v", spec)
	}
	got := strings.Join(spec.Args, " ")
	if !strings.HasPrefix(got, "compose -f local/docker-compose.yml ") || !strings.HasSuffix(got, " logs -f doctor") {
		t.Fatal(got)
	}
}

func TestComposeAddsOverlayWhenPresent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	compose := "local/docker-compose.yml"
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, compose), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "local", "docker-compose.msc.yml"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	spec, err := Compose(root, compose, []string{"ps"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(spec.Args, " ") != "compose -f local/docker-compose.yml -f local/docker-compose.msc.yml ps" {
		t.Fatal(spec.Args)
	}
}

func TestComposeAddsOverlayOnRecommendedLayout(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	compose := "compose/docker-compose.yml"
	if err := os.MkdirAll(filepath.Join(root, "compose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, compose), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "compose", "docker-compose.msc.yml"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	spec, err := Compose(root, compose, []string{"ps"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(spec.Args, " ") != "compose -f compose/docker-compose.yml -f compose/docker-compose.msc.yml ps" {
		t.Fatal(spec.Args)
	}
}

func TestParseGitRequiresDashDash(t *testing.T) {
	t.Parallel()
	if _, _, err := ParseGit([]string{"status"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseGitMetaAndRepo(t *testing.T) {
	t.Parallel()

	repo, args, err := ParseGit([]string{"--", "status", "-sb"})
	if err != nil || repo != "" || strings.Join(args, " ") != "status -sb" {
		t.Fatalf("%q %v %v", repo, args, err)
	}
	repo, args, err = ParseGit([]string{"doctor", "--", "log", "-1"})
	if err != nil || repo != "doctor" || strings.Join(args, " ") != "log -1" {
		t.Fatalf("%q %v %v", repo, args, err)
	}
}

func TestGitResolvesCloneDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	clones := filepath.Join(root, "local")
	dest := filepath.Join(clones, "doctor")
	if err := os.MkdirAll(filepath.Join(dest, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	spec, err := Git(root, clones, []string{"doctor"}, []string{"doctor", "--", "status"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name != "git" || spec.Dir != dest || strings.Join(spec.Args, " ") != "status" {
		t.Fatalf("%+v", spec)
	}

	spec, err = Git(root, clones, []string{"doctor"}, []string{"--", "status"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Dir != root {
		t.Fatal(spec.Dir)
	}
}

func TestGitUnknownOrMissingClone(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := Git(root, filepath.Join(root, "local"), []string{"doctor"}, []string{"nope", "--", "status"})
	if err == nil || !strings.Contains(err.Error(), "unknown repo") {
		t.Fatalf("%v", err)
	}
	_, err = Git(root, filepath.Join(root, "local"), []string{"doctor"}, []string{"doctor", "--", "status"})
	if err == nil || !strings.Contains(err.Error(), "not cloned") {
		t.Fatalf("%v", err)
	}
}
