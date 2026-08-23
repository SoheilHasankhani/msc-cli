package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupportBundleCommandExists(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	cmd, _, err := root.Find([]string{"support-bundle"})
	if err != nil || cmd == nil || cmd.Name() != "support-bundle" {
		t.Fatalf("support-bundle: %v %#v", err, cmd)
	}
}

func TestSupportBundleWritesZip(t *testing.T) {
	logDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(logDir, "msc.jsonl"), []byte(`{"msg":"hi"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MSC_LOG_DIR", logDir)

	dest := filepath.Join(t.TempDir(), "out.zip")
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"support-bundle", "-o", dest})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), dest) {
		t.Fatalf("%s", out.String())
	}
}
