package cli

import (
	"fmt"
	"time"

	"github.com/SoheilHasankhani/msc-cli/internal/gitops"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/progress"
	"github.com/SoheilHasankhani/msc-cli/internal/syncsvc"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var list, refresh, cloneOnly, pullOnly bool
	var all bool // legacy alias for --clone-only
	cmd := &cobra.Command{
		Use:               "sync [REPO...]",
		Short:             "Sync accessible service repos over SSH (clone missing, pull updates)",
		ValidArgsFunction: completeRepos,
		Long: `Probes Git host access with git ls-remote (SSH only; no tokens). Results are cached for 7 days unless --refresh is passed.

Default: clone repos you can access but have not cloned yet, and git pull --ff-only repos already on disk. Repos you cannot access are skipped (never cloned, never pulled). Revoked access on an existing clone is warned once and left on disk.

Use --list to inspect without changing anything. Use --clone-only or --pull-only for advanced workflows.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ren := ui.For(cmd.OutOrStdout())
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			opt := syncsvc.Options{
				Project:   p,
				Git:       gitops.Exec{},
				CachePath: gitops.AccessCachePath(paths.Default().StateDir(), p.Name),
				Now:       time.Now(),
				Refresh:   refresh,
				ListOnly:  list,
				CloneOnly: cloneOnly || all,
				PullOnly:  pullOnly,
				Names:     args,
				Progress:  progress.Options{Output: cmd.ErrOrStderr()},
			}
			if list {
				plan, err := syncsvc.Plan(cmd.Context(), opt)
				if err != nil {
					return err
				}
				_, err = fmt.Fprint(cmd.OutOrStdout(), syncsvc.FormatPlan(plan, ren))
				return err
			}
			if pullOnly {
				res, err := syncsvc.Pull(cmd.Context(), opt)
				if err != nil {
					return err
				}
				_, err = fmt.Fprint(cmd.OutOrStdout(), syncsvc.FormatPullResult(res, ren))
				return err
			}
			if cloneOnly || all {
				n, err := syncsvc.Clone(cmd.Context(), opt)
				if err != nil {
					return err
				}
				_, err = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ren, fmt.Sprintf("cloned %d repo(s)", n), ""))
				return err
			}
			plan, res, err := syncsvc.Update(cmd.Context(), opt)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), syncsvc.FormatUpdateResult(plan, res, ren))
			return err
		},
	}
	cmd.Flags().BoolVar(&list, "list", false, "list cloned and available repos without cloning or pulling")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "ignore the cached access probe and check ls-remote again")
	cmd.Flags().BoolVar(&cloneOnly, "clone-only", false, "clone accessible repos only (no pull)")
	cmd.Flags().BoolVar(&pullOnly, "pull-only", false, "pull cloned repos only (no clone)")
	cmd.Flags().BoolVar(&all, "all", false, "deprecated alias for --clone-only")
	cmd.Flags().BoolVar(&pullOnly, "pull", false, "deprecated alias for --pull-only")
	_ = cmd.Flags().MarkHidden("all")
	_ = cmd.Flags().MarkHidden("pull")
	return cmd
}
