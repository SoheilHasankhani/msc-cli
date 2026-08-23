package logging

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteBundleZipsRecentLogs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "msc.jsonl"), []byte(`{"msg":"a"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "msc.jsonl.1"), []byte(`{"msg":"old"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "bundle.zip")
	got, err := WriteBundle(dir, dest, time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC), map[string]string{
		"version": "dev",
		"os":      "linux",
	})
	if err != nil || got != dest {
		t.Fatalf("%q %v", got, err)
	}

	zr, err := zip.OpenReader(dest)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = zr.Close() })

	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	if !names["logs/msc.jsonl"] || !names["logs/msc.jsonl.1"] || !names["meta.json"] {
		t.Fatalf("%v", names)
	}
	if names["logs/notes.txt"] {
		t.Fatal("non-log files must not be bundled")
	}
}

func TestWriteBundleDefaultName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outDir := t.TempDir()
	got, err := WriteBundle(dir, outDir, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "msc-support-20260102-030405.zip") {
		t.Fatalf("%s", got)
	}
}

func TestWriteBundleEmptyLogDir(t *testing.T) {
	t.Parallel()

	dest := filepath.Join(t.TempDir(), "empty.zip")
	if _, err := WriteBundle(t.TempDir(), dest, time.Now(), map[string]string{"version": "dev"}); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(dest)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = zr.Close() })
	if len(zr.File) != 1 || zr.File[0].Name != "meta.json" {
		t.Fatalf("%d files", len(zr.File))
	}
}
