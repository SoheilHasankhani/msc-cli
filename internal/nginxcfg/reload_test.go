package nginxcfg

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
)

var errReload = errors.New("signal: connection reset")

type recordingClient struct {
	list      []dockerapi.Container
	signaled  string
	signal    string
	listErr   error
	signalErr error
}

func (c *recordingClient) ListContainers(context.Context) ([]dockerapi.Container, error) {
	return c.list, c.listErr
}
func (c *recordingClient) StopService(context.Context, string) error  { return nil }
func (c *recordingClient) StartService(context.Context, string) error { return nil }
func (c *recordingClient) SignalContainer(_ context.Context, name, signal string) error {
	c.signaled = name
	c.signal = signal
	return c.signalErr
}
func (c *recordingClient) Pull(context.Context, string) (io.ReadCloser, error) { return nil, nil }

func TestReloadNginxSendsHUP(t *testing.T) {
	t.Parallel()

	c := &recordingClient{list: []dockerapi.Container{
		{Name: "/isos-doctor", ComposeService: "doctor", Running: true},
		{Name: "/isos-nginx", ComposeService: "nginx", Running: true},
	}}
	if err := Reload(context.Background(), c, ""); err != nil {
		t.Fatal(err)
	}
	if c.signaled != "isos-nginx" || c.signal != "HUP" {
		t.Fatalf("signaled %q %q", c.signaled, c.signal)
	}
}

func TestReloadNginxMissingContainer(t *testing.T) {
	t.Parallel()

	c := &recordingClient{list: []dockerapi.Container{
		{Name: "/isos-doctor", ComposeService: "doctor", Running: true},
	}}
	if err := Reload(context.Background(), c, "nginx"); err != nil {
		t.Fatal(err)
	}
	if c.signaled != "" {
		t.Fatalf("signaled %q", c.signaled)
	}
}

func TestReloadNginxSignalErrorKeepsRoutingNote(t *testing.T) {
	t.Parallel()

	c := &recordingClient{
		list: []dockerapi.Container{
			{Name: "/isos-nginx", ComposeService: "nginx", Running: true},
		},
		signalErr: errReload,
	}
	err := Reload(context.Background(), c, "nginx")
	if err == nil || !strings.Contains(err.Error(), "previous routing is unchanged") {
		t.Fatalf("%v", err)
	}
}

func TestReloadNginxNotRunning(t *testing.T) {
	t.Parallel()

	c := &recordingClient{list: []dockerapi.Container{
		{Name: "/isos-nginx", ComposeService: "nginx", Running: false},
	}}
	if err := Reload(context.Background(), c, "nginx"); err != nil {
		t.Fatal(err)
	}
	if c.signaled != "" {
		t.Fatalf("signaled %q", c.signaled)
	}
}
