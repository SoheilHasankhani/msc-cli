package stack

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/progress"
)

type nopPuller struct{}

func (nopPuller) Pull(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(`{"status":"Already exists","id":"latest"}` + "\n")), nil
}

type failPuller struct{ err error }

func (p failPuller) Pull(context.Context, string) (io.ReadCloser, error) {
	return nil, p.err
}

func TestStackUpSourcePullThenUpSameRow(t *testing.T) {
	t.Parallel()

	fc := &fakeCompose{
		services: []dockerapi.ComposeService{
			{Name: "patient", ContainerName: "isos-patient", Image: "registry.isos.clinic/isos/patient:latest"},
		},
		upStatus: []dockerapi.ComposeStatus{
			{Container: "isos-patient", State: "Starting"},
			{Container: "isos-patient", State: "Started"},
		},
	}
	src := StackUpSource{
		Compose:  fc,
		Root:     "/work",
		File:     "local/docker-compose.yml",
		SkipPull: true,
	}
	updates := make(chan progress.Update, 8)
	done := make(chan error, 1)
	go func() {
		done <- src.Run(context.Background(), updates)
		close(updates)
	}()

	var got []progress.Update
	for u := range updates {
		got = append(got, u)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(got) < 3 {
		t.Fatalf("updates = %#v", got)
	}
	if got[0].ID != "patient" || got[0].Label != "patient" {
		t.Fatalf("first row = %#v", got[0])
	}
	last := got[len(got)-1]
	if last.ID != "patient" || last.Status != "Started" || !last.Done {
		t.Fatalf("last = %#v", last)
	}
}

func TestStackUpSourceSkipsSourceModeServices(t *testing.T) {
	t.Parallel()

	fc := &fakeCompose{
		services: []dockerapi.ComposeService{
			{Name: "nginx", ContainerName: "isos-nginx", Image: "nginx"},
			{Name: "patient", ContainerName: "isos-patient", Image: "patient:latest"},
		},
		upStatus: []dockerapi.ComposeStatus{
			{Container: "isos-nginx", State: "Started"},
		},
	}
	src := StackUpSource{
		Compose:      fc,
		Root:         "/work",
		File:         "local/docker-compose.yml",
		SkipPull:     true,
		SkipServices: []string{"patient"},
	}
	updates := make(chan progress.Update, 8)
	done := make(chan error, 1)
	go func() {
		done <- src.Run(context.Background(), updates)
		close(updates)
	}()
	var got []progress.Update
	for u := range updates {
		got = append(got, u)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !fc.lastOpts.NoDeps || len(fc.lastOpts.Services) != 1 || fc.lastOpts.Services[0] != "nginx" {
		t.Fatalf("up opts = %#v", fc.lastOpts)
	}
	foundSource := false
	for _, u := range got {
		if u.ID == "patient" && u.Status == "source" && u.Done {
			foundSource = true
		}
	}
	if !foundSource {
		t.Fatalf("missing source-mode row: %#v", got)
	}
}

func TestStackUpSourcePullOnlySkipsUp(t *testing.T) {
	t.Parallel()

	fc := &fakeCompose{
		services: []dockerapi.ComposeService{
			{Name: "patient", ContainerName: "isos-patient", Image: "patient:latest"},
		},
	}
	src := StackUpSource{
		Compose:  fc,
		Root:     "/work",
		File:     "local/docker-compose.yml",
		PullOnly: true,
		Puller:   nopPuller{},
	}
	tty := false
	if err := progress.Run(context.Background(), []progress.Source{src}, progress.Options{Output: io.Discard, IsTTY: &tty}); err != nil {
		t.Fatal(err)
	}
	if fc.upCalled {
		t.Fatal("compose up should not run with pull-only")
	}
}

func TestStackUpSourcePullWarnWithRunningServiceSucceeds(t *testing.T) {
	t.Parallel()

	pullErr := errors.New(`docker pull: unknown: failed to resolve reference "docker.io/library/sos-rabbitmq:4-management"`)
	fc := &fakeCompose{
		services: []dockerapi.ComposeService{
			{Name: "rabbitmq", ContainerName: "sos-rabbitmq", Image: "sos-rabbitmq:4-management"},
		},
		upStatus: []dockerapi.ComposeStatus{
			{Container: "sos-rabbitmq", State: "Running"},
		},
	}
	src := StackUpSource{
		Compose: fc,
		Root:    "/work",
		File:    "local/docker-compose.yml",
		Puller:  failPuller{err: pullErr},
	}
	tty := false
	err := progress.Run(context.Background(), []progress.Source{src}, progress.Options{Output: io.Discard, IsTTY: &tty})
	if err != nil {
		t.Fatalf("expected success when service is running despite pull failure: %v", err)
	}
}

func TestStackUpSourcePullOnlyFailsOnPullError(t *testing.T) {
	t.Parallel()

	fc := &fakeCompose{
		services: []dockerapi.ComposeService{
			{Name: "rabbitmq", ContainerName: "sos-rabbitmq", Image: "sos-rabbitmq:4-management"},
		},
	}
	src := StackUpSource{
		Compose:  fc,
		Root:     "/work",
		File:     "local/docker-compose.yml",
		PullOnly: true,
		Puller:   failPuller{err: errors.New(`docker pull: unknown: failed to resolve reference "docker.io/library/sos-rabbitmq:4-management"`)},
	}
	tty := false
	err := progress.Run(context.Background(), []progress.Source{src}, progress.Options{Output: io.Discard, IsTTY: &tty})
	if err == nil {
		t.Fatal("expected pull-only to fail when pull fails")
	}
}

func TestStackUpSourceComposeFailureFailsCommand(t *testing.T) {
	t.Parallel()

	fc := &fakeCompose{
		services: []dockerapi.ComposeService{
			{Name: "patient", ContainerName: "isos-patient", Image: "patient:latest"},
		},
		upStatus: []dockerapi.ComposeStatus{
			{Container: "isos-patient", State: "Error response from daemon"},
		},
	}
	src := StackUpSource{
		Compose:  fc,
		Root:     "/work",
		File:     "local/docker-compose.yml",
		SkipPull: true,
	}
	tty := false
	err := progress.Run(context.Background(), []progress.Source{src}, progress.Options{Output: io.Discard, IsTTY: &tty})
	if err == nil {
		t.Fatal("expected failure when compose reports service error")
	}
}
