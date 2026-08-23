package version

import (
	"strings"
	"testing"
)

func TestFormatIncludesAllFields(t *testing.T) {
	t.Parallel()

	got := Format("1.2.3", "abc1234", "2026-08-18T00:00:00Z")

	for _, want := range []string{"1.2.3", "abc1234", "2026-08-18T00:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Format() = %q, want it to contain %q", got, want)
		}
	}
}

func TestStringUsesPackageDefaults(t *testing.T) {
	t.Parallel()

	got := String()
	want := Format(Version, Commit, Date)
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestDefaultDevIdentityIsNonEmpty(t *testing.T) {
	t.Parallel()

	if Version == "" || Commit == "" || Date == "" {
		t.Fatalf("default version fields must be non-empty, got Version=%q Commit=%q Date=%q", Version, Commit, Date)
	}
}
