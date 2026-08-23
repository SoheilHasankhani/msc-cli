package nginxcfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasHostGateway(t *testing.T) {
	t.Parallel()

	if HasHostGateway("services:\n  nginx:\n    image: nginx\n") {
		t.Fatal("plain compose should not report host-gateway")
	}
	if !HasHostGateway(`extra_hosts:
      - "host.docker.internal:host-gateway"`) {
		t.Fatal("quoted extra_hosts")
	}
	if !HasHostGateway("extra_hosts:\n  - host.docker.internal:host-gateway") {
		t.Fatal("unquoted extra_hosts")
	}
}

func TestOverlayYAML(t *testing.T) {
	t.Parallel()

	got := OverlayYAML("nginx")
	if !strings.Contains(got, "host.docker.internal:host-gateway") {
		t.Fatalf("missing host-gateway:\n%s", got)
	}
	if !strings.Contains(got, "nginx:") {
		t.Fatalf("missing nginx service:\n%s", got)
	}
}

func TestEnsureHostGatewayWritesSiblingOverlay(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	composeRel := filepath.Join("local", "docker-compose.yml")
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, composeRel), []byte("services:\n  nginx:\n    image: nginx\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	path, wrote, err := EnsureHostGateway(root, composeRel)
	if err != nil {
		t.Fatal(err)
	}
	if !wrote {
		t.Fatal("expected overlay write")
	}
	if filepath.Base(path) != OverlayFileName {
		t.Fatalf("path = %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !HasHostGateway(string(data)) {
		t.Fatalf("overlay missing host-gateway:\n%s", data)
	}

	_, wrote, err = EnsureHostGateway(root, composeRel)
	if err != nil {
		t.Fatal(err)
	}
	if wrote {
		t.Fatal("second call should be a no-op")
	}
}

func TestEnsureHostGatewaySkipsWhenAlreadyPresent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	composeRel := "docker-compose.yml"
	body := "services:\n  nginx:\n    extra_hosts:\n      - host.docker.internal:host-gateway\n"
	if err := os.WriteFile(filepath.Join(root, composeRel), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, wrote, err := EnsureHostGateway(root, composeRel)
	if err != nil {
		t.Fatal(err)
	}
	if wrote {
		t.Fatal("should not write overlay when compose already has host-gateway")
	}
	if _, err := os.Stat(filepath.Join(root, OverlayFileName)); err == nil {
		t.Fatal("overlay file should not exist")
	}
}
