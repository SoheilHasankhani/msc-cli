package paths

import (
	"path/filepath"
	"testing"
)

func TestConfigDirLinuxUsesXDG(t *testing.T) {
	t.Parallel()

	r := Resolver{
		GOOS:       "linux",
		Home:       "/home/dev",
		ConfigHome: "/home/dev/.config-custom",
	}

	got := r.ConfigDir()
	want := filepath.Join("/home/dev/.config-custom", AppName)
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDirLinuxFallsBackToDotConfig(t *testing.T) {
	t.Parallel()

	r := Resolver{GOOS: "linux", Home: "/home/dev"}

	got := r.ConfigDir()
	want := filepath.Join("/home/dev", ".config", AppName)
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDirDarwinUsesApplicationSupport(t *testing.T) {
	t.Parallel()

	r := Resolver{GOOS: "darwin", Home: "/Users/dev"}

	got := r.ConfigDir()
	want := filepath.Join("/Users/dev", "Library", "Application Support", AppName)
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestConfigDirWindowsUsesAppData(t *testing.T) {
	t.Parallel()

	r := Resolver{
		GOOS:    "windows",
		Home:    `C:\Users\dev`,
		AppData: `C:\Users\dev\AppData\Roaming`,
	}

	got := r.ConfigDir()
	want := filepath.Join(`C:\Users\dev\AppData\Roaming`, AppName)
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestDerivedPathsSitUnderConfigDir(t *testing.T) {
	t.Parallel()

	r := Resolver{GOOS: "linux", Home: "/home/dev", ConfigHome: "/xdg"}

	if got, want := r.RegistryFile(), filepath.Join(r.ConfigDir(), "projects.yml"); got != want {
		t.Fatalf("RegistryFile() = %q, want %q", got, want)
	}
	if got, want := r.StateDir(), filepath.Join(r.ConfigDir(), "state"); got != want {
		t.Fatalf("StateDir() = %q, want %q", got, want)
	}
	if got, want := r.LogDir(), filepath.Join(r.ConfigDir(), "logs"); got != want {
		t.Fatalf("LogDir() = %q, want %q", got, want)
	}
	if got, want := r.CertsDir(), filepath.Join(r.ConfigDir(), "certs"); got != want {
		t.Fatalf("CertsDir() = %q, want %q", got, want)
	}
}

func TestConfigDirWindowsFallsBackWithoutAppData(t *testing.T) {
	t.Parallel()

	r := Resolver{GOOS: "windows", Home: `C:\Users\dev`}
	got := r.ConfigDir()
	want := filepath.Join(`C:\Users\dev`, "AppData", "Roaming", AppName)
	if got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestDefaultResolverUsesRealEnvironment(t *testing.T) {
	t.Parallel()

	r := Default()
	if r.ConfigDir() == "" || r.RegistryFile() == "" || r.ShimDir() == "" {
		t.Fatalf("Default() produced empty paths: %#v", r)
	}
}

func TestShimDirIsUnderHomeDotMsc(t *testing.T) {
	t.Parallel()

	r := Resolver{GOOS: "linux", Home: "/home/dev"}

	got := r.ShimDir()
	want := filepath.Join("/home/dev", ".msc", "shims")
	if got != want {
		t.Fatalf("ShimDir() = %q, want %q", got, want)
	}
}
