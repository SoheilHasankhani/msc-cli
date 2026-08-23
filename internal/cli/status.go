package cli

import (
	"fmt"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/state"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show live Docker/Source state for each service",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			engine, err := dockerapi.NewEngine()
			if err != nil {
				return err
			}
			defer engine.Close()

			live, err := inferLive(cmd.Context(), p, engine)
			if err != nil {
				return err
			}

			cacheFile := state.CachePath(paths.Default().StateDir(), p.Name)
			snap, err := state.LoadCache(cacheFile)
			if err != nil {
				return err
			}
			var warnings []string
			for name, lv := range live {
				res := state.Reconcile(snap.Services[name], lv)
				if res.Drift {
					warnings = append(warnings, res.Message)
				}
			}
			if err := state.SaveCache(cacheFile, snap.SyncLive(live)); err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), state.FormatStatus(live, warnings, ui.For(cmd.OutOrStdout())))
			return err
		},
	}
}
