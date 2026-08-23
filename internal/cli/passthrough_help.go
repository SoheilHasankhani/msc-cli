package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type helpSection struct {
	Title string
	Lines []string
}

func expandBrand(line, brand string) string {
	return strings.ReplaceAll(line, "{{brand}}", brand)
}

// printStructuredHelp renders help in the same shape as normal Cobra commands:
// command path, Short, Long, optional sections (Usage, Examples, …).
func printStructuredHelp(cmd *cobra.Command, sections ...helpSection) error {
	brand := commandName(cmd)
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", cmd.CommandPath())
	if s := strings.TrimSpace(cmd.Short); s != "" {
		fmt.Fprintf(&b, "%s\n\n", s)
	}
	if l := strings.TrimSpace(cmd.Long); l != "" {
		fmt.Fprintf(&b, "%s\n\n", l)
	}
	for _, sec := range sections {
		if len(sec.Lines) == 0 {
			continue
		}
		fmt.Fprintf(&b, "%s:\n", sec.Title)
		for _, line := range sec.Lines {
			fmt.Fprintf(&b, "  %s\n", expandBrand(line, brand))
		}
		b.WriteByte('\n')
	}
	_, err := fmt.Fprint(cmd.OutOrStdout(), strings.TrimRight(b.String(), "\n")+"\n")
	return err
}

func passthroughWantsHelp(args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}

func gitHelpSections() []helpSection {
	return []helpSection{
		{
			Title: "Usage",
			Lines: []string{
				"{{brand}} git -- <git args>              run in the meta-repository",
				"{{brand}} git <repo> -- <git args>      run in a cloned service repo",
			},
		},
		{
			Title: "Examples",
			Lines: []string{
				"{{brand}} git -- status -sb",
				"{{brand}} git -- commit -m \"chore: update manifest\"",
				"{{brand}} git identity-api -- log -1 --oneline",
				"{{brand}} git doctor -- pull --ff-only",
				"{{brand}} git doctor -- --help            git's own help inside the clone",
			},
		},
		{
			Title: "Notes",
			Lines: []string{
				"<repo> is a Manifest repo name — list clones with {{brand}} sync",
				"Clone a service repo first: {{brand}} sync <repo>",
				"Put `--` before git flags so msc does not parse them",
			},
		},
	}
}

func printGitHelp(cmd *cobra.Command) error {
	return printStructuredHelp(cmd, gitHelpSections()...)
}
