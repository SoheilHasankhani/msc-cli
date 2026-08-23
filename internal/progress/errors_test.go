package progress

import (
	"errors"
	"fmt"
	"testing"
)

func TestShortImageRef(t *testing.T) {
	t.Parallel()

	got := ShortImageRef("registry.isos.clinic/isos/identity-api:latest")
	if got != "identity-api:latest" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatDockerPullErrorStripsHTTPPost(t *testing.T) {
	t.Parallel()

	err := errors.New(`Post "http://docker/images/create?fromImage=redis&tag=latest": connection refused`)
	got := FormatDockerPullError(err)
	if got != "connection refused" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatDockerPullErrorExtractsJSONMessage(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf(`docker pull: 500 Internal Server Error: {"message":"failed to resolve reference \"registry.isos.clinic/mssql/server:2025\": pull access denied, repository does not exist or may require authorization: authorization failed: no basic auth credentials"}`)
	got := FormatDockerPullError(err)
	if got != "pull access denied — run: docker login <registry>" {
		t.Fatalf("got %q", got)
	}
}

func TestShortProgressErrorUsesDockerFormatter(t *testing.T) {
	t.Parallel()

	got := shortProgressError(fmt.Errorf(`docker pull: denied`))
	if got != "denied" {
		t.Fatalf("got %q", got)
	}
}
