package cli

import (
	"io"

	"github.com/SoheilHasankhani/msc-cli/internal/completesvc"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/spf13/cobra"
)

func enhanceShellCompletion(root *cobra.Command) {
	comp, _, err := root.Find([]string{"completion"})
	if err != nil || comp == nil {
		return
	}
	wrapShellCompletion(comp, "bash", completesvc.WriteBashBrandCompleters)
	wrapShellCompletion(comp, "zsh", completesvc.WriteZshBrandCompleters)
	wrapShellCompletion(comp, "powershell", completesvc.WritePowerShellBrandCompleters)
	comp.AddCommand(newCompletionInstallCmd())
}

func wrapShellCompletion(comp *cobra.Command, shell string, write func(io.Writer, []string) error) {
	sub, _, err := comp.Find([]string{shell})
	if err != nil || sub == nil || sub.RunE == nil {
		return
	}
	orig := sub.RunE
	sub.RunE = func(cmd *cobra.Command, args []string) error {
		if err := orig(cmd, args); err != nil {
			return err
		}
		names, err := completesvc.BrandNames(paths.Default())
		if err != nil {
			return err
		}
		return write(cmd.OutOrStdout(), names)
	}
}
