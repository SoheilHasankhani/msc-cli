package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/testenv"
)

func TestGitHelpFlagShowsUsageGuide(t *testing.T) {
	t.Setenv(project.EnvVar, "isos")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"git", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"isos git",
		"Run git in the meta-repo",
		"Usage:",
		"Examples:",
		"Notes:",
		"isos git -- status -sb",
		"isos git identity-api -- log",
		"isos git doctor -- --help",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "flags are not eaten by msc") {
		t.Fatalf("should show structured help:\n%s", got)
	}
}

func TestGitHelpBeforeRepoName(t *testing.T) {
	t.Setenv(project.EnvVar, "isos")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"git", "doctor", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "isos git <repo> -- <git args>") {
		t.Fatalf("expected usage guide, got:\n%s", got)
	}
}

func TestGitMissingDashDashShowsExamples(t *testing.T) {
	testenv.InstallBrandProject(t, "isos")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"git", "status"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `git -- status -sb`) {
		t.Fatalf("err = %v", err)
	}
}

func TestDoctorHelpDoesNotDuplicateLong(t *testing.T) {
	t.Setenv(project.EnvVar, "isos")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"doctor", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Usage:\n  isos doctor") {
		t.Fatalf("expected doctor usage:\n%s", got)
	}
	long := "Hosts-file and OS trust-store writes re-invoke this binary under sudo / UAC."
	if strings.Count(got, long) != 1 {
		t.Fatalf("expected Long once, got %d:\n%s", strings.Count(got, long), got)
	}
	if strings.Contains(got, "Brand-agnostic engine") {
		t.Fatalf("root banner should not appear on subcommand help:\n%s", got)
	}
}
