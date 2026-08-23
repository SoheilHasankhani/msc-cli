package update

import (
	"strings"
	"testing"
)

func TestPlanUpdateFindsAssetAndChecksums(t *testing.T) {
	t.Parallel()

	rel := Release{
		Tag: "v1.2.3",
		Assets: []Asset{
			{Name: "checksums.txt", URL: "https://example/checksums.txt"},
			{Name: "msc_1.2.3_linux_amd64.tar.gz", URL: "https://example/linux.tgz"},
			{Name: "msc_1.2.3_darwin_arm64.tar.gz", URL: "https://example/mac.tgz"},
		},
	}
	p, err := PlanUpdate("1.0.0", rel, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if p.AlreadyLatest || p.Latest != "1.2.3" || p.AssetURL != "https://example/linux.tgz" || p.ChecksumsURL == "" {
		t.Fatalf("%+v", p)
	}
}

func TestPlanUpdateAlreadyLatest(t *testing.T) {
	t.Parallel()

	rel := Release{Tag: "v1.0.0", Assets: []Asset{
		{Name: "checksums.txt", URL: "c"},
		{Name: "msc_1.0.0_linux_amd64.tar.gz", URL: "a"},
	}}
	p, err := PlanUpdate("v1.0.0", rel, "linux", "amd64")
	if err != nil || !p.AlreadyLatest {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestPlanUpdateMissingAsset(t *testing.T) {
	t.Parallel()
	_, err := PlanUpdate("dev", Release{Tag: "v1.0.0"}, "linux", "amd64")
	if err == nil || !strings.Contains(err.Error(), "no release asset") {
		t.Fatalf("%v", err)
	}
}

func TestPlanAssetsWhenAlreadyLatest(t *testing.T) {
	t.Parallel()

	rel := Release{Tag: "v1.0.0", Assets: []Asset{
		{Name: "checksums.txt", URL: "c"},
		{Name: "msc_1.0.0_linux_amd64.tar.gz", URL: "a"},
	}}
	p, err := PlanAssets("1.0.0", rel, "linux", "amd64")
	if err != nil || p.AlreadyLatest || p.AssetURL != "a" {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestParseGitHubReleaseJSON(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"tag_name":"v0.2.0","assets":[{"name":"checksums.txt","browser_download_url":"https://ex/sum"},{"name":"msc_0.2.0_linux_amd64.tar.gz","browser_download_url":"https://ex/tgz"}]}`)
	rel, err := ParseGitHubRelease(raw)
	if err != nil || rel.Tag != "v0.2.0" || len(rel.Assets) != 2 || rel.Assets[0].URL != "https://ex/sum" {
		t.Fatalf("%+v %v", rel, err)
	}
}
