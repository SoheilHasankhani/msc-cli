package ui

import (
	"strings"
	"testing"
)

func TestColorEnabledRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if ColorEnabled(nil) {
		t.Fatal("NO_COLOR should disable color")
	}
}

func TestTablePlain(t *testing.T) {
	t.Parallel()
	got := Table{
		Headers: []string{"A", "B"},
		Rows:    [][]string{{"1", "2"}},
	}.String()
	if !strings.Contains(got, "A") || !strings.Contains(got, "1") {
		t.Fatalf("%q", got)
	}
}

func TestErrorNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if Error("boom") != "boom" {
		t.Fatal(Error("boom"))
	}
}
