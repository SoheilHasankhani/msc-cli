package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// DefaultRepo is the GitHub owner/name that publishes msc releases.
const DefaultRepo = "SoheilHasankhani/msc-cli"

// RepoFromEnv returns MSC_RELEASES_REPO or DefaultRepo.
func RepoFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("MSC_RELEASES_REPO")); v != "" {
		return v
	}
	return DefaultRepo
}

// Client talks to the GitHub Releases API.
type Client struct {
	HTTP *http.Client
	Base string // default https://api.github.com
	Repo string // owner/name
}

// Latest returns the newest published GitHub Release.
func (c *Client) Latest(ctx context.Context) (Release, error) {
	return c.get(ctx, "/releases/latest")
}

// ReleaseByTag returns one published release.
func (c *Client) ReleaseByTag(ctx context.Context, tag string) (Release, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return Release{}, fmt.Errorf("release tag is empty")
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return c.get(ctx, "/releases/tags/"+tag)
}

func (c *Client) get(ctx context.Context, path string) (Release, error) {
	repo := c.Repo
	if repo == "" {
		repo = RepoFromEnv()
	}
	base := strings.TrimRight(c.Base, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/repos/"+repo+path, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "msc-cli")
	if tok := strings.TrimSpace(os.Getenv("MSC_GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return Release{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("no GitHub release for %s (%s): HTTP %d", repo, path, resp.StatusCode)
	}
	return ParseGitHubRelease(body)
}
