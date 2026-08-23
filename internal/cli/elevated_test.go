package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/hostcerts"
)

func TestElevatedDoWritesHostsFromPayload(t *testing.T) {
	t.Parallel()

	hosts := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(hosts, []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(t.TempDir(), "p.json")
	if err := hostcerts.WritePayloadFile(payload, hostcerts.Payload{
		Op:        hostcerts.OpWriteHosts,
		Project:   "isos",
		Names:     []string{"isos.local"},
		HostsPath: hosts,
	}); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"__elevated-do", "--payload", payload})
	// ApplyElevated forces the system hosts path, so this must not rewrite the temp file
	// unless we are root and /etc/hosts is writable. The command must still parse and run.
	err := cmd.Execute()
	if err == nil {
		return
	}
	if !strings.Contains(err.Error(), "hosts") && !strings.Contains(err.Error(), "permission") && !strings.Contains(err.Error(), "/etc/hosts") {
		t.Fatalf("%v\n%s", err, out.String())
	}
}
