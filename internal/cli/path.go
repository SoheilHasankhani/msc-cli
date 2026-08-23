package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/shim"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/spf13/cobra"
)

func newPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Manage the msc install directory on PATH",
	}
	cmd.AddCommand(newPathInstallCmd())
	return cmd
}

func newPathInstallCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Add the msc install directory to PATH for new terminals",
		Long:  "Upserts a managed block into shell startup files (~/.bashrc, ~/.zshrc, $PROFILE). On Windows, also adds the install directory to the user Path environment variable.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs := paths.Default()
			binDir := resolveInstallBinDir(dir, dirs)
			files, err := shim.EnsureBinDirOnPATH(dirs.Home, dirs.GOOS, binDir)
			if err != nil {
				return err
			}
			ren := ui.For(cmd.OutOrStdout())
			detail := binDir
			if len(files) > 0 {
				detail = fmt.Sprintf("%s (updated %s)", binDir, strings.Join(files, ", "))
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ren, "PATH configured for new terminals", detail))
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.Dim(ren, "open a new terminal, then run: msc --version"))
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "install directory (default: MSC_INSTALL_DIR or platform default)")
	return cmd
}

func resolveInstallBinDir(flagDir string, dirs paths.Resolver) string {
	if flagDir != "" {
		return flagDir
	}
	if v := strings.TrimSpace(os.Getenv("MSC_INSTALL_DIR")); v != "" {
		return v
	}
	return dirs.BinDir()
}
