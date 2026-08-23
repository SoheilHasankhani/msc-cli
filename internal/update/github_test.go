package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRepoFromEnvDefault(t *testing.T) {
	t.Setenv("MSC_RELEASES_REPO", "")
	if RepoFromEnv() != DefaultRepo {
		t.Fatal(RepoFromEnv())
	}
	t.Setenv("MSC_RELEASES_REPO", "acme/msc")
	if RepoFromEnv() != "acme/msc" {
		t.Fatal(RepoFromEnv())
	}
}

func TestClientLatest(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/SoheilHasankhani/msc-cli/releases/latest" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name":"v1.4.0","assets":[{"name":"checksums.txt","browser_download_url":"https://ex/sum"}]}`))
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), Base: srv.URL, Repo: DefaultRepo}
	rel, err := c.Latest(context.Background())
	if err != nil || rel.Tag != "v1.4.0" || len(rel.Assets) != 1 {
		t.Fatalf("%+v %v", rel, err)
	}
}

func TestClientLatestHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), Base: srv.URL, Repo: DefaultRepo}
	if _, err := c.Latest(context.Background()); err == nil {
		t.Fatal("expected HTTP error")
	}
}
