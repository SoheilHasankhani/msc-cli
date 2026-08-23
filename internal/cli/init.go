package cli

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/doctor"
	"github.com/SoheilHasankhani/msc-cli/internal/elevate"
	"github.com/SoheilHasankhani/msc-cli/internal/gitops"
	"github.com/SoheilHasankhani/msc-cli/internal/initsvc"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/shim"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var repo, dest, as string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Register a meta-repo: clone, Manifest, registry, and brand shim",
		Long:  "Clones the meta-repository when --path does not exist yet (--repo required then). For an existing checkout, --repo is optional (read from git origin). Loads or drafts msc.manifest.yml (never auto-commits), registers the project, writes a brand shim, and links it onto PATH. Runs doctor --fix; does not start the stack.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ren := ui.For(cmd.OutOrStdout())
			if dest == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				dest = wd
			}
			engine := shim.ResolveEnginePath()
			dirs := paths.Default()
			res, err := initsvc.Run(cmd.Context(), initsvc.Options{
				RepoURL:      repo,
				Path:         dest,
				As:           as,
				Git:          gitops.Exec{},
				RegistryFile: dirs.RegistryFile(),
				ShimDir:      dirs.ShimDir(),
				PathDirs:     dirs,
				EnginePath:   engine,
				GOOS:         runtime.GOOS,
				Stdout:       cmd.ErrOrStderr(),
				SpinnerOut:   cmd.ErrOrStderr(),
				PromptAs:     ui.PromptProjectAlias,
				ConfirmDraft: func(path string) (bool, error) { return ui.ConfirmManifestDraft(path) },
				After: func(name, root string) error {
					p, err := project.Resolve(name, dirs)
					if err != nil {
						return err
					}
					var report doctor.Report
					err = ui.WithSpinner(cmd.Context(), cmd.ErrOrStderr(), "Running doctor --fix", func(ctx context.Context) error {
						var runErr error
						report, runErr = doctor.Run(ctx, doctor.Options{Project: p, Fix: true, Elevate: elevate.NewProcess()})
						return runErr
					})
					if err != nil {
						return err
					}
					_, _ = fmt.Fprint(cmd.OutOrStdout(), doctor.Format(report, ren))
					return nil
				},
			})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ren,
				fmt.Sprintf("registered %s → %s", res.Name, res.Path),
				fmt.Sprintf("command: %s", res.CommandPath)))
			if len(res.ShellFiles) > 0 {
				hint := "updated " + strings.Join(res.ShellFiles, ", ") + "; open a new terminal"
				if runtime.GOOS == "windows" {
					hint += " or run: . $PROFILE"
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.Dim(ren, hint))
			}
			if res.CommitHint != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.Warn(ren, res.CommitHint))
			}
			refreshShellCompletionQuiet()
			return nil
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "SSH URL of the meta-repository (required only when --path does not exist yet)")
	cmd.Flags().StringVar(&dest, "path", "", "clone/register path (default: current directory)")
	cmd.Flags().StringVar(&as, "as", "", "register under this command name (required when the brand name is taken by another project)")
	return cmd
}
