//go:build ignore

// Isolated unit-test runner used by `make test`.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	pinGoCaches()

	root, err := os.MkdirTemp("", "msc-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(root) }()

	os.Setenv("HOME", root)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	os.Setenv("APPDATA", filepath.Join(root, "AppData", "Roaming"))
	os.Setenv("USERPROFILE", root)
	os.Setenv("LOCALAPPDATA", filepath.Join(root, "AppData", "Local"))

	args := append([]string{"test", "./...", "-count=1"}, os.Args[1:]...)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// pinGoCaches keeps the real module/build caches after HOME is redirected.
func pinGoCaches() {
	home := os.Getenv("HOME")
	profile := os.Getenv("USERPROFILE")
	local := os.Getenv("LOCALAPPDATA")
	if os.Getenv("GOPATH") == "" {
		base := profile
		if base == "" {
			base = home
		}
		if base != "" {
			os.Setenv("GOPATH", filepath.Join(base, "go"))
		}
	}
	if os.Getenv("GOMODCACHE") == "" && os.Getenv("GOPATH") != "" {
		os.Setenv("GOMODCACHE", filepath.Join(os.Getenv("GOPATH"), "pkg", "mod"))
	}
	if os.Getenv("GOCACHE") == "" {
		switch {
		case local != "":
			os.Setenv("GOCACHE", filepath.Join(local, "go-build"))
		case home != "":
			os.Setenv("GOCACHE", filepath.Join(home, ".cache", "go-build"))
		}
	}
}
