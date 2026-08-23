package completesvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/spf13/cobra"
)

func TestWriteBashBrandCompleters(t *testing.T) {
	t.Parallel()
	var out strings.Builder
	if err := WriteBashBrandCompleters(&out, []string{"isos", "mores"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, `complete -o default -F __start_msc "isos"`) {
		t.Fatalf("missing isos registration:\n%s", got)
	}
	if !strings.Contains(got, `complete -o default -F __start_msc "mores"`) {
		t.Fatalf("missing mores registration:\n%s", got)
	}
}

func TestUpsertBlockReplacesExisting(t *testing.T) {
	t.Parallel()
	old := "# msc-begin shell-completion\nold\n# msc-end shell-completion\n"
	newBlock := "# msc-begin shell-completion\nnew\n# msc-end shell-completion\n"
	got := upsertBlock(old, newBlock)
	if strings.Contains(got, "old") {
		t.Fatalf("expected old block replaced:\n%s", got)
	}
	if !strings.Contains(got, "new") {
		t.Fatalf("expected new block:\n%s", got)
	}
}

func TestInstallWritesScriptsAndHooks(t *testing.T) {
	home := t.TempDir()
	dirs := paths.Resolver{GOOS: "linux", Home: home, ConfigHome: filepath.Join(home, ".config")}
	bashRC := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(bashRC, []byte("# user\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := &cobra.Command{Use: "msc"}
	root.CompletionOptions.DisableDescriptions = true
	root.InitDefaultCompletionCmd()

	res, err := Install(Options{Root: root, Dirs: dirs})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.BashScript); err != nil {
		t.Fatalf("bash script: %v", err)
	}
	if _, err := os.Stat(res.ZshScript); err != nil {
		t.Fatalf("zsh script: %v", err)
	}
	data, err := os.ReadFile(bashRC)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, hookBegin) || !strings.Contains(got, filepath.ToSlash(res.BashScript)) {
		t.Fatalf("bashrc hook missing:\n%s", got)
	}
}

func TestWritePowerShellBrandCompleters(t *testing.T) {
	t.Parallel()
	var out strings.Builder
	if err := WritePowerShellBrandCompleters(&out, []string{"isos", "msc"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, `Register-ArgumentCompleter -CommandName "isos" -ScriptBlock ${__mscCompleterBlock}`) {
		t.Fatalf("missing isos completer:\n%s", got)
	}
	if strings.Contains(got, `"msc"`) {
		t.Fatalf("must not re-register msc:\n%s", got)
	}
}

func TestInstallWritesPowerShellProfile(t *testing.T) {
	home := t.TempDir()
	docs := filepath.Join(home, "Documents")
	t.Setenv("MSC_WINDOWS_DOCUMENTS", docs)
	dirs := paths.Resolver{
		GOOS:    "windows",
		Home:    home,
		AppData: filepath.Join(home, "AppData", "Roaming"),
	}
	root := &cobra.Command{Use: "msc"}
	root.CompletionOptions.DisableDescriptions = true
	root.InitDefaultCompletionCmd()

	res, err := Install(Options{Root: root, Dirs: dirs})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.PowerShellScript); err != nil {
		t.Fatalf("powershell script: %v", err)
	}
	data, err := os.ReadFile(res.PowerShellScript)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Register-ArgumentCompleter") {
		t.Fatalf("completion.ps1 missing completer:\n%s", data)
	}
	profile := filepath.Join(docs, "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	hook, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(hook)
	if !strings.Contains(text, hookBegin) || !strings.Contains(text, res.PowerShellScript) {
		t.Fatalf("profile hook missing:\n%s", text)
	}
}
