package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/project"
)

func quotePath(p string) string {
	return `"` + strings.ReplaceAll(p, `"`, `\"`) + `"`
}

// UnixScript is a POSIX shim that sets MSC_PROJECT and execs the engine.
func UnixScript(enginePath, projectName string) string {
	return fmt.Sprintf("#!/bin/sh\nexec env %s=%q %s \"$@\"\n", project.EnvVar, projectName, quotePath(enginePath))
}

// CmdScript is a cmd.exe shim for Windows.
func CmdScript(enginePath, projectName string) string {
	return fmt.Sprintf("@echo off\r\nset %s=%s\r\n%s %%*\r\n", project.EnvVar, projectName, quotePath(enginePath))
}

// Write creates platform shims in shimDir. On Windows it writes a .cmd launcher.
func Write(shimDir, projectName, enginePath, goos string) (string, error) {
	if projectName == "" {
		return "", fmt.Errorf("project name is required")
	}
	if enginePath == "" {
		return "", fmt.Errorf("engine path is required")
	}
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		return "", err
	}
	if goos == "windows" {
		path := filepath.Join(shimDir, projectName+".cmd")
		if err := os.WriteFile(path, []byte(CmdScript(enginePath, projectName)), 0o755); err != nil {
			return "", err
		}
		return path, nil
	}
	path := filepath.Join(shimDir, projectName)
	if err := os.WriteFile(path, []byte(UnixScript(enginePath, projectName)), 0o755); err != nil {
		return "", err
	}
	return path, nil
}

// EngineOnPATH is a fallback when os.Executable fails.
func EngineOnPATH() string {
	return "msc"
}

// ValidUnix reports whether path is a POSIX shim script (not a copied binary).
func ValidUnix(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 2 {
		return false
	}
	return strings.HasPrefix(string(data), "#!")
}

// ValidCmd reports whether path is a cmd.exe brand shim that launches msc.
func ValidCmd(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 8 {
		return false
	}
	text := strings.ToLower(string(data))
	return strings.HasPrefix(text, "@echo off") && strings.Contains(text, "msc_project=")
}

// ValidBrandShim reports whether the platform shim for projectName exists in shimDir.
func ValidBrandShim(shimDir, projectName, goos string) bool {
	if projectName == "" {
		return false
	}
	if goos == "windows" {
		return ValidCmd(filepath.Join(shimDir, projectName+".cmd"))
	}
	return ValidUnix(filepath.Join(shimDir, projectName))
}
