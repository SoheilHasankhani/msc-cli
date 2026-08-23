package cli

import (
	"fmt"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/registry"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/spf13/cobra"
)

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List, remove, or relink registered projects",
	}
	cmd.AddCommand(newProjectsListCmd(), newProjectsRemoveCmd(), newProjectsRelinkCmd())
	return cmd
}

func newProjectsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := registry.Load(paths.Default().RegistryFile())
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), registry.FormatList(reg, ui.For(cmd.OutOrStdout())))
			return err
		},
	}
}

func newProjectsRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove NAME",
		Short: "Unregister a project (does not delete files on disk)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := paths.Default().RegistryFile()
			reg, err := registry.Load(file)
			if err != nil {
				return err
			}
			if err := reg.Remove(args[0]); err != nil {
				return err
			}
			if err := reg.Save(file); err != nil {
				return err
			}
			ren := ui.For(cmd.OutOrStdout())
			_, err = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ren,
				fmt.Sprintf("removed %s from the registry", args[0]),
				"files on disk were not deleted"))
			if err != nil {
				return err
			}
			refreshShellCompletionQuiet()
			return nil
		},
	}
}

func newProjectsRelinkCmd() *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "relink NAME",
		Short: "Point a registered name at a new meta-repo path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := paths.Default().RegistryFile()
			reg, err := registry.Load(file)
			if err != nil {
				return err
			}
			if err := reg.Relink(args[0], path); err != nil {
				return err
			}
			if err := reg.Save(file); err != nil {
				return err
			}
			ren := ui.For(cmd.OutOrStdout())
			_, err = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ren, fmt.Sprintf("relinked %s → %s", args[0], path), ""))
			return err
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "new absolute or relative path to the meta-repo")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}
