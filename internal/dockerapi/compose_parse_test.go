package dockerapi

import "testing"

func TestParseComposeStatusLine(t *testing.T) {
	t.Parallel()

	cases := []struct {
		line string
		want ComposeStatus
		ok   bool
	}{
		{" Container isos-patient Starting ", ComposeStatus{Container: "isos-patient", State: "Starting"}, true},
		{" Container isos-patient Started ", ComposeStatus{Container: "isos-patient", State: "Started"}, true},
		{" Container sos-elasticsearch Waiting ", ComposeStatus{Container: "sos-elasticsearch", State: "Waiting"}, true},
		{" Container sos-elasticsearch Healthy ", ComposeStatus{Container: "sos-elasticsearch", State: "Healthy"}, true},
		{" Container sos-kibana-settings Exited ", ComposeStatus{Container: "sos-kibana-settings", State: "Exited"}, true},
		{"network foo", ComposeStatus{}, false},
	}
	for _, tc := range cases {
		got, ok := ParseComposeStatusLine(tc.line)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("ParseComposeStatusLine(%q) = (%#v, %v), want (%#v, %v)", tc.line, got, ok, tc.want, tc.ok)
		}
	}
}

func TestParseComposePullLine(t *testing.T) {
	t.Parallel()

	got, ok := ParseComposePullLine(" Image registry.isos.clinic/isos/wallet:latest Pulling ")
	if !ok || got.Container != "registry.isos.clinic/isos/wallet:latest" || got.State != "Pulling" {
		t.Fatalf("got %#v ok=%v", got, ok)
	}
	got, ok = ParseComposePullLine(" Image sos-rabbitmq:4-management Skipped Image is already present locally")
	if !ok || got.State != "Skipped" {
		t.Fatalf("skipped = %#v ok=%v", got, ok)
	}
}

func TestIsTerminalComposeStatus(t *testing.T) {
	t.Parallel()

	for _, state := range []string{"Started", "Running", "Healthy", "Exited", "Recreated", "Stopped", "Pulled", "Skipped"} {
		if !IsTerminalComposeStatus(state) {
			t.Fatalf("%q should be terminal", state)
		}
	}
	for _, state := range []string{"Starting", "Waiting", "Recreate", "Pulling"} {
		if IsTerminalComposeStatus(state) {
			t.Fatalf("%q should not be terminal", state)
		}
	}
}
