package registry

import (
	"os"
	"path/filepath"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
)

// PathStatus is the proactive check of a registered path.
type PathStatus int

const (
	// PathOK means the path is a directory that contains a manifest.
	PathOK PathStatus = iota
	// PathMissing means the path does not exist.
	PathMissing
	// PathNotDir means the path exists but is not a directory.
	PathNotDir
	// PathInvalid means the path is a directory but has no Project Manifest.
	PathInvalid
)

// RepairAction is an interactive option offered when a path is broken.
type RepairAction int

const (
	// RepairRelink asks the user for a corrected path.
	RepairRelink RepairAction = iota
	// RepairRemove drops the registry entry.
	RepairRemove
)

// CheckPath classifies a registered filesystem path.
func CheckPath(path string) PathStatus {
	info, err := os.Stat(path)
	if err != nil {
		return PathMissing
	}
	if !info.IsDir() {
		return PathNotDir
	}
	manifestPath := filepath.Join(path, manifest.FileName)
	fi, err := os.Stat(manifestPath)
	if err != nil || fi.IsDir() {
		return PathInvalid
	}
	return PathOK
}

// SuggestRepair returns the actions the CLI should offer for status.
// PathOK needs no repair.
func SuggestRepair(status PathStatus) []RepairAction {
	if status == PathOK {
		return nil
	}
	return []RepairAction{RepairRelink, RepairRemove}
}
