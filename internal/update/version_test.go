package update

import "testing"

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()
	if Normalize("v1.2.3") != "1.2.3" || Normalize("1.2.3") != "1.2.3" {
		t.Fatal(Normalize("v1.2.3"), Normalize("1.2.3"))
	}
}

func TestNeedsUpdate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"dev", "1.0.0", true},
		{"none", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"v1.0.0", "1.0.1", true},
		{"1.0.1", "v1.0.0", false},
		{"1.2.0", "1.10.0", true},
		{"1.10.0", "1.2.0", false},
	}
	for _, tc := range cases {
		if got := NeedsUpdate(tc.cur, tc.latest); got != tc.want {
			t.Fatalf("NeedsUpdate(%q,%q)=%v want %v", tc.cur, tc.latest, got, tc.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	t.Parallel()
	if AssetName("v1.2.3", "linux", "amd64") != "msc_1.2.3_linux_amd64.tar.gz" {
		t.Fatal(AssetName("v1.2.3", "linux", "amd64"))
	}
	if AssetName("1.2.3", "windows", "amd64") != "msc_1.2.3_windows_amd64.zip" {
		t.Fatal(AssetName("1.2.3", "windows", "amd64"))
	}
}
