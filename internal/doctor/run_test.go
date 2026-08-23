package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/elevate"
	"github.com/SoheilHasankhani/msc-cli/internal/hostcerts"
	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/testenv"
)

func TestSSHPrivateKeysIn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if sshPrivateKeysIn(dir) {
		t.Fatal("empty dir should have no keys")
	}
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519.pub"), []byte("pub"), 0o644); err != nil {
		t.Fatal(err)
	}
	if sshPrivateKeysIn(dir) {
		t.Fatal(".pub alone should not count")
	}
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !sshPrivateKeysIn(dir) {
		t.Fatal("expected private key")
	}
}

func TestCheckSSHWithoutAgent(t *testing.T) {
	t.Parallel()

	c := checkSSH(Options{SSHStatus: func() (bool, bool, error) { return false, true, nil }})
	if c.Status != StatusPass || !strings.Contains(c.Message, "~/.ssh") {
		t.Fatalf("check = %#v", c)
	}
	c = checkSSH(Options{SSHStatus: func() (bool, bool, error) { return false, false, nil }})
	if c.Status != StatusFail || !strings.Contains(c.Message, "no private keys") {
		t.Fatalf("check = %#v", c)
	}
}

func TestRunMachineChecksOnly(t *testing.T) {
	t.Parallel()

	r, err := Run(context.Background(), Options{
		LookPath: func(name string) (string, error) {
			if name == "git" {
				return "/usr/bin/git", nil
			}
			return "", errors.New("missing")
		},
		DockerPing: func(context.Context) error { return errors.New("no docker") },
		SSHStatus:  func() (bool, bool, error) { return false, false, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	by := byName(r)
	if by["git"].Status != StatusPass {
		t.Fatalf("git = %#v", by["git"])
	}
	if by["docker"].Status != StatusFail || by["docker"].Fix != FixNone {
		t.Fatalf("docker = %#v", by["docker"])
	}
	if by["ssh"].Status != StatusFail {
		t.Fatalf("ssh = %#v", by["ssh"])
	}
	if _, ok := by["hosts"]; ok {
		t.Fatal("hosts must not run without a project")
	}
	if !r.HasFail() {
		t.Fatal("expected fail")
	}
}

func TestRunProjectChecksHostsAndOverlay(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	comp := filepath.Join(root, "local", "config", "nginx", "components")
	if err := os.MkdirAll(comp, 0o755); err != nil {
		t.Fatal(err)
	}
	nginx, err := os.ReadFile(testenv.TestdataPath(t, "nginx", "default.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comp, "default.conf"), nginx, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "local", "docker-compose.yml"), []byte("services:\n  nginx:\n    image: nginx\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &manifest.Manifest{
		Brand:         manifest.BrandInfo{DisplayName: "isos", Command: "isos"},
		GitHost:       manifest.GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain:   "isos.local",
		Prerequisites: []string{"docker", "git", "ssh", "dotnet"},
		Repos: []manifest.RepoDef{{
			Name: "doctor-api", Git: "sos/doctor-api",
			Services: []manifest.ServiceDef{{ComposeService: "doctor", Path: ".", SourcePort: 5010}},
		}},
	}
	m.ApplyDefaults()
	p := &project.Context{Name: "isos", Root: root, Manifest: m}

	var overlayCalled bool
	r, err := Run(context.Background(), Options{
		Project: p,
		Fix:     true,
		LookPath: func(name string) (string, error) {
			if name == "dotnet" {
				return "", errors.New("missing")
			}
			return "/bin/" + name, nil
		},
		DockerPing:      func(context.Context) error { return nil },
		SSHStatus:       func() (bool, bool, error) { return true, true, nil },
		HostsText:       "127.0.0.1 localhost\n",
		MachineCertsDir: filepath.Join(t.TempDir(), "msc-certs"),
		EnsureOverlay: func() (bool, error) {
			overlayCalled = true
			return true, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	by := byName(r)
	if by["dotnet"].Status != StatusFail || by["dotnet"].Fix != FixNone {
		t.Fatalf("dotnet = %#v", by["dotnet"])
	}
	if by["hosts"].Status != StatusFail || !strings.Contains(by["hosts"].Message, "doctor.isos.local") {
		t.Fatalf("hosts = %#v", by["hosts"])
	}
	if by["hosts"].Fix != FixHosts {
		t.Fatalf("hosts fix = %s", by["hosts"].Fix)
	}
	if !overlayCalled {
		t.Fatal("expected overlay fix")
	}
	if len(r.Fixed) == 0 || !strings.Contains(strings.Join(r.Fixed, " "), "overlay") {
		t.Fatalf("fixed = %v", r.Fixed)
	}
	if !containsSkipped(r, "hosts") {
		t.Fatalf("hosts should be skipped without an elevator: %v", r.Skipped)
	}
	if !containsSkipped(r, "certs") {
		t.Fatalf("OS/NSS trust should be skipped without an elevator: %v", r.Skipped)
	}
	if !containsFixed(r, "certs") {
		t.Fatalf("expected cert files to be generated: %v", r.Fixed)
	}
	if by["certs"].Status != StatusPass {
		t.Fatalf("certs after successful --fix = %#v", by["certs"])
	}
}

func TestRunProjectFixAppliesHostsAndTrust(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	comp := filepath.Join(root, "local", "config", "nginx", "components")
	if err := os.MkdirAll(comp, 0o755); err != nil {
		t.Fatal(err)
	}
	nginx, err := os.ReadFile(testenv.TestdataPath(t, "nginx", "default.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comp, "default.conf"), nginx, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "local", "docker-compose.yml"), []byte("services:\n  nginx:\n    extra_hosts:\n      - host.docker.internal:host-gateway\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &manifest.Manifest{
		Brand:       manifest.BrandInfo{DisplayName: "isos", Command: "isos"},
		GitHost:     manifest.GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain: "isos.local",
		Repos: []manifest.RepoDef{{
			Name: "doctor-api", Git: "sos/doctor-api",
			Services: []manifest.ServiceDef{{ComposeService: "doctor", Path: ".", SourcePort: 5010}},
		}},
	}
	m.ApplyDefaults()
	p := &project.Context{Name: "isos", Root: root, Manifest: m}

	var gotNames []string
	var trusted string
	r, err := Run(context.Background(), Options{
		Project:    p,
		Fix:        true,
		LookPath:   func(string) (string, error) { return "/bin/x", nil },
		DockerPing: func(context.Context) error { return nil },
		SSHStatus:  func() (bool, bool, error) { return true, true, nil },
		HostsText:  "127.0.0.1 localhost\n",
		ApplyHosts: func(project string, names []string) error {
			if project != "isos" {
				t.Fatalf("project %s", project)
			}
			gotNames = names
			return nil
		},
		MachineCertsDir: filepath.Join(t.TempDir(), "msc-certs"),
		TrustCA:         func(caPath string) error { trusted = caPath; return nil },
		TrustNSS:        func(string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsFixed(r, "hosts") || !containsFixed(r, "certs") {
		t.Fatalf("fixed=%v skipped=%v", r.Fixed, r.Skipped)
	}
	if trusted == "" || !strings.Contains(strings.Join(gotNames, ","), "doctor.isos.local") {
		t.Fatalf("names=%v trusted=%s", gotNames, trusted)
	}
	if strings.Contains(strings.Join(gotNames, ","), "metrics_status_internal") {
		t.Fatal(gotNames)
	}
}

func TestRunProjectFixNoElevateStillWritesCerts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	comp := filepath.Join(root, "local", "config", "nginx", "components")
	if err := os.MkdirAll(comp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comp, "default.conf"), []byte("server_name doctor.isos.local;"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "local", "docker-compose.yml"), []byte("services:\n  nginx:\n    extra_hosts:\n      - host.docker.internal:host-gateway\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Brand:       manifest.BrandInfo{DisplayName: "isos", Command: "isos"},
		GitHost:     manifest.GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain: "isos.local",
		Repos: []manifest.RepoDef{{
			Name: "doctor-api", Git: "sos/doctor-api",
			Services: []manifest.ServiceDef{{ComposeService: "doctor", Path: ".", SourcePort: 5010}},
		}},
	}
	m.ApplyDefaults()
	p := &project.Context{Name: "isos", Root: root, Manifest: m}

	hostsCalled := false
	r, err := Run(context.Background(), Options{
		Project:         p,
		Fix:             true,
		NoElevate:       true,
		LookPath:        func(string) (string, error) { return "/bin/x", nil },
		DockerPing:      func(context.Context) error { return nil },
		SSHStatus:       func() (bool, bool, error) { return true, true, nil },
		HostsText:       "127.0.0.1 localhost\n",
		MachineCertsDir: filepath.Join(t.TempDir(), "msc-certs"),
		Elevate:         elevate.Direct{Handle: func(context.Context, []string) error { t.Fatal("elevate"); return nil }},
		ApplyHosts:      func(string, []string) error { hostsCalled = true; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if hostsCalled {
		t.Fatal("hosts must not run with --no-elevate")
	}
	if !containsFixed(r, "certs") || !containsSkipped(r, "no-elevate") {
		t.Fatalf("fixed=%v skipped=%v", r.Fixed, r.Skipped)
	}
}

func TestRunProjectFixHostsNeedTTY(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	comp := filepath.Join(root, "local", "config", "nginx", "components")
	if err := os.MkdirAll(comp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comp, "default.conf"), []byte("server_name doctor.isos.local;"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "local", "docker-compose.yml"), []byte("services:\n  nginx:\n    extra_hosts:\n      - host.docker.internal:host-gateway\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Brand:       manifest.BrandInfo{DisplayName: "isos", Command: "isos"},
		GitHost:     manifest.GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain: "isos.local",
		Repos: []manifest.RepoDef{{
			Name: "doctor-api", Git: "sos/doctor-api",
			Services: []manifest.ServiceDef{{ComposeService: "doctor", Path: ".", SourcePort: 5010}},
		}},
	}
	m.ApplyDefaults()
	p := &project.Context{Name: "isos", Root: root, Manifest: m}

	r, err := Run(context.Background(), Options{
		Project:         p,
		Fix:             true,
		LookPath:        func(string) (string, error) { return "/bin/x", nil },
		DockerPing:      func(context.Context) error { return nil },
		SSHStatus:       func() (bool, bool, error) { return true, true, nil },
		HostsText:       "127.0.0.1 localhost\n",
		MachineCertsDir: filepath.Join(t.TempDir(), "msc-certs"),
		ApplyHosts:      func(string, []string) error { return elevate.ErrNeedTTY },
		TrustCA:         func(string) error { return elevate.ErrNeedTTY },
		TrustNSS:        func(string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsSkipped(r, "interactive terminal") {
		t.Fatalf("%v", r.Skipped)
	}
}

func TestRunProjectFixRechecksCertsAndTrust(t *testing.T) {
	t.Parallel()

	p, machine := doctorProject(t)
	r, err := Run(context.Background(), Options{
		Project:         p,
		Fix:             true,
		LookPath:        func(string) (string, error) { return "/bin/x", nil },
		DockerPing:      func(context.Context) error { return nil },
		SSHStatus:       func() (bool, bool, error) { return true, true, nil },
		HostsText:       "127.0.0.1 isos.local doctor.isos.local\n",
		MachineCertsDir: machine,
		TrustOK:         func(string) bool { return true },
		TrustCA:         func(string) error { return nil },
		TrustNSS:        func(string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	by := byName(r)
	if by["certs"].Status != StatusPass {
		t.Fatalf("certs = %#v fixed=%v", by["certs"], r.Fixed)
	}
	if by["os-trust"].Status != StatusPass {
		t.Fatalf("os-trust = %#v fixed=%v", by["os-trust"], r.Fixed)
	}
}

func TestRunProjectFixRetrustsWhenFingerprintDiffers(t *testing.T) {
	t.Parallel()

	p, machine := doctorProject(t)
	if _, err := hostcerts.Ensure(machine, hostcerts.Dir(p.ConfigDir()), "isos.local"); err != nil {
		t.Fatal(err)
	}
	var trusted string
	r, err := Run(context.Background(), Options{
		Project:         p,
		Fix:             true,
		LookPath:        func(string) (string, error) { return "/bin/x", nil },
		DockerPing:      func(context.Context) error { return nil },
		SSHStatus:       func() (bool, bool, error) { return true, true, nil },
		HostsText:       "127.0.0.1 isos.local doctor.isos.local\n",
		MachineCertsDir: machine,
		TrustOK:         func(string) bool { return false },
		TrustCA:         func(caPath string) error { trusted = caPath; return nil },
		TrustNSS:        func(string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if trusted == "" {
		t.Fatalf("expected OS trust reinstall: fixed=%v skipped=%v", r.Fixed, r.Skipped)
	}
	if !containsFixed(r, "OS trust") {
		t.Fatalf("fixed=%v", r.Fixed)
	}
}

func TestRunProjectFixSkipsTrustWhenFingerprintMatches(t *testing.T) {
	t.Parallel()

	p, machine := doctorProject(t)
	if _, err := hostcerts.Ensure(machine, hostcerts.Dir(p.ConfigDir()), "isos.local"); err != nil {
		t.Fatal(err)
	}
	r, err := Run(context.Background(), Options{
		Project:         p,
		Fix:             true,
		LookPath:        func(string) (string, error) { return "/bin/x", nil },
		DockerPing:      func(context.Context) error { return nil },
		SSHStatus:       func() (bool, bool, error) { return true, true, nil },
		HostsText:       "127.0.0.1 isos.local doctor.isos.local\n",
		MachineCertsDir: machine,
		TrustOK:         func(string) bool { return true },
		TrustCA:         func(string) error { t.Fatal("must not reinstall matching CA"); return nil },
		TrustNSS:        func(string) error { t.Fatal("must not touch NSS"); return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if containsFixed(r, "OS trust") || containsSkipped(r, "OS trust") {
		t.Fatalf("fixed=%v skipped=%v", r.Fixed, r.Skipped)
	}
}

func doctorProject(t *testing.T) (*project.Context, string) {
	t.Helper()
	root := t.TempDir()
	comp := filepath.Join(root, "local", "config", "nginx", "components")
	if err := os.MkdirAll(comp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comp, "default.conf"), []byte("server_name doctor.isos.local;"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "local", "docker-compose.yml"), []byte("services:\n  nginx:\n    extra_hosts:\n      - host.docker.internal:host-gateway\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Brand:       manifest.BrandInfo{DisplayName: "isos", Command: "isos"},
		GitHost:     manifest.GitHostInfo{BaseURL: "https://gitlab.example.com"},
		LocalDomain: "isos.local",
		Repos: []manifest.RepoDef{{
			Name: "doctor-api", Git: "sos/doctor-api",
			Services: []manifest.ServiceDef{{ComposeService: "doctor", Path: ".", SourcePort: 5010}},
		}},
	}
	m.ApplyDefaults()
	return &project.Context{Name: "isos", Root: root, Manifest: m}, filepath.Join(t.TempDir(), "msc-certs")
}

func byName(r Report) map[string]Check {
	out := map[string]Check{}
	for _, c := range r.Checks {
		out[c.Name] = c
	}
	return out
}

func containsFixed(r Report, needle string) bool {
	for _, s := range r.Fixed {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func containsSkipped(r Report, needle string) bool {
	for _, s := range r.Skipped {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
