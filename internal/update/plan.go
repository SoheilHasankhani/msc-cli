package update

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Asset is one GitHub Release file.
type Asset struct {
	Name string
	URL  string
}

// Release is a GitHub Release tag plus its assets.
type Release struct {
	Tag    string
	Assets []Asset
}

// Plan is a resolved self-update (or install) decision.
type Plan struct {
	Current       string
	Latest        string
	AlreadyLatest bool
	AssetName     string
	AssetURL      string
	ChecksumsURL  string
}

type githubReleaseJSON struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// ParseGitHubRelease decodes the GitHub releases API body.
func ParseGitHubRelease(data []byte) (Release, error) {
	var raw githubReleaseJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return Release{}, err
	}
	rel := Release{Tag: raw.TagName}
	for _, a := range raw.Assets {
		rel.Assets = append(rel.Assets, Asset{Name: a.Name, URL: a.URL})
	}
	return rel, nil
}

// PlanUpdate picks the archive and checksums for this platform.
func PlanUpdate(current string, rel Release, goos, goarch string) (Plan, error) {
	latest := Normalize(rel.Tag)
	p := Plan{Current: Normalize(current), Latest: latest}
	if latest == "" {
		return p, fmt.Errorf("release has no tag")
	}
	if !NeedsUpdate(current, rel.Tag) {
		p.AlreadyLatest = true
		return p, nil
	}
	return attachAssets(p, rel, goos, goarch)
}

// PlanAssets resolves archive URLs even when current already matches latest.
func PlanAssets(current string, rel Release, goos, goarch string) (Plan, error) {
	latest := Normalize(rel.Tag)
	p := Plan{Current: Normalize(current), Latest: latest}
	if latest == "" {
		return p, fmt.Errorf("release has no tag")
	}
	return attachAssets(p, rel, goos, goarch)
}

func attachAssets(p Plan, rel Release, goos, goarch string) (Plan, error) {
	want := AssetName(rel.Tag, goos, goarch)
	p.AssetName = want
	for _, a := range rel.Assets {
		switch {
		case a.Name == want:
			p.AssetURL = a.URL
		case a.Name == "checksums.txt" || strings.HasSuffix(a.Name, "checksums.txt"):
			p.ChecksumsURL = a.URL
		}
	}
	if p.AssetURL == "" {
		return p, fmt.Errorf("no release asset %s for %s/%s", want, goos, goarch)
	}
	if p.ChecksumsURL == "" {
		return p, fmt.Errorf("release is missing checksums.txt")
	}
	return p, nil
}
