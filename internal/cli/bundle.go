package cli

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/SoheilHasankhani/msc-cli/internal/logging"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/SoheilHasankhani/msc-cli/internal/version"
	"github.com/spf13/cobra"
)

func newSupportBundleCmd() *cobra.Command {
	var dest string
	cmd := &cobra.Command{
		Use:   "support-bundle",
		Short: "Zip recent local JSON logs for sharing",
		Long:  "Writes a zip of ~/.config/msc/logs (JSON-lines only) plus a small meta.json. Logs stay on this machine until you send the zip. No registry, Manifest, or secrets are included.",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := os.Getenv("MSC_LOG_DIR")
			if dir == "" {
				dir = paths.Default().LogDir()
			}
			path, err := logging.WriteBundle(dir, dest, time.Now().UTC(), map[string]string{
				"version": version.Version,
				"commit":  version.Commit,
				"os":      runtime.GOOS,
				"arch":    runtime.GOARCH,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ui.For(cmd.OutOrStdout()), "Support bundle written", path))
			return err
		},
	}
	cmd.Flags().StringVarP(&dest, "output", "o", "", "zip file or directory (default: ./msc-support-<timestamp>.zip)")
	return cmd
}
