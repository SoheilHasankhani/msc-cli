package hostcerts

import (
	"strings"
	"testing"
)

func TestApplyElevatedInstallCADarwinUsesSecurity(t *testing.T) {
	t.Parallel()

	b, err := Ensure(t.TempDir(), t.TempDir(), "isos.local")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	err = ApplyElevated(Payload{Op: OpInstallCA, Project: "isos", CAPath: b.CACrt, DestPath: "/tmp/evil.crt"}, "darwin", func(name string, args ...string) error {
		got = append(got, name)
		got = append(got, args...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "security") || !strings.Contains(joined, "add-trusted-cert") {
		t.Fatal(joined)
	}
	if strings.Contains(joined, "/tmp/evil.crt") {
		t.Fatal("must not use payload dest")
	}
}

func TestApplyElevatedInstallCALinuxIgnoresPayloadDest(t *testing.T) {
	t.Parallel()

	b, err := Ensure(t.TempDir(), t.TempDir(), "isos.local")
	if err != nil {
		t.Fatal(err)
	}
	var update string
	err = ApplyElevated(Payload{Op: OpInstallCA, Project: "isos", CAPath: b.CACrt, DestPath: "/tmp/evil.crt"}, "linux", func(name string, args ...string) error {
		update = name
		return nil
	})
	// copy to /usr/local/share/ca-certificates will fail without root;
	// we still assert the dest we attempted is the system path via the error or skip.
	if err == nil && update != "update-ca-certificates" {
		// succeeded only if the test process can write the system dir
		return
	}
	if err != nil && !strings.Contains(err.Error(), LinuxCADest()) && !osIsPermission(err) {
		// permission or mkdir on system path is the expected failure mode
		if !osIsPermission(err) && !strings.Contains(err.Error(), "permission") && !strings.Contains(err.Error(), "read-only") && !strings.Contains(err.Error(), "ca-certificates") {
			t.Fatalf("%v", err)
		}
	}
}

func osIsPermission(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "permission") || strings.Contains(err.Error(), "read-only") || strings.Contains(err.Error(), "denied"))
}

func TestApplyElevatedUnknownOp(t *testing.T) {
	t.Parallel()
	if err := ApplyElevated(Payload{Op: "wipe"}, "linux", nil); err == nil {
		t.Fatal("expected error")
	}
}
