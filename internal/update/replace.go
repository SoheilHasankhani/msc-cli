package update

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ReplaceExecutable writes data over target using a same-directory rename.
func ReplaceExecutable(target string, data []byte) error {
	if target == "" {
		return fmt.Errorf("install path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".msc-new-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if runtime.GOOS == "windows" {
		old := target + ".old"
		_ = os.Remove(old)
		if _, err := os.Stat(target); err == nil {
			if err := os.Rename(target, old); err != nil {
				_ = os.Remove(tmpName)
				return err
			}
		}
		if err := os.Rename(tmpName, target); err != nil {
			_ = os.Rename(old, target)
			return err
		}
		_ = os.Remove(old)
		return nil
	}
	return os.Rename(tmpName, target)
}
