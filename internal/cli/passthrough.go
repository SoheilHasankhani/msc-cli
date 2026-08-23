package cli

import (
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/complete"
	"github.com/SoheilHasankhani/msc-cli/internal/passthru"
	"github.com/spf13/cobra"
)

func newComposeCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "compose [docker compose args]",
		Short:              "Run docker compose in this project's stack (thin passthrough)",
		Long:               "Resolves the Manifest compose file (and host-gateway overlay if present) then forwards every argument to docker compose. Example: isos compose logs -f doctor",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		ValidArgsFunction:  completeCompose,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			spec, err := passthru.Compose(p.Root, p.Manifest.Layout.ComposeFile, args)
			if err != nil {
				return err
			}
			return passthru.Exec(cmd.Context(), spec, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

func newGitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git [<repo>] -- [git args]",
		Short: "Run git in the meta-repo or a cloned service repo (thin passthrough)",
		Long: strings.TrimSpace(`
Runs the system git binary in one of two directories:
  • meta-repository root (Manifest, compose, nginx config) when <repo> is omitted
  • local/<repo> clone directory when <repo> is a Manifest repo name

Everything after the required "--" is forwarded unchanged to git.`),
		DisableFlagParsing: true,
		SilenceUsage:       true,
		ValidArgsFunction:  completeRepos,
		RunE: func(c *cobra.Command, args []string) error {
			if passthroughWantsHelp(args) {
				return c.Help()
			}
			p, err := resolveProject(c)
			if err != nil {
				return err
			}
			spec, err := passthru.Git(p.Root, p.ClonesDir(), complete.Repos(p.Manifest), args)
			if err != nil {
				return err
			}
			return passthru.Exec(c.Context(), spec, c.InOrStdin(), c.OutOrStdout(), c.ErrOrStderr())
		},
	}
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		_ = printGitHelp(c)
	})
	return cmd
}
