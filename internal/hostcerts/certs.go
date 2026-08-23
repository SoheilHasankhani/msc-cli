package hostcerts

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MachineCABase = "local-ca"
	ProjectCACopy = "local-ca.crt"
	caYears       = 10
	leafDays      = 825
	rsaBits       = 2048
	validSkew     = 30 * 24 * time.Hour
)

// Bundle is the machine CA plus one project's wildcard leaf and CA copy.
type Bundle struct {
	MachineDir string
	Dir        string
	CACrt      string
	CAKey      string
	CACopy     string
	LeafCrt    string
	LeafKey    string
}

// LeafBase turns local_domain into the leaf filename (isos.local → isos-local).
func LeafBase(localDomain string) string {
	return strings.ReplaceAll(strings.TrimSpace(localDomain), ".", "-")
}

// Dir is layout.config_dir/certificates.
func Dir(configDir string) string {
	return filepath.Join(configDir, "certificates")
}

// MachineDir is the engine config certs directory (local-ca.crt / local-ca.key).
func MachineDir(engineConfigDir string) string {
	return filepath.Join(engineConfigDir, "certs")
}

// Paths returns machine CA paths plus the project leaf and CA copy.
func Paths(machineDir, projectDir, localDomain string) Bundle {
	leaf := LeafBase(localDomain)
	return Bundle{
		MachineDir: machineDir,
		Dir:        projectDir,
		CACrt:      filepath.Join(machineDir, MachineCABase+".crt"),
		CAKey:      filepath.Join(machineDir, MachineCABase+".key"),
		CACopy:     filepath.Join(projectDir, ProjectCACopy),
		LeafCrt:    filepath.Join(projectDir, leaf+".crt"),
		LeafKey:    filepath.Join(projectDir, leaf+".key"),
	}
}

// Ensure writes a machine-level CA (reused across projects) and a wildcard leaf
// for localDomain. The public CA is always copied to projectDir/local-ca.crt so
// compose can bind-mount a repo-relative path.
func Ensure(machineDir, projectDir, localDomain string) (Bundle, error) {
	if strings.TrimSpace(localDomain) == "" {
		return Bundle{}, fmt.Errorf("local_domain is required to generate certificates")
	}
	if err := os.MkdirAll(machineDir, 0o755); err != nil {
		return Bundle{}, err
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return Bundle{}, err
	}
	b := Paths(machineDir, projectDir, localDomain)
	if err := ensureCA(b); err != nil {
		return Bundle{}, err
	}
	if err := validLeaf(b, localDomain); err != nil {
		if err := writeLeaf(b, localDomain); err != nil {
			return Bundle{}, err
		}
	}
	if err := publishCACopy(b); err != nil {
		return Bundle{}, err
	}
	if err := Valid(b, localDomain); err != nil {
		return Bundle{}, err
	}
	return b, nil
}

