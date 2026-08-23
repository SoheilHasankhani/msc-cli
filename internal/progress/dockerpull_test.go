package progress

import (
	"strings"
	"testing"
)

func TestParsePullStreamAggregatesLayerBytes(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		`{"status":"Pulling from library/alpine","id":"latest"}`,
		`{"status":"Downloading","progressDetail":{"current":100,"total":400},"id":"layer1"}`,
		`{"status":"Downloading","progressDetail":{"current":50,"total":200},"id":"layer2"}`,
		`{"status":"Download complete","id":"layer1"}`,
		`{"status":"Download complete","id":"layer2"}`,
		`not-json`,
		`{"status":"Status: Downloaded newer image"}`,
	}, "\n")

	var got []Update
	if err := ParsePullStream(strings.NewReader(input), "alpine", "alpine:latest", func(u Update) {
		got = append(got, u)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("expected updates")
	}
	last := got[len(got)-1]
	if !last.Done || last.ID != "alpine" {
		t.Fatalf("last = %#v", last)
	}
	var sawPartial bool
	for _, u := range got {
		if u.Current == 150 && u.Total == 600 {
			sawPartial = true
		}
	}
	if !sawPartial {
		t.Fatalf("expected aggregated 150/600 in %#v", got)
	}
}

func TestParsePullStreamSurfacesErrorStatus(t *testing.T) {
	t.Parallel()

	input := `{"error":"denied: requested access to the resource is denied"}` + "\n"
	err := ParsePullStream(strings.NewReader(input), "img", "img", func(u Update) {})
	if err == nil {
		t.Fatal("expected error")
	}
	if IsRetryableDockerPullError(err) {
		t.Fatalf("auth error should not be retryable: %v", err)
	}
}
