package project

import (
	"os"
	"strings"
)

// EnvVar is set by brand shims (isos, mores, …) instead of passing --project on argv.
const EnvVar = "MSC_PROJECT"

// FromEnv returns the project name injected by a brand shim.
func FromEnv() string {
	return strings.TrimSpace(os.Getenv(EnvVar))
}

// MissingMessage explains how to select a project when none is set.
func MissingMessage() string {
	return "no project specified; use a brand shim (e.g. isos status) or msc --project <name>"
}
