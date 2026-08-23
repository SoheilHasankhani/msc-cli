package cli

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/SoheilHasankhani/msc-cli/internal/progress"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/SoheilHasankhani/msc-cli/internal/update"
	"github.com/SoheilHasankhani/msc-cli/internal/version"
	"github.com/spf13/cobra"
)

func newSelfUpdateCmd() *cobra.Command {
	var check, force bool
	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Replace this binary with the latest GitHub Release",
		Long:  "Downloads the matching platform archive from GitHub Releases, verifies checksums.txt, and replaces the running msc binary. --check reports only. --force reinstalls even if versions already match. Override the repo with MSC_RELEASES_REPO (owner/name).",
		RunE: func(cmd *cobra.Command, args []string) error {
			ren := ui.For(cmd.OutOrStdout())
			dest, err := os.Executable()
			if err != nil {
				return err
			}
			if resolved, err := filepath.EvalSymlinks(dest); err == nil && resolved != "" {
				dest = resolved
			}
			httpClient := &http.Client{Timeout: 2 * time.Minute}
			res, err := update.Run(cmd.Context(), update.Options{
				Current:   version.Version,
				Dest:      dest,
				Force:     force,
				CheckOnly: check,
				Client:    &update.Client{HTTP: httpClient},
				Fetch:     update.HTTPFetch(httpClient),
				Progress:  progress.Options{Output: cmd.ErrOrStderr()},
			})
			if res.Message != "" {
				line := res.Message
				if res.Updated {
					line = ui.Success(ren, res.Message)
				} else if res.Skipped {
					line = ui.Dim(ren, res.Message)
				} else if check {
					line = ui.Warn(ren, res.Message)
				}
				if _, werr := fmt.Fprintln(cmd.OutOrStdout(), line); werr != nil {
					return werr
				}
			}
			if err != nil {
				return err
			}
			if res.Updated {
				refreshShellCompletionQuiet()
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "report whether an update is available without installing")
	cmd.Flags().BoolVar(&force, "force", false, "reinstall the latest release even if this binary is already current")
	return cmd
}
