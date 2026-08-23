// Package cli owns Cobra command wiring only. Business logic lives in sibling
// internal packages; this package parses flags, calls those packages, and
// formats output.
package cli

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/SoheilHasankhani/msc-cli/internal/logging"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/SoheilHasankhani/msc-cli/internal/version"
	"github.com/spf13/cobra"
)

var logCloser io.Closer

// Execute runs the root command and flushes the local JSON log file.
func Execute() error {
	err := NewRootCmd().Execute()
	if err != nil {
		slog.Error("command failed", "err", err.Error())
	}
	if logCloser != nil {
		_ = logCloser.Close()
		logCloser = nil
	}
	return err
}

// NewRootCmd constructs the command tree, applying brand shim mode when MSC_PROJECT is set.
func NewRootCmd() *cobra.Command {
	cmd := newRootCmdEngine()
	applyBrandMode(cmd)
	return cmd
}

// newRootCmdEngine builds the neutral msc command tree (used for completion generation).
func newRootCmdEngine() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "msc",
		Short:            "Brand-agnostic engine for local microservice development environments",
		Long:             "msc is a single shared engine. Brand commands (isos, mores, ...) are shims that set MSC_PROJECT and exec this binary.",
		Version:          version.String(),
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true, // parse --project before compose/git DisableFlagParsing (direct msc usage)
	}

	cmd.SetVersionTemplate("{{.Version}}\n")
	cmd.PersistentFlags().String("project", "", "registered project name (engine-only; brand shims set MSC_PROJECT)")
	cmd.PersistentFlags().Bool("verbose", false, "write debug-level JSON logs to the local log file")
	cmd.PersistentPreRun = func(c *cobra.Command, args []string) {
		logCloser = startFileLog(c)
		slog.Info("command", "name", c.CommandPath(), "args", args, "version", version.Version)
	}

	cmd.AddCommand(
		newStatusCmd(), newUpCmd(), newDownCmd(), newSwitchCmd(), newSyncCmd(),
		newDoctorCmd(), newInitCmd(), newProjectsCmd(), newComposeCmd(), newGitCmd(),
		newSelfUpdateCmd(), newSupportBundleCmd(), newElevatedDoCmd(), newPathCmd(),
	)
	cmd.CompletionOptions.DisableDescriptions = true
	cmd.InitDefaultCompletionCmd()
	enhanceShellCompletion(cmd)

	defaultHelp := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if c == cmd {
			ren := ui.For(c.OutOrStdout())
			_, _ = fmt.Fprint(c.OutOrStdout(), ui.HelpBanner(ren, commandName(c), c.Short))
		}
		defaultHelp(c, args)
	})

	return cmd
}

func startFileLog(cmd *cobra.Command) io.Closer {
	if flag.Lookup("test.v") != nil {
		return nil
	}
	dir := os.Getenv("MSC_LOG_DIR")
	if dir == "" {
		dir = paths.Default().LogDir()
	}
	level := logging.ParseLevel(os.Getenv("MSC_LOG_LEVEL"))
	if verbose, err := cmd.Root().PersistentFlags().GetBool("verbose"); err == nil && verbose {
		level = slog.LevelDebug
	}
	lg, closer, err := logging.New(logging.Options{Dir: dir, Level: level})
	if err != nil {
		return nil
	}
	slog.SetDefault(lg)
	return closer
}
