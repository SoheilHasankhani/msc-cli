package hostcerts

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLeafBaseFromDomain(t *testing.T) {
	t.Parallel()
	if LeafBase("isos.local") != "isos-local" {
		t.Fatal(LeafBase("isos.local"))
	}
}

func TestEnsureCreatesMachineCAAndDomainLeaf(t *testing.T) {
	t.Parallel()

	machine, project := t.TempDir(), t.TempDir()
	b, err := Ensure(machine, project, "isos.local")
	if err != nil {
		t.Fatal(err)
	}
	if b.CACrt != filepath.Join(machine, "local-ca.crt") || b.LeafCrt != filepath.Join(project, "isos-local.crt") {
		t.Fatalf("%+v", b)
	}
	if b.CACopy != filepath.Join(project, "local-ca.crt") {
		t.Fatalf("CA copy: %s", b.CACopy)
	}
	if err := Valid(b, "isos.local"); err != nil {
		t.Fatal(err)
	}

	ca := mustParseCert(t, mustRead(t, b.CACrt))
	leaf := mustParseCert(t, mustRead(t, b.LeafCrt))
	if !ca.IsCA {
		t.Fatal("CA must be a CA")
	}
	if ca.Subject.CommonName != "msc local Root CA" {
		t.Fatalf("CA CN: %s", ca.Subject.CommonName)
	}
	if err := leaf.CheckSignatureFrom(ca); err != nil {
		t.Fatal(err)
	}
	if err := leaf.VerifyHostname("isos.local"); err != nil {
		t.Fatal(err)
	}
	if err := leaf.VerifyHostname("doctor.isos.local"); err != nil {
		t.Fatal(err)
	}
	if mustRead(t, b.CACopy) == nil || string(mustRead(t, b.CACopy)) != string(mustRead(t, b.CACrt)) {
		t.Fatal("project local-ca.crt must match the machine CA")
	}
	st, err := os.Stat(b.CAKey)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && st.Mode().Perm() != 0o600 {
		t.Fatalf("CA key mode: %v", st.Mode())
	}
}

func TestEnsureRestoresProjectCACopy(t *testing.T) {
	t.Parallel()

	machine, project := t.TempDir(), t.TempDir()
	b, err := Ensure(machine, project, "isos.local")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b.CACopy, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Ensure(machine, project, "isos.local"); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, b.CACopy)) != string(mustRead(t, b.CACrt)) {
		t.Fatal("stale project local-ca.crt was not refreshed")
	}
}

func TestEnsureSharesMachineCAAcrossProjects(t *testing.T) {
	t.Parallel()

	machine := t.TempDir()
	first, err := Ensure(machine, t.TempDir(), "isos.local")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Ensure(machine, t.TempDir(), "mores.local")
	if err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, first.CACrt)) != string(mustRead(t, second.CACrt)) {
		t.Fatal("projects must share the machine CA")
	}
	if err := Valid(second, "mores.local"); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureReusesValidCA(t *testing.T) {
	t.Parallel()

	machine, project := t.TempDir(), t.TempDir()
	first, err := Ensure(machine, project, "isos.local")
	if err != nil {
		t.Fatal(err)
	}
	ca1 := mustRead(t, first.CACrt)
	second, err := Ensure(machine, project, "isos.local")
	if err != nil {
		t.Fatal(err)
	}
	if string(ca1) != string(mustRead(t, second.CACrt)) {
		t.Fatal("valid CA was regenerated")
	}
}

func TestEnsureRewritesLeafWhenDomainChanges(t *testing.T) {
	t.Parallel()

	machine, project := t.TempDir(), t.TempDir()
	if _, err := Ensure(machine, project, "isos.local"); err != nil {
		t.Fatal(err)
	}
	b, err := Ensure(machine, project, "mores.local")
	if err != nil {
		t.Fatal(err)
	}
	if err := Valid(b, "mores.local"); err != nil {
		t.Fatal(err)
	}
	leaf := mustParseCert(t, mustRead(t, b.LeafCrt))
	if err := leaf.VerifyHostname("iam.mores.local"); err != nil {
		t.Fatal(err)
	}
}

func TestValidFailsWhenProjectCACopyMissing(t *testing.T) {
	t.Parallel()

	machine, project := t.TempDir(), t.TempDir()
	b, err := Ensure(machine, project, "isos.local")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(b.CACopy); err != nil {
		t.Fatal(err)
	}
	if err := Valid(b, "isos.local"); err == nil {
		t.Fatal("expected missing copy to fail Valid")
	}
}

func TestFileFingerprintStable(t *testing.T) {
	t.Parallel()

	b := mustBundle(t)
	a, err := FileFingerprint(b.CACrt)
	if err != nil || a == "" {
		t.Fatal(err)
	}
	again, err := FileFingerprint(b.CACrt)
	if err != nil || again != a {
		t.Fatalf("%s vs %s", a, again)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustParseCert(t *testing.T, pemBytes []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		t.Fatal("no PEM")
	}
	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
