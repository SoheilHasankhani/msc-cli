package cli

import (
	"fmt"

	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/spf13/cobra"
)

// Engine-only subcommands are not meaningful from a brand shim (isos, mores, …).
var engineOnlyCommands = []string{
	"init",
	"projects",
	"self-update",
	"support-bundle",
	"completion",
	"path",
}

func applyBrandMode(cmd *cobra.Command) {
	brand := project.FromEnv()
	if brand == "" {
		return
	}
	_ = cmd.PersistentFlags().Set("project", brand)
	if f := cmd.PersistentFlags().Lookup("project"); f != nil {
		f.Hidden = true
	}
	cmd.Use = brand
	cmd.CompletionOptions.DisableDefaultCmd = true
	hideEngineOnlyCommands(cmd)
	cmd.RunE = brandRootRunE
}

func hideEngineOnlyCommands(root *cobra.Command) {
	for _, name := range engineOnlyCommands {
		for _, sub := range root.Commands() {
			if sub.Name() == name {
				root.RemoveCommand(sub)
				break
			}
		}
	}
}

func brandRootRunE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return c.Help()
	}
	for _, name := range engineOnlyCommands {
		if args[0] == name {
			return fmt.Errorf("%q is an engine command; use msc %s instead", name, engineOnlyUse(name))
		}
	}
	return fmt.Errorf("unknown command %q for %q", args[0], c.Use)
}

func engineOnlyUse(name string) string {
	if name == "completion" {
		return "completion install"
	}
	return name
}

func commandName(cmd *cobra.Command) string {
	if cmd == nil {
		return "msc"
	}
	root := cmd.Root()
	if root != nil && root.Use != "" && root.Use != "msc" {
		return root.Use
	}
	return "msc"
}
