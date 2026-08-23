package hostcerts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/testenv"
)

func TestCollectHostnamesFromNginxAndDomain(t *testing.T) {
	t.Parallel()

	nginx, err := os.ReadFile(testenv.TestdataPath(t, "nginx", "default.conf"))
	if err != nil {
		t.Fatal(err)
	}
	got := CollectHostnames("isos.local", string(nginx))
	want := []string{"doctor.isos.local", "isos.local", "schedule.isos.local"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCollectHostnamesSkipsInternalServerNames(t *testing.T) {
	t.Parallel()

	got := CollectHostnames("isos.local", "server_name _;\nserver_name metrics_status_internal;\nserver_name doctor.isos.local;")
	if strings.Join(got, ",") != "doctor.isos.local,isos.local" {
		t.Fatalf("%v", got)
	}
}

func TestCollectHostnamesSkipsUnderscore(t *testing.T) {
	t.Parallel()

	got := CollectHostnames("isos.local", "server_name _;\nserver_name iam.isos.local;")
	if len(got) != 2 || got[0] != "iam.isos.local" || got[1] != "isos.local" {
		t.Fatalf("%v", got)
	}
}

func TestMissingNames(t *testing.T) {
	t.Parallel()

	hosts := "127.0.0.1 localhost\n127.0.0.1 isos.local doctor.isos.local\n"
	miss := Missing(hosts, []string{"isos.local", "doctor.isos.local", "wallet.isos.local"})
	if len(miss) != 1 || miss[0] != "wallet.isos.local" {
		t.Fatalf("%v", miss)
	}
}

func TestUpsertBlockTouchesOnlyThisProject(t *testing.T) {
	t.Parallel()

	in := strings.Join([]string{
		"127.0.0.1 localhost",
		BeginMarker("mores"),
		"127.0.0.1 mores.local",
		EndMarker("mores"),
		BeginMarker("isos"),
		"127.0.0.1 isos.local",
		EndMarker("isos"),
		"",
	}, "\n")

	out := UpsertBlock(in, "isos", []string{"isos.local", "doctor.isos.local"})
	if !strings.Contains(out, "127.0.0.1 mores.local") {
		t.Fatalf("mores block was changed:\n%s", out)
	}
	if !strings.Contains(out, BeginMarker("isos")) || !strings.Contains(out, "doctor.isos.local") {
		t.Fatalf("isos block not updated:\n%s", out)
	}
	if strings.Count(out, BeginMarker("isos")) != 1 {
		t.Fatalf("duplicate isos block:\n%s", out)
	}
}

func TestWriteFileUpsertsOnlyThisProject(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "hosts")
	initial := "127.0.0.1 localhost\n" + BeginMarker("mores") + "\n127.0.0.1 mores.local\n" + EndMarker("mores") + "\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, "isos", []string{"isos.local", "doctor.isos.local"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "mores.local") || !strings.Contains(text, "doctor.isos.local") {
		t.Fatalf("%s", text)
	}
	if err := WriteFile(path, "isos", []string{"isos.local", "wallet.isos.local"}); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text = string(got)
	if !strings.Contains(text, "wallet.isos.local") || strings.Count(text, BeginMarker("isos")) != 1 {
		t.Fatalf("%s", text)
	}
}

func TestUpsertBlockAppendsWhenMissing(t *testing.T) {
	t.Parallel()

	out := UpsertBlock("127.0.0.1 localhost\n", "isos", []string{"isos.local"})
	if !strings.Contains(out, BeginMarker("isos")) || !strings.Contains(out, "127.0.0.1 isos.local") {
		t.Fatalf("%s", out)
	}
}

func TestSystemHostsPath(t *testing.T) {
	t.Parallel()
	if SystemHostsPath("linux") != "/etc/hosts" {
		t.Fatal("linux")
	}
	if SystemHostsPath("windows") != `C:\Windows\System32\drivers\etc\hosts` {
		t.Fatal("windows")
	}
}
