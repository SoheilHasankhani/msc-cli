//go:build ignore

// Lint runner used by `make lint`. Downloads the same golangci-lint version as CI.
package main

import (
	"fmt"
	"os"
	"os/exec"
)

const golangciLint = "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0"

func main() {
	args := append([]string{"run", golangciLint, "run"}, os.Args[1:]...)
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
