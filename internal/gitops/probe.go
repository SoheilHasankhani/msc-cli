package gitops

import (
	"context"
	"fmt"
	"sync"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"golang.org/x/sync/errgroup"
)

// ProbeAccess runs ls-remote for each URL in parallel. A missing SSH agent
// or unreachable Git host fails the batch (access denials do not).
func ProbeAccess(ctx context.Context, r Runner, urls []string) (map[string]bool, error) {
	return Probe(ctx, r, urls, nil)
}

// Probe is ProbeAccess with an optional progress callback.
func Probe(ctx context.Context, r Runner, urls []string, report func(done, total int, url string)) (map[string]bool, error) {
	out := make(map[string]bool, len(urls))
	if len(urls) == 0 {
		return out, nil
	}
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(DefaultProbeParallel)
	var mu sync.Mutex
	done := 0
	total := len(urls)
	for _, u := range urls {
		u := u
		g.Go(func() error {
			ok, err := r.LsRemote(ctx, u)
			if err != nil {
				return err
			}
			mu.Lock()
			out[u] = ok
			done++
			n := done
			mu.Unlock()
			if report != nil {
				report(n, total, u)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// Remote is a Manifest repo name plus its SSH URL.
type Remote struct {
	Name string
	URL  string
}

// ManifestRemotes returns name/URL pairs for every Manifest repo.
func ManifestRemotes(m *manifest.Manifest) ([]Remote, error) {
	if m == nil {
		return nil, fmt.Errorf("manifest is required")
	}
	var out []Remote
	for _, repo := range m.Repos {
		u, err := SSHURL(m.GitHost.BaseURL, repo.Git)
		if err != nil {
			return nil, fmt.Errorf("repo %s: %w", repo.Name, err)
		}
		out = append(out, Remote{Name: repo.Name, URL: u})
	}
	return out, nil
}

// ManifestURLs returns the SSH remote for every Manifest repo.
func ManifestURLs(m *manifest.Manifest) ([]string, error) {
	remotes, err := ManifestRemotes(m)
	if err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(remotes))
	for _, r := range remotes {
		urls = append(urls, r.URL)
	}
	return urls, nil
}
