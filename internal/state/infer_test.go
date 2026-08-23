package state

import (
	"context"
	"testing"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
)

type fakeContainers map[string]bool

func (f fakeContainers) Running(_ context.Context, name string) (bool, error) {
	return f[name], nil
}

type fakeNginx map[string]string

func (f fakeNginx) Target(name string) string {
	return f[name]
}

func TestInferCombinesDockerAndNginx(t *testing.T) {
	t.Parallel()

	m := &manifest.Manifest{
		Repos: []manifest.RepoDef{{
			Name: "doctor",
			Services: []manifest.ServiceDef{
				{ComposeService: "doctor", Path: ".", SourcePort: 5010},
				{ComposeService: "identity.api", Path: ".", SourcePort: 5000},
			},
		}},
	}
	in := Inferer{
		Containers: fakeContainers{"doctor": false, "identity.api": true},
		Nginx:      fakeNginx{"doctor": NginxSourceTarget(5010), "identity.api": NginxTargetContainer},
	}

	got, err := in.Infer(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	if got["doctor"].Mode != ModeSource || got["doctor"].ContainerUp {
		t.Fatalf("doctor = %#v", got["doctor"])
	}
	if got["identity.api"].Mode != ModeDocker || !got["identity.api"].ContainerUp {
		t.Fatalf("identity.api = %#v", got["identity.api"])
	}
}
