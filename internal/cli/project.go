package cli

import (
	"fmt"

	"github.com/SoheilHasankhani/msc-cli/internal/paths"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/registry"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/spf13/cobra"
)

func projectName(cmd *cobra.Command) (string, error) {
	name, err := cmd.Root().PersistentFlags().GetString("project")
	if err != nil {
		return "", err
	}
	if name == "" {
		name = project.FromEnv()
	}
	if name == "" {
		return "", fmt.Errorf("%s", project.MissingMessage())
	}
	return name, nil
}

func resolveProject(cmd *cobra.Command) (*project.Context, error) {
	name, err := projectName(cmd)
	if err != nil {
		return nil, err
	}
	dirs := paths.Default()
	p, err := project.Resolve(name, dirs)
	if err == nil {
		return p, nil
	}
	reg, rerr := registry.Load(dirs.RegistryFile())
	if rerr != nil {
		return nil, err
	}
	entry, rerr := reg.Resolve(name)
	if rerr != nil {
		return nil, err
	}
	status := registry.CheckPath(entry.Path)
	if status == registry.PathOK {
		return nil, err
	}
	ren := ui.For(cmd.ErrOrStderr())
	action, newPath, perr := ui.RepairBrokenPath(name, entry.Path, pathStatusLabel(status))
	if perr != nil {
		return nil, perr
	}
	var repairAction registry.RepairAction
	switch action {
	case ui.PathRepairRemove:
		repairAction = registry.RepairRemove
	case ui.PathRepairRelink:
		repairAction = registry.RepairRelink
	default:
		return nil, fmt.Errorf("cancelled")
	}
	if err := registry.ApplyRepair(reg, dirs.RegistryFile(), name, repairAction, newPath); err != nil {
		return nil, err
	}
	switch repairAction {
	case registry.RepairRemove:
		return nil, fmt.Errorf("removed %q from the registry — run msc init to register again", name)
	case registry.RepairRelink:
		_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ren, fmt.Sprintf("relinked %q → %s", name, newPath), ""))
	}
	return project.Resolve(name, dirs)
}

func pathStatusLabel(status registry.PathStatus) string {
	switch status {
	case registry.PathMissing:
		return "missing"
	case registry.PathNotDir:
		return "not a directory"
	case registry.PathInvalid:
		return "invalid (no msc.manifest.yml)"
	default:
		return "invalid"
	}
}
