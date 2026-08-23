package testenv

import (
	"os"
	"path/filepath"
	"testing"
)

func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found while resolving module root")
		}
		dir = parent
	}
}

// TestdataPath returns an absolute path under the repo testdata/ directory.
func TestdataPath(t *testing.T, elems ...string) string {
	t.Helper()
	parts := append([]string{moduleRoot(t), "testdata"}, elems...)
	return filepath.Join(parts...)
}
