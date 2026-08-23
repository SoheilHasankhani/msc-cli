package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/project"
)

func TestBrandModeHidesProjectFlag(t *testing.T) {
	t.Setenv(project.EnvVar, "isos")
	cmd := NewRootCmd()
	flag := cmd.PersistentFlags().Lookup("project")
	if flag == nil || !flag.Hidden {
		t.Fatalf("expected hidden project flag in brand mode, got %#v", flag)
	}
	if cmd.Use != "isos" {
		t.Fatalf("Use = %q", cmd.Use)
	}
	val, _ := cmd.PersistentFlags().GetString("project")
	if val != "isos" {
		t.Fatalf("project = %q", val)
	}
}

func TestBrandModeHelpUsesBrandName(t *testing.T) {
	t.Setenv(project.EnvVar, "isos")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "isos") {
		t.Fatalf("help should use brand name:\n%s", got)
	}
	if strings.Contains(got, "--project") {
		t.Fatalf("help should not show --project in brand mode:\n%s", got)
	}
}

func TestBrandModeHidesEngineCommands(t *testing.T) {
	t.Setenv(project.EnvVar, "isos")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, name := range engineOnlyCommands {
		if strings.Contains(got, "\n  "+name+" ") {
			t.Fatalf("brand help should not list engine command %q:\n%s", name, got)
		}
	}
	for _, name := range []string{"status", "sync", "doctor", "compose", "git"} {
		if !strings.Contains(got, name) {
			t.Fatalf("brand help missing %q:\n%s", name, got)
		}
	}
}

func TestBrandModeRejectsEngineCommand(t *testing.T) {
	t.Setenv(project.EnvVar, "isos")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"projects", "list"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for isos projects")
	}
	if !strings.Contains(err.Error(), "engine command") {
		t.Fatalf("err = %v", err)
	}
}

func TestBrandModeStatusWorks(t *testing.T) {
	t.Setenv(project.EnvVar, "isos")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"status"})
	err := cmd.Execute()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), project.MissingMessage()) {
		t.Fatalf("brand mode should select project from env: %v", err)
	}
}
