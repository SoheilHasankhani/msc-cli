package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

const maxArchiveBytes = 128 << 20

// ExtractBinary pulls the msc / msc.exe file out of a GoReleaser tar.gz or zip.
func ExtractBinary(archive []byte, goos string) ([]byte, error) {
	want := "msc"
	if goos == "windows" {
		want = "msc.exe"
	}
	if len(archive) >= 2 && archive[0] == 0x1f && archive[1] == 0x8b {
		return extractTarGz(archive, want)
	}
	if len(archive) >= 2 && archive[0] == 'P' && archive[1] == 'K' {
		return extractZip(archive, want)
	}
	return nil, fmt.Errorf("unknown archive format")
}

func extractTarGz(archive []byte, want string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if !hdr.FileInfo().Mode().IsRegular() {
			continue
		}
		if !isWantedBinary(hdr.Name, want) {
			continue
		}
		return io.ReadAll(io.LimitReader(tr, maxArchiveBytes))
	}
	return nil, fmt.Errorf("archive does not contain %s", want)
}

func extractZip(archive []byte, want string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !isWantedBinary(f.Name, want) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(rc, maxArchiveBytes))
		_ = rc.Close()
		return data, err
	}
	return nil, fmt.Errorf("archive does not contain %s", want)
}

func isWantedBinary(name, want string) bool {
	base := filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	return base == want
}
