package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestExtractBinaryFromTarGz(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, f := range []struct {
		name, body string
	}{
		{"README.md", "docs"},
		{"msc", "linux-binary"},
	} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Size: int64(len(f.body)), Mode: 0o755}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(f.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := ExtractBinary(buf.Bytes(), "linux")
	if err != nil || string(got) != "linux-binary" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestExtractBinaryFromZip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("msc.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("win-binary")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := ExtractBinary(buf.Bytes(), "windows")
	if err != nil || string(got) != "win-binary" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestExtractBinaryUnknownFormat(t *testing.T) {
	t.Parallel()
	if _, err := ExtractBinary([]byte("not-an-archive"), "linux"); err == nil {
		t.Fatal("expected unknown format")
	}
}
