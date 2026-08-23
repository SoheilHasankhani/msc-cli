package dockerapi

import (
	"context"
	"io"
	"testing"
)

type fakeClient struct {
	list []Container
}

func (f fakeClient) ListContainers(context.Context) ([]Container, error) { return f.list, nil }
func (f fakeClient) StopService(context.Context, string) error           { return nil }
func (f fakeClient) StartService(context.Context, string) error          { return nil }
func (f fakeClient) SignalContainer(context.Context, string, string) error {
	return nil
}
func (f fakeClient) Pull(context.Context, string) (io.ReadCloser, error) { return nil, nil }

func TestRunningFindsComposeService(t *testing.T) {
	t.Parallel()

	c := fakeClient{list: []Container{
		{ComposeService: "doctor", Running: true},
		{ComposeService: "wallet", Running: false},
	}}
	up, err := Running(context.Background(), c, "doctor")
	if err != nil || !up {
		t.Fatalf("doctor running=%v err=%v", up, err)
	}
	up, err = Running(context.Background(), c, "wallet")
	if err != nil || up {
		t.Fatalf("wallet running=%v err=%v", up, err)
	}
	up, err = Running(context.Background(), c, "missing")
	if err != nil || up {
		t.Fatalf("missing running=%v err=%v", up, err)
	}
}
