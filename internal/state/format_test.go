package state

import (
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/ui"
)

func TestFormatStatusStableTable(t *testing.T) {
	t.Parallel()

	got := FormatStatus(map[string]ServiceState{
		"doctor":       {ComposeService: "doctor", Mode: ModeSource, ContainerUp: false, NginxTarget: NginxSourceTarget(5010)},
		"identity.api": {ComposeService: "identity.api", Mode: ModeDocker, ContainerUp: true, NginxTarget: NginxTargetContainer},
	}, []string{"service doctor drifted"}, ui.Render{})

	if !strings.Contains(got, "warning: service doctor drifted") {
		t.Fatalf("missing warning:\n%s", got)
	}
	if !strings.Contains(got, "SERVICE") || !strings.Contains(got, "doctor") || !strings.Contains(got, "identity.api") {
		t.Fatalf("table:\n%s", got)
	}
	doc := strings.Index(got, "\ndoctor")
	id := strings.Index(got, "identity.api")
	if doc < 0 || id < 0 || doc > id {
		t.Fatalf("expected doctor before identity.api by sort, got:\n%s", got)
	}
}
