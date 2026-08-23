//go:build integration

package dockerapi

import (
	"context"
	"testing"
	"time"
)

func TestEngineListContainers(t *testing.T) {
	e, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := e.ListContainers(ctx); err != nil {
		t.Fatalf("ListContainers: %v", err)
	}
}
