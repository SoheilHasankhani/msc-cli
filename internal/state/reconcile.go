package state

import (
	"fmt"
	"strings"
)

// ReconcileResult is live state plus whether it disagreed with the cache.
type ReconcileResult struct {
	State   ServiceState
	Drift   bool
	Message string
}

// Reconcile applies the hybrid rule: live inference always wins.
// An empty cached ComposeService means "no prior observation", not drift.
// Commands that change containers (up/down/switch) must SyncLive first so
// the next status does not treat that expected change as drift.
func Reconcile(cached, live ServiceState) ReconcileResult {
	if cached.ComposeService == "" {
		return ReconcileResult{State: live}
	}
	if cached.Mode == live.Mode && cached.ContainerUp == live.ContainerUp && cached.NginxTarget == live.NginxTarget {
		return ReconcileResult{State: live}
	}
	return ReconcileResult{
		State: live,
		Drift: true,
		Message: fmt.Sprintf(
			"service %s in cache was %s (container_up=%t) but live is %s (container_up=%t); cache updated from reality",
			live.ComposeService, cached.Mode, cached.ContainerUp, live.Mode, live.ContainerUp,
		),
	}
}

// InferMode decides Docker vs Source from the nginx target. Container running
// state is recorded separately; nginx is the routing source of truth.
func InferMode(containerUp bool, nginxTarget string) Mode {
	_ = containerUp
	if strings.HasPrefix(nginxTarget, "source_port:") {
		return ModeSource
	}
	return ModeDocker
}
