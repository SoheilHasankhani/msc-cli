package registry

import (
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/ui"
)

func TestFormatList(t *testing.T) {
	t.Parallel()

	r := New()
	_, _ = r.Register("mores", sampleEntry("/work/mores"))
	_, _ = r.Register("isos", sampleEntry("/work/isos"))
	got := FormatList(r, ui.Render{})
	if !strings.Contains(got, "NAME") || !strings.Contains(got, "isos") || !strings.Contains(got, "mores") {
		t.Fatalf("%s", got)
	}
	if strings.Index(got, "isos") > strings.Index(got, "mores") {
		t.Fatalf("expected isos before mores:\n%s", got)
	}
}

func TestFormatListEmpty(t *testing.T) {
	t.Parallel()
	got := FormatList(New(), ui.Render{})
	if !strings.Contains(got, "no projects") {
		t.Fatalf("%s", got)
	}
}
