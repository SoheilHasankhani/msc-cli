// Package version holds build-time identity for the msc engine binary.
// Values are overwritten via -ldflags at build/release time.
package version

import "fmt"

var (
	// Version is the semantic version or git describe string. Defaults to "dev"
	// when running from source without ldflags.
	Version = "dev"
	// Commit is the short git SHA, or "none" for an unversioned tree.
	Commit = "none"
	// Date is the UTC build timestamp, or "unknown" when running from source.
	Date = "unknown"
)

// String returns a single-line identity suitable for `msc --version`.
func String() string {
	return Format(Version, Commit, Date)
}

// Format builds the version identity from explicit fields. Kept separate from
// String so tests do not mutate package-level ldflag variables.
func Format(ver, commit, date string) string {
	return fmt.Sprintf("%s (%s, built %s)", ver, commit, date)
}
