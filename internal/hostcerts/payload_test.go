package hostcerts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPayloadWriteHosts(t *testing.T) {
	t.Parallel()

	hosts := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(hosts, []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := Payload{
		Op:        OpWriteHosts,
		Project:   "isos",
		Names:     []string{"isos.local", "iam.isos.local"},
		HostsPath: hosts,
	}
	if err := ApplyPayload(p); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(hosts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), BeginMarker("isos")) || !strings.Contains(string(got), "iam.isos.local") {
		t.Fatal(string(got))
	}
}

func TestApplyPayloadRejectsUnknownOp(t *testing.T) {
	t.Parallel()
	if err := ApplyPayload(Payload{Op: "rm-rf"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyPayloadInstallCACopiesPEM(t *testing.T) {
	t.Parallel()

	b, err := Ensure(t.TempDir(), t.TempDir(), "isos.local")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "msc-isos.crt")
	p := Payload{Op: OpInstallCA, Project: "isos", CAPath: b.CACrt, DestPath: dest}
	if err := ApplyPayload(p); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(b.CACrt)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("CA not copied")
	}
}

func TestReadWritePayloadRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "p.json")
	in := Payload{Op: OpWriteHosts, Project: "isos", Names: []string{"isos.local"}}
	if err := WritePayloadFile(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := ReadPayloadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.Op != in.Op || out.Project != in.Project || strings.Join(out.Names, ",") != "isos.local" {
		t.Fatalf("%+v", out)
	}
}

func TestApplyPayloadInstallCARejectsNonPEM(t *testing.T) {
	t.Parallel()
	src := filepath.Join(t.TempDir(), "not.crt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := ApplyPayload(Payload{Op: OpInstallCA, Project: "isos", CAPath: src, DestPath: filepath.Join(t.TempDir(), "out.crt")})
	if err == nil {
		t.Fatal("expected error")
	}
}
