package cli

import (
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/complete"
	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/state"
	"github.com/spf13/cobra"
)

func completeServices(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	p, err := resolveProject(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cache, _ := state.LoadCache(state.CachePath(paths.Default().StateDir(), p.Name))
	return complete.FilterPrefix(complete.Services(p.Manifest, cache), toComplete), cobra.ShellCompDirectiveNoFileComp
}

func completeSwitchModes(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return complete.FilterPrefix([]string{"docker", "source"}, toComplete), cobra.ShellCompDirectiveNoFileComp
}

func completeRepos(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	p, err := resolveProject(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return complete.FilterPrefix(complete.Repos(p.Manifest), toComplete), cobra.ShellCompDirectiveNoFileComp
}

func completeCompose(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if strings.HasPrefix(toComplete, "-") {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 0 {
		return complete.FilterPrefix(complete.ComposeSubcommands(), toComplete), cobra.ShellCompDirectiveNoFileComp
	}
	if !complete.ComposeTakesServices(args[0]) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	p, err := resolveProject(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cache, _ := state.LoadCache(state.CachePath(paths.Default().StateDir(), p.Name))
	return complete.FilterPrefix(complete.Services(p.Manifest, cache), toComplete), cobra.ShellCompDirectiveNoFileComp
}
