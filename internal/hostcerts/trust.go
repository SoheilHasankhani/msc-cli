package hostcerts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	linuxCAFilename = "msc-local-ca.crt"
	nssNickname     = "msc-local-ca"
	trustStampName  = ".os-trust-fingerprint"
)

// TrustPlan is the privileged OS trust-store install for the machine CA.
type TrustPlan struct {
	Dest      string
	Tool      string
	Args      []string
	UpdateCmd string
}

// LinuxCADest is where update-ca-certificates picks up the shared machine CA.
func LinuxCADest() string {
	return "/usr/local/share/ca-certificates/" + linuxCAFilename
}

// NSSNickname is the Chrome/Chromium NSS nickname for the shared machine CA.
func NSSNickname() string {
	return nssNickname
}

// OSTrustPlan returns the copy destination / tool invocation for goos.
// DestPath in the payload is only used on Linux (copy then update-ca-certificates).
func OSTrustPlan(goos, caPath string) TrustPlan {
	switch goos {
	case "windows":
		return TrustPlan{Tool: "certutil", Args: []string{"-addstore", "-f", "ROOT", caPath}}
	case "darwin":
		return TrustPlan{
			Tool: "security",
			Args: []string{"add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", caPath},
		}
	default:
		return TrustPlan{
			Dest:      LinuxCADest(),
			Tool:      "cp",
			Args:      []string{caPath, LinuxCADest()},
			UpdateCmd: "update-ca-certificates",
		}
	}
}

// TrustStampPath is the last successfully installed CA fingerprint (non-Linux
// stores are not readable without elevation).
func TrustStampPath(machineDir string) string {
	return filepath.Join(machineDir, trustStampName)
}

// WriteTrustStamp records the fingerprint of the CA that was installed.
func WriteTrustStamp(machineDir, fingerprint string) error {
	if err := os.MkdirAll(machineDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(TrustStampPath(machineDir), []byte(strings.TrimSpace(fingerprint)+"\n"), 0o644)
}

// ReadTrustStamp returns the last installed CA fingerprint, or empty.
func ReadTrustStamp(machineDir string) string {
	data, err := os.ReadFile(TrustStampPath(machineDir))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// OSTrustMatches reports whether the current machine CA is what the OS was
// last told to trust. Linux compares the system dest file (source of truth).
// Darwin/Windows compare the stamp written after a successful install.
func OSTrustMatches(goos, caPath, machineDir string) bool {
	fp, err := FileFingerprint(caPath)
	if err != nil {
		return false
	}
	if goos == "linux" {
		installed, err := FileFingerprint(LinuxCADest())
		if err == nil {
			return installed == fp
		}
		return false
	}
	return ReadTrustStamp(machineDir) == fp
}

// CommandRunner runs a process. Tests inject a fake.
type CommandRunner func(name string, args ...string) error

// InstallNSS adds (or replaces) the CA in the user Chrome/Chromium NSS db
// (~/.pki/nssdb). run is non-interactive. prompt is used only when the existing
// DB has a password (one unlock prompt — never "set a new password").
func InstallNSS(home, caPath string, run, prompt CommandRunner) error {
	if run == nil {
		return fmt.Errorf("NSS install requires certutil (libnss3-tools)")
	}
	if _, err := os.Stat(caPath); err != nil {
		return fmt.Errorf("CA file: %w", err)
	}
	db := filepath.Join(home, ".pki", "nssdb")
	if err := os.MkdirAll(db, 0o700); err != nil {
		return err
	}
	sql := "sql:" + db
	if !nssDBInitialized(db) {
		if err := run("certutil", "-N", "-d", sql, "--empty-password"); err != nil {
			return fmt.Errorf("create NSS db: %w", err)
		}
	}
	pw, cleanup, err := writeEmptyNSSPassword()
	if err != nil {
		return err
	}
	defer cleanup()
	// Drop the previous nickname so a rotated CA is not treated as "already present".
	_ = run("certutil", "-D", "-d", sql, "-n", nssNickname, "-f", pw)
	add := []string{"-A", "-d", sql, "-t", "C,,", "-n", nssNickname, "-i", caPath}
	if err := run("certutil", append(add, "-f", pw)...); err == nil || isNSSAlreadyPresent(err) {
		return nil
	} else if !isNSSPasswordError(err) {
		return err
	}
	if prompt == nil {
		return fmt.Errorf("chrome NSS db has a password — re-run doctor --fix in a terminal and enter the existing NSS PIN (not a new password)")
	}
	_ = prompt("certutil", "-D", "-d", sql, "-n", nssNickname)
	return prompt("certutil", add...)
}

func nssDBInitialized(dir string) bool {
	for _, name := range []string{"cert9.db", "cert8.db", "key4.db", "key3.db"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func writeEmptyNSSPassword() (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "msc-nss-pass-*")
	if err != nil {
		return "", nil, err
	}
	path = f.Name()
	_, err = f.Write([]byte("\n"))
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(path)
		return "", nil, err
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", nil, closeErr
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = os.Remove(path)
		return "", nil, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func isNSSAlreadyPresent(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "already in database") ||
		strings.Contains(s, "already exists") ||
		(strings.Contains(s, "nickname") && strings.Contains(s, "exists"))
}

func isNSSPasswordError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "sec_error_bad_database") {
		return false
	}
	return strings.Contains(s, "password") ||
		strings.Contains(s, "sec_error_bad_password") ||
		strings.Contains(s, "authentication failed")
}

// ExecRunner is the non-interactive production CommandRunner.
func ExecRunner(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w (%s)", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// InteractiveRunner attaches the terminal so certutil can ask for an existing NSS PIN.
func InteractiveRunner(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// LookPathNSS reports whether certutil is on PATH.
func LookPathNSS() error {
	_, err := exec.LookPath("certutil")
	return err
}
