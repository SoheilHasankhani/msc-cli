package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDoctorHelpMentionsFixAndNoInstall(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"doctor", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "--fix") || !strings.Contains(got, "Docker") || !strings.Contains(got, "--no-elevate") {
		t.Fatalf("help:\n%s", got)
	}
}

func TestDoctorRunsWithoutProject(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"doctor"})
	err := cmd.Execute()
	// Real machine may pass or fail; the command must run and print the table.
	if !strings.Contains(out.String(), "CHECK") {
		t.Fatalf("err=%v out=\n%s", err, out.String())
	}
}
