package cli

import (
	"fmt"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/nginxcfg"
	"github.com/SoheilHasankhani/msc-cli/internal/progress"
	"github.com/SoheilHasankhani/msc-cli/internal/stack"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	var noPull, pullOnly bool
	var profiles []string
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Pull stack images (with progress) and start docker compose",
		Long: `Pulls each service image, then runs docker compose up -d on the same per-service progress rows.

Compose profiles come from layout.compose_profile in msc.manifest.yml unless --profile is passed.
Pull uses the docker CLI credential store (same as docker compose pull).
Use --pull-only to pull without starting the stack. Use --no-pull to skip pull.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noPull && pullOnly {
				return fmt.Errorf("--pull-only and --no-pull cannot be used together")
			}
			ren := ui.For(cmd.OutOrStdout())
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			if _, _, err := nginxcfg.EnsureHostGateway(p.Root, p.Manifest.Layout.ComposeFile); err != nil {
				return err
			}
			if err := nginxcfg.EnsureOverlay(nginxcfg.GeneratedFile(p.ConfigDir()), nginxcfg.OverlayServices(p.Manifest)); err != nil {
				return err
			}
			runOpts := stack.RunOpts{Profiles: profiles}
			compose := dockerapi.ExecCompose{}
			prog := progress.Options{Output: cmd.ErrOrStderr()}
			if err := stack.Up(cmd.Context(), p, compose, prog, stack.UpOpts{
				RunOpts:  runOpts,
				SkipPull: noPull,
				PullOnly: pullOnly,
			}); err != nil {
				return err
			}
			if !pullOnly {
				if err := refreshProjectCache(cmd.Context(), p); err != nil {
					return err
				}
			}
			hint := ""
			if profile := stack.ProfileLabel(p, runOpts); profile != "" {
				hint = "profile: " + profile
			}
			msg := "stack is up"
			if pullOnly {
				msg = "images pulled"
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ren, msg, hint))
			return err
		},
	}
	cmd.Flags().BoolVar(&noPull, "no-pull", false, "skip pulling images; use local images only")
	cmd.Flags().BoolVar(&pullOnly, "pull-only", false, "pull service images only; do not start the stack")
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "compose profile(s) to enable (default from msc.manifest.yml)")
	return cmd
}

func newDownCmd() *cobra.Command {
	var profiles []string
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Stop the project's docker compose stack",
		Long:  "Stops services for the manifest compose profile (layout.compose_profile) unless --profile is passed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ren := ui.For(cmd.OutOrStdout())
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			runOpts := stack.RunOpts{Profiles: profiles}
			compose := dockerapi.ExecCompose{}
			prog := progress.Options{Output: cmd.ErrOrStderr()}
			if err := stack.Down(cmd.Context(), p, compose, prog, stack.DownOpts{RunOpts: runOpts}); err != nil {
				return err
			}
			if err := refreshProjectCache(cmd.Context(), p); err != nil {
				return err
			}
			hint := ""
			if profile := stack.ProfileLabel(p, runOpts); profile != "" {
				hint = "profile: " + profile
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ren, "stack is down", hint))
			return err
		},
	}
	cmd.Flags().StringSliceVar(&profiles, "profile", nil, "compose profile(s) to stop (default from msc.manifest.yml)")
	return cmd
}
