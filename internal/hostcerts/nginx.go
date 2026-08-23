package hostcerts

import (
	"os"
	"path/filepath"
	"strings"
)

// ReadNginxDir concatenates *.conf files under the nginx components directory.
func ReadNginxDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", err
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String(), nil
}
