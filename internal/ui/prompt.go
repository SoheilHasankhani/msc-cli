package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
)

// PathRepairChoice is returned by RepairBrokenPath (no registry import — avoids cycles).
type PathRepairChoice int

const (
	PathRepairCancel PathRepairChoice = iota
	PathRepairRemove
	PathRepairRelink
)

// PromptInteractive reports whether huh forms may run.
func PromptInteractive() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("MSC_NO_PROMPT") != "" {
		return false
	}
	return ColorEnabled(os.Stdin)
}

// RepairBrokenPath asks to relink or remove a broken registry entry.
func RepairBrokenPath(name, oldPath, statusLabel string) (PathRepairChoice, string, error) {
	if !PromptInteractive() {
		return PathRepairCancel, "", fmt.Errorf("registered path for %q is invalid (%s): %s — run: msc projects relink %s --path <dir> or msc projects remove %s",
			name, statusLabel, oldPath, name, name)
	}
	var action string
	var newPath string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(fmt.Sprintf("Project %q path is invalid (%s)", name, statusLabel)).
				Description(oldPath).
				Options(
					huh.NewOption("Relink to a new path", "relink"),
					huh.NewOption("Remove from registry", "remove"),
					huh.NewOption("Cancel", "cancel"),
				).
				Value(&action),
		),
	)
	if err := form.Run(); err != nil {
		return PathRepairCancel, "", err
	}
	switch action {
	case "remove":
		return PathRepairRemove, "", nil
	case "cancel":
		return PathRepairCancel, "", fmt.Errorf("cancelled")
	case "relink":
		pathForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("New meta-repo path").
					Placeholder("/path/to/meta-repo").
					Value(&newPath),
			),
		)
		if err := pathForm.Run(); err != nil {
			return PathRepairCancel, "", err
		}
		if newPath == "" {
			return PathRepairCancel, "", fmt.Errorf("path is required")
		}
		return PathRepairRelink, newPath, nil
	default:
		return PathRepairCancel, "", fmt.Errorf("cancelled")
	}
}

// PromptProjectAlias asks for an alternate command name when the brand is taken.
func PromptProjectAlias(brand, existingPath string) (string, error) {
	if !PromptInteractive() {
		return "", fmt.Errorf("command name %q is already registered to a different project at %s; use --as <other-name>", brand, existingPath)
	}
	var as string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Name %q is taken by another project", brand)).
				Description(existingPath).
				Placeholder("my-isos").
				Value(&as),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	if as == "" {
		return "", fmt.Errorf("a project alias is required")
	}
	return as, nil
}

// ConfirmManifestDraft optionally reviews a drafted manifest before init continues.
func ConfirmManifestDraft(path string) (bool, error) {
	if !PromptInteractive() {
		return true, nil
	}
	var ok bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Draft manifest written").
				Description(path + "\nReview and commit msc.manifest.yml when ready. Continue init?").
				Value(&ok),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}
	if !ok {
		return false, fmt.Errorf("cancelled")
	}
	return true, nil
}
