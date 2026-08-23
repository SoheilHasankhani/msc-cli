package dockerapi

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryAuthHeaderFromAuths(t *testing.T) {
	dir := t.TempDir()
	userPass := base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	cfg := `{"auths":{"registry.isos.clinic":{"auth":"` + userPass + `"}}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOCKER_CONFIG", dir)

	got, err := RegistryAuthHeader("registry.isos.clinic/isos/wallet:latest")
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected auth header")
	}
}

func TestRegistryHost(t *testing.T) {
	t.Parallel()

	if registryHost("registry.isos.clinic/isos/wallet:latest") != "registry.isos.clinic" {
		t.Fatal("private registry host")
	}
	if registryHost("alpine:latest") != "https://index.docker.io/v1/" {
		t.Fatal("hub default")
	}
}
