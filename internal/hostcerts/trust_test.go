package hostcerts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxCADest(t *testing.T) {
	t.Parallel()
	if LinuxCADest() != "/usr/local/share/ca-certificates/msc-local-ca.crt" {
		t.Fatal(LinuxCADest())
	}
}

func TestOSTrustPlan(t *testing.T) {
	t.Parallel()

	linux := OSTrustPlan("linux", "/tmp/local-ca.crt")
	if linux.Dest != LinuxCADest() || linux.UpdateCmd != "update-ca-certificates" {
		t.Fatalf("%+v", linux)
	}
	darwin := OSTrustPlan("darwin", "/tmp/local-ca.crt")
	if darwin.Tool != "security" || !strings.Contains(strings.Join(darwin.Args, " "), "add-trusted-cert") {
		t.Fatalf("%+v", darwin)
	}
	win := OSTrustPlan("windows", `C:\ca.crt`)
	if win.Tool != "certutil" || !contains(win.Args, "ROOT") {
		t.Fatalf("%+v", win)
	}
}

func TestOSTrustMatchesUsesStampOnDarwin(t *testing.T) {
	t.Parallel()

	b := mustBundle(t)
	fp, err := FileFingerprint(b.CACrt)
	if err != nil {
		t.Fatal(err)
	}
	if OSTrustMatches("darwin", b.CACrt, b.MachineDir) {
		t.Fatal("no stamp yet")
	}
	if err := WriteTrustStamp(b.MachineDir, fp); err != nil {
		t.Fatal(err)
	}
	if !OSTrustMatches("darwin", b.CACrt, b.MachineDir) {
		t.Fatal("stamp should match")
	}
	if err := WriteTrustStamp(b.MachineDir, "deadbeef"); err != nil {
		t.Fatal(err)
	}
	if OSTrustMatches("darwin", b.CACrt, b.MachineDir) {
		t.Fatal("stale stamp must not match")
	}
}

func TestInstallNSSInitsEmptyDBThenAddsWithoutPrompt(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	b := mustBundle(t)

	var got [][]string
	err := InstallNSS(home, b.CACrt, func(name string, args ...string) error {
		got = append(got, append([]string{name}, args...))
		return nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
	if !contains(got[0], "-N") || !contains(got[0], "--empty-password") {
		t.Fatalf("init: %v", got[0])
	}
	if !contains(got[1], "-D") || !contains(got[1], NSSNickname()) {
		t.Fatalf("delete: %v", got[1])
	}
	if !contains(got[2], "-A") || !contains(got[2], "-f") || !contains(got[2], NSSNickname()) {
		t.Fatalf("add: %v", got[2])
	}
}

func TestInstallNSSSkipsInitWhenDBExists(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	db := filepath.Join(home, ".pki", "nssdb")
	if err := os.MkdirAll(db, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(db, "cert9.db"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	b := mustBundle(t)

	var got [][]string
	err := InstallNSS(home, b.CACrt, func(_ string, args ...string) error {
		got = append(got, append([]string(nil), args...))
		return nil
	}, func(string, ...string) error {
		t.Fatal("must not prompt when empty password works")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || contains(got[0], "-N") || !contains(got[0], "-D") || !contains(got[1], "-A") || !contains(got[1], "-f") {
		t.Fatalf("%v", got)
	}
}

func TestInstallNSSPromptsExistingPasswordOnly(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	db := filepath.Join(home, ".pki", "nssdb")
	if err := os.MkdirAll(db, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(db, "cert9.db"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	b := mustBundle(t)

	var silent [][]string
	var prompted [][]string
	err := InstallNSS(home, b.CACrt, func(_ string, args ...string) error {
		silent = append(silent, append([]string(nil), args...))
		if contains(args, "-D") {
			return nil
		}
		return fmt.Errorf("certutil: SEC_ERROR_BAD_PASSWORD: The security password entered is incorrect.")
	}, func(_ string, args ...string) error {
		prompted = append(prompted, append([]string(nil), args...))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(silent) != 2 || !contains(silent[1], "-A") || !contains(silent[1], "-f") {
		t.Fatalf("silent=%v", silent)
	}
	if len(prompted) != 2 || !contains(prompted[1], "-A") || contains(prompted[1], "-f") || contains(prompted[1], "-N") {
		t.Fatalf("prompted=%v", prompted)
	}
}

func TestInstallNSSPasswordRequiredWithoutTTY(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	db := filepath.Join(home, ".pki", "nssdb")
	if err := os.MkdirAll(db, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(db, "cert9.db"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := InstallNSS(home, mustBundle(t).CACrt, func(_ string, args ...string) error {
		if contains(args, "-D") {
			return nil
		}
		return fmt.Errorf("SEC_ERROR_BAD_PASSWORD")
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "password") {
		t.Fatalf("%v", err)
	}
}

func TestInstallNSSPropagatesRunnerError(t *testing.T) {
	t.Parallel()
	err := InstallNSS(t.TempDir(), mustBundle(t).CACrt, func(_ string, args ...string) error {
		if contains(args, "-D") {
			return nil
		}
		return os.ErrPermission
	}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallNSSTreatsExistingNicknameAsSuccess(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	db := filepath.Join(home, ".pki", "nssdb")
	if err := os.MkdirAll(db, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(db, "cert9.db"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := InstallNSS(home, mustBundle(t).CACrt, func(_ string, args ...string) error {
		if contains(args, "-A") {
			return fmt.Errorf("certutil: certificate already exists")
		}
		return nil
	}, func(string, ...string) error {
		t.Fatal("must not prompt")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIsNSSPasswordError(t *testing.T) {
	t.Parallel()
	if !isNSSPasswordError(fmt.Errorf("SEC_ERROR_BAD_PASSWORD")) {
		t.Fatal("bad password")
	}
	if isNSSPasswordError(fmt.Errorf("SEC_ERROR_BAD_DATABASE")) {
		t.Fatal("other error")
	}
}

func mustBundle(t *testing.T) Bundle {
	t.Helper()
	b, err := Ensure(t.TempDir(), t.TempDir(), "isos.local")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func contains(ss []string, n string) bool {
	for _, s := range ss {
		if s == n {
			return true
		}
	}
	return false
}
