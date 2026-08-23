package cli

import (
	"fmt"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/completesvc"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/spf13/cobra"
)

func installShellCompletion() (completesvc.Result, error) {
	return completesvc.Install(completesvc.Options{
		Root: newRootCmdEngine(),
		Dirs: paths.Default(),
	})
}

func newCompletionInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install shell completion for new terminals",
		Long:  "Writes cached bash/zsh/PowerShell completion scripts under the msc config directory and upserts a managed block into your shell startup files (~/.bashrc, ~/.zshrc, $PROFILE). Run automatically on init and install.",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := installShellCompletion()
			if err != nil {
				return err
			}
			ren := ui.For(cmd.OutOrStdout())
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ren,
				"shell completion installed",
				fmt.Sprintf("bash: %s", res.BashScript)))
			if res.ZshScript != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.Dim(ren, fmt.Sprintf("zsh: %s", res.ZshScript)))
			}
			if res.PowerShellScript != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.Dim(ren, fmt.Sprintf("powershell: %s", res.PowerShellScript)))
			}
			if len(res.RCFiles) > 0 {
				reload := "source " + res.BashScript
				if res.PowerShellScript != "" {
					reload = ". $PROFILE"
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.Dim(ren,
					"updated "+strings.Join(res.RCFiles, ", ")+"; open a new terminal or run: "+reload))
			}
			return nil
		},
	}
}

func refreshShellCompletionQuiet() {
	if _, err := installShellCompletion(); err != nil {
		return
	}
}