func publishCACopy(b Bundle) error {
	if samePath(b.CACrt, b.CACopy) {
		return nil
	}
	data, err := os.ReadFile(b.CACrt)
	if err != nil {
		return err
	}
	return os.WriteFile(b.CACopy, data, 0o644)
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// Valid reports whether the machine CA signed a wildcard leaf for localDomain,
// the bundle is not near expiry, and the project CA copy matches the machine CA.
func Valid(b Bundle, localDomain string) error {
	if err := validLeaf(b, localDomain); err != nil {
		return err
	}
	return caCopyMatches(b)
}

func validLeaf(b Bundle, localDomain string) error {
	ca, err := parseCertFile(b.CACrt)
	if err != nil {
		return err
	}
	if !ca.IsCA {
		return fmt.Errorf("local-ca.crt is not a CA")
	}
	if time.Now().After(ca.NotAfter.Add(-validSkew)) {
		return fmt.Errorf("local CA is expired or near expiry")
	}
	leaf, err := parseCertFile(b.LeafCrt)
	if err != nil {
		return err
	}
	if err := leaf.CheckSignatureFrom(ca); err != nil {
		return fmt.Errorf("leaf is not signed by the local CA: %w", err)
	}
	if time.Now().After(leaf.NotAfter.Add(-validSkew)) {
		return fmt.Errorf("wildcard leaf is expired or near expiry")
	}
	if err := leaf.VerifyHostname(localDomain); err != nil {
		return err
	}
	if err := leaf.VerifyHostname("svc." + localDomain); err != nil {
		return fmt.Errorf("leaf is not a wildcard for %s: %w", localDomain, err)
	}
	if _, err := os.Stat(b.CAKey); err != nil {
		return fmt.Errorf("CA private key missing: %w", err)
	}
	if _, err := os.Stat(b.LeafKey); err != nil {
		return fmt.Errorf("leaf private key missing: %w", err)
	}
	return nil
}

func caCopyMatches(b Bundle) error {
	if samePath(b.CACrt, b.CACopy) {
		return nil
	}
	want, err := FileFingerprint(b.CACrt)
	if err != nil {
		return err
	}
	got, err := FileFingerprint(b.CACopy)
	if err != nil {
		return fmt.Errorf("project local-ca.crt is missing or unreadable: %w", err)
	}
	if got != want {
		return fmt.Errorf("project local-ca.crt does not match the machine CA")
	}
	return nil
}

func ensureCA(b Bundle) error {
	if ca, err := parseCertFile(b.CACrt); err == nil && ca.IsCA && time.Now().Before(ca.NotAfter.Add(-validSkew)) {
		if _, err := os.Stat(b.CAKey); err == nil {
			return nil
		}
	}
	return writeCA(b)
}

func writeCA(b Bundle) error {
	key, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return err
	}
	serial, err := randSerial()
	if err != nil {
		return err
	}
	now := time.Now().Add(-time.Hour)
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"msc local"},
			CommonName:   "msc local Root CA",
		},
		NotBefore:             now,
		NotAfter:              now.AddDate(caYears, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
		SubjectKeyId:          ski(&key.PublicKey),
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	if err := writePEM(b.CACrt, "CERTIFICATE", der, 0o644); err != nil {
		return err
	}
	return writeKey(b.CAKey, key)
}

func writeLeaf(b Bundle, localDomain string) error {
	ca, err := parseCertFile(b.CACrt)
	if err != nil {
		return err
	}
	caKey, err := parseKeyFile(b.CAKey)
	if err != nil {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return err
	}
	serial, err := randSerial()
	if err != nil {
		return err
	}
	now := time.Now().Add(-time.Hour)
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"msc local"},
			CommonName:   localDomain,
		},
		NotBefore:             now,
		NotAfter:              now.Add(leafDays * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{localDomain, "*." + localDomain},
		SubjectKeyId:          ski(&key.PublicKey),
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, ca, &key.PublicKey, caKey)
	if err != nil {
		return err
	}
	if err := writePEM(b.LeafCrt, "CERTIFICATE", der, 0o644); err != nil {
		return err
	}
	return writeKey(b.LeafKey, key)
}

func parseCertFile(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseCertificatePEM(data)
}

// ParseCertificatePEM decodes the first CERTIFICATE PEM block.
func ParseCertificatePEM(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("not a PEM certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}

// FileFingerprint is the SHA-256 of the certificate DER (hex).
func FileFingerprint(path string) (string, error) {
	cert, err := parseCertFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:]), nil
}

func parseKeyFile(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("not a PEM private key")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("CA key is not RSA")
	}
	return rsaKey, nil
}

func writeKey(path string, key *rsa.PrivateKey) error {
	return writePEM(path, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key), 0o600)
}

func writePEM(path, typ string, der []byte, mode os.FileMode) error {
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), mode)
}

func randSerial() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

func ski(pub *rsa.PublicKey) []byte {
	sum := sha1.Sum(x509.MarshalPKCS1PublicKey(pub))
	return sum[:]
}
