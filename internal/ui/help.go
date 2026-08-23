package ui

import "fmt"

// HelpBanner is a short branded header printed before command help.
func HelpBanner(r Render, name, tagline string) string {
	return fmt.Sprintf("%s\n%s\n\n", Title(r, name), Dim(r, tagline))
}

// VersionLine styles --version output when color is enabled.
func VersionLine(r Render, version string) string {
	return fmt.Sprintf("%s\n%s\n", Title(r, "msc"), Dim(r, version))
}
