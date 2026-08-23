package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDownloadsVerifiesExtractsReplaces(t *testing.T) {
	t.Parallel()

	archive := mustTarGz(t, "new-engine")
	sum := sha256.Sum256(archive)
	hexSum := hex.EncodeToString(sum[:])
	asset := "msc_1.2.3_linux_amd64.tar.gz"
	checksums := hexSum + "  " + asset + "\n"

	dest := filepath.Join(t.TempDir(), "msc")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	fetch := func(_ context.Context, url string) ([]byte, error) {
		switch url {
		case "https://example/linux.tgz":
			return archive, nil
		case "https://example/checksums.txt":
			return []byte(checksums), nil
		default:
			t.Fatalf("unexpected url %s", url)
			return nil, nil
		}
	}
	p := Plan{
		AssetName:    asset,
		AssetURL:     "https://example/linux.tgz",
		ChecksumsURL: "https://example/checksums.txt",
	}
	if err := Apply(context.Background(), p, dest, "linux", fetch); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-engine" {
		t.Fatalf("%q", got)
	}
}

func TestApplyRejectsBadChecksum(t *testing.T) {
	t.Parallel()

	archive := mustTarGz(t, "payload")
	p := Plan{
		AssetName:    "msc_1.0.0_linux_amd64.tar.gz",
		AssetURL:     "a",
		ChecksumsURL: "c",
	}
	fetch := func(_ context.Context, url string) ([]byte, error) {
		if url == "c" {
			return []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  msc_1.0.0_linux_amd64.tar.gz\n"), nil
		}
		return archive, nil
	}
	err := Apply(context.Background(), p, filepath.Join(t.TempDir(), "msc"), "linux", fetch)
	if err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func mustTarGz(t *testing.T, body string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "msc", Size: int64(len(body)), Mode: 0o755}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
