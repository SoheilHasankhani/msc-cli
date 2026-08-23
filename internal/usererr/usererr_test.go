package usererr

import (
	"errors"
	"strings"
	"testing"
)

func TestDockerDaemonDown(t *testing.T) {
	t.Parallel()

	err := Docker(errors.New("Cannot connect to the Docker daemon at unix:///var/run/docker.sock"), "")
	if err == nil || !strings.Contains(err.Error(), "Docker Engine is not running") {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(err.Error(), "DOCKER_HOST") {
		t.Fatalf("missing DOCKER_HOST hint: %v", err)
	}
}

func TestDockerConnectionRefused(t *testing.T) {
	t.Parallel()

	err := Docker(errors.New("docker ping: connect: connection refused"), "")
	if !IsDockerDown(err) {
		t.Fatalf("%v", err)
	}
}

func TestDockerPermission(t *testing.T) {
	t.Parallel()

	err := Docker(errors.New("dial unix /var/run/docker.sock: connect: permission denied"), "")
	if err == nil || !strings.Contains(err.Error(), "docker group") {
		t.Fatalf("%v", err)
	}
}

func TestComposePortConflict(t *testing.T) {
	t.Parallel()

	err := Compose(errors.New("exit status 1"), "Error: Bind for 0.0.0.0:80 failed: port is already allocated\n")
	if err == nil || !strings.Contains(err.Error(), "only one stack") {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(err.Error(), "down") {
		t.Fatalf("missing down hint: %v", err)
	}
}

func TestComposeDockerDownOnStderr(t *testing.T) {
	t.Parallel()

	err := Compose(errors.New("exit status 1"), "Cannot connect to the Docker daemon. Is the docker daemon running?\n")
	if !IsDockerDown(err) {
		t.Fatalf("%v", err)
	}
}

func TestNginxReloadKeepsPreviousRouting(t *testing.T) {
	t.Parallel()

	err := NginxReload(errors.New("signal: connection reset"))
	if err == nil || !strings.Contains(err.Error(), "previous routing is unchanged") {
		t.Fatalf("%v", err)
	}
}

func TestPassthroughUnknownLeavesMessage(t *testing.T) {
	t.Parallel()

	orig := errors.New("unexpected compose failure")
	err := Compose(orig, "some other problem\n")
	if err == nil || !strings.Contains(err.Error(), "unexpected compose failure") {
		t.Fatalf("%v", err)
	}
}
