//go:build tools

// Package tools pins CLI development binaries (mockgen, ...) so `go generate`
// stays reproducible across machines and CI.
package tools

import (
	_ "go.uber.org/mock/mockgen"
)
