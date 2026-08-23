package state

import "testing"

func TestReconcileLiveWinsOnDrift(t *testing.T) {
	t.Parallel()

	cached := ServiceState{ComposeService: "doctor", Mode: ModeDocker, ContainerUp: true, NginxTarget: NginxTargetContainer}
	live := ServiceState{ComposeService: "doctor", Mode: ModeSource, ContainerUp: false, NginxTarget: NginxSourceTarget(5010)}

	got := Reconcile(cached, live)
	if !got.Drift {
		t.Fatal("expected drift")
	}
	if got.State.Mode != ModeSource || got.State.ContainerUp {
		t.Fatalf("live must win: %#v", got.State)
	}
	if got.Message == "" {
		t.Fatal("drift must produce a user-visible message")
	}
}

func TestReconcileNoDriftWhenEqual(t *testing.T) {
	t.Parallel()

	s := ServiceState{ComposeService: "doctor", Mode: ModeDocker, ContainerUp: true, NginxTarget: NginxTargetContainer}
	got := Reconcile(s, s)
	if got.Drift {
		t.Fatal("identical cache and live must not report drift")
	}
	if got.Message != "" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

func TestReconcileEmptyCacheIsNotDrift(t *testing.T) {
	t.Parallel()

	live := ServiceState{ComposeService: "doctor", Mode: ModeDocker, ContainerUp: true, NginxTarget: NginxTargetContainer}
	got := Reconcile(ServiceState{}, live)
	if got.Drift {
		t.Fatal("first observation (empty cache) is not drift")
	}
	if got.State.ComposeService != "doctor" {
		t.Fatalf("state = %#v", got.State)
	}
}

func TestInferModeFromNginxTarget(t *testing.T) {
	t.Parallel()

	if got := InferMode(true, NginxTargetContainer); got != ModeDocker {
		t.Fatalf("got %s, want docker", got)
	}
	if got := InferMode(false, NginxSourceTarget(5000)); got != ModeSource {
		t.Fatalf("got %s, want source", got)
	}
	if got := InferMode(true, ""); got != ModeDocker {
		t.Fatalf("empty nginx target defaults to docker, got %s", got)
	}
}
