package dockerapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSplitRef(t *testing.T) {
	t.Parallel()
	img, tag := splitRef("alpine:3.19")
	if img != "alpine" || tag != "3.19" {
		t.Fatalf("%s %s", img, tag)
	}
	img, tag = splitRef("registry.example.com/isos/doctor")
	if img != "registry.example.com/isos/doctor" || tag != "latest" {
		t.Fatalf("%s %s", img, tag)
	}
}

func TestPingOK(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_ping" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, "OK")
	}))
	t.Cleanup(srv.Close)

	e := &Engine{http: srv.Client(), base: srv.URL}
	if err := e.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestListContainersParsesComposeLabel(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/containers/json" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `[{"Id":"1","Names":["/isos-doctor"],"State":"running","Labels":{"com.docker.compose.service":"doctor"}}]`)
	}))
	t.Cleanup(srv.Close)

	e := &Engine{http: srv.Client(), base: srv.URL}
	list, err := e.ListContainers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ComposeService != "doctor" || !list[0].Running {
		t.Fatalf("%#v", list)
	}
}

func TestStopServiceTreatsNotModifiedAsSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stop") {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = io.WriteString(w, `[{"Id":"abc","Names":["/doctor"],"State":"exited","Labels":{"com.docker.compose.service":"doctor"}}]`)
	}))
	t.Cleanup(srv.Close)

	e := &Engine{http: srv.Client(), base: srv.URL}
	if err := e.StopService(context.Background(), "doctor"); err != nil {
		t.Fatal(err)
	}
}

func TestPullUsesFromImageQuery(t *testing.T) {
	t.Parallel()

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RawQuery
		_, _ = io.WriteString(w, `{"status":"Already exists"}`+"\n")
	}))
	t.Cleanup(srv.Close)

	e := &Engine{http: srv.Client(), base: srv.URL}
	rc, err := e.Pull(context.Background(), "alpine:3.19")
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	if !strings.Contains(got, "fromImage=alpine") || !strings.Contains(got, "tag=3.19") {
		t.Fatalf("query = %q", got)
	}
}
