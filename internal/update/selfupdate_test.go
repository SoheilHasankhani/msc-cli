package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

type fakeRel struct {
	rel Release
	err error
}

func (f fakeRel) Latest(context.Context) (Release, error) { return f.rel, f.err }

func sampleRelease() Release {
	return Release{
		Tag: "v1.2.3",
		Assets: []Asset{
			{Name: "checksums.txt", URL: "https://example/checksums.txt"},
			{Name: "msc_1.2.3_linux_amd64.tar.gz", URL: "https://example/linux.tgz"},
		},
	}
}

func TestRunAlreadyLatest(t *testing.T) {
	t.Parallel()

	res, err := Run(context.Background(), Options{
		Current: "1.2.3",
		GOOS:    "linux",
		GOARCH:  "amd64",
		Client:  fakeRel{rel: sampleRelease()},
		Fetch:   func(context.Context, string) ([]byte, error) { t.Fatal("fetch"); return nil, nil },
	})
	if err != nil || !res.Skipped || res.Updated {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestRunCheckOnlyDoesNotDownload(t *testing.T) {
	t.Parallel()

	res, err := Run(context.Background(), Options{
		Current:   "1.0.0",
		GOOS:      "linux",
		GOARCH:    "amd64",
		CheckOnly: true,
		Client:    fakeRel{rel: sampleRelease()},
		Fetch:     func(context.Context, string) ([]byte, error) { t.Fatal("fetch"); return nil, nil },
	})
	if err != nil || res.Updated || res.Latest != "1.2.3" {
		t.Fatalf("%+v %v", res, err)
	}
	if res.Message == "" {
		t.Fatal("expected message")
	}
}

func TestRunForceReinstalls(t *testing.T) {
	t.Parallel()

	archive := mustTarGz(t, "forced")
	sum := sha256.Sum256(archive)
	checksums := hex.EncodeToString(sum[:]) + "  msc_1.2.3_linux_amd64.tar.gz\n"
	dest := filepath.Join(t.TempDir(), "msc")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Run(context.Background(), Options{
		Current: "1.2.3",
		Dest:    dest,
		GOOS:    "linux",
		GOARCH:  "amd64",
		Force:   true,
		Client:  fakeRel{rel: sampleRelease()},
		Fetch: func(_ context.Context, url string) ([]byte, error) {
			if url == "https://example/checksums.txt" {
				return []byte(checksums), nil
			}
			return archive, nil
		},
	})
	if err != nil || !res.Updated {
		t.Fatalf("%+v %v", res, err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "forced" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestRunUpdatesOlderBinary(t *testing.T) {
	t.Parallel()

	archive := mustTarGz(t, "fresh")
	sum := sha256.Sum256(archive)
	checksums := hex.EncodeToString(sum[:]) + "  msc_1.2.3_linux_amd64.tar.gz\n"
	dest := filepath.Join(t.TempDir(), "msc")

	res, err := Run(context.Background(), Options{
		Current: "dev",
		Dest:    dest,
		GOOS:    "linux",
		GOARCH:  "amd64",
		Client:  fakeRel{rel: sampleRelease()},
		Fetch: func(_ context.Context, url string) ([]byte, error) {
			if url == "https://example/checksums.txt" {
				return []byte(checksums), nil
			}
			return archive, nil
		},
	})
	if err != nil || !res.Updated || res.Latest != "1.2.3" {
		t.Fatalf("%+v %v", res, err)
	}
}
