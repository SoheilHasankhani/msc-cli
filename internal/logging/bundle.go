package logging

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteBundle zips JSON log files from logDir plus a small meta.json.
// dest may be a .zip path or a directory; an empty dest uses the working directory.
func WriteBundle(logDir, dest string, now time.Time, meta map[string]string) (string, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	out, err := resolveDest(dest, now)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	zw := zip.NewWriter(f)
	ok := false
	defer func() {
		_ = zw.Close()
		_ = f.Close()
		if !ok {
			_ = os.Remove(out)
		}
	}()

	if meta == nil {
		meta = map[string]string{}
	}
	if meta["created"] == "" {
		meta["created"] = now.UTC().Format(time.RFC3339)
	}
	body, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", err
	}
	if err := writeZipFile(zw, "meta.json", body); err != nil {
		return "", err
	}

	entries, err := os.ReadDir(logDir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() || !isLogName(e.Name()) {
			continue
		}
		path := filepath.Join(logDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		if err := writeZipFile(zw, "logs/"+e.Name(), data); err != nil {
			return "", err
		}
	}
	ok = true
	return out, nil
}

func resolveDest(dest string, now time.Time) (string, error) {
	name := fmt.Sprintf("msc-support-%s.zip", now.UTC().Format("20060102-150405"))
	if dest == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(wd, name), nil
	}
	info, err := os.Stat(dest)
	if err == nil && info.IsDir() {
		return filepath.Join(dest, name), nil
	}
	if strings.HasSuffix(strings.ToLower(dest), ".zip") {
		return dest, nil
	}
	return filepath.Join(dest, name), nil
}

func isLogName(name string) bool {
	base := strings.ToLower(name)
	return strings.Contains(base, ".jsonl")
}

func writeZipFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
