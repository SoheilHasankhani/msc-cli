package update

import (
	"strconv"
	"strings"
)

// Normalize strips a leading v and trims space.
func Normalize(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// NeedsUpdate reports whether current should be replaced by latest.
// Unreleased builds ("dev", "none", empty) always update when latest is set.
func NeedsUpdate(current, latest string) bool {
	cur := Normalize(current)
	lat := Normalize(latest)
	if lat == "" {
		return false
	}
	if cur == "" || cur == "dev" || cur == "none" {
		return true
	}
	return compareSemver(cur, lat) < 0
}

func compareSemver(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var ai, bi int
		if i < len(as) {
			ai, _ = strconv.Atoi(numericPrefix(as[i]))
		}
		if i < len(bs) {
			bi, _ = strconv.Atoi(numericPrefix(bs[i]))
		}
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}

func numericPrefix(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return "0"
	}
	return b.String()
}

// AssetName is the GoReleaser archive name for this OS/arch.
func AssetName(version, goos, goarch string) string {
	v := Normalize(version)
	base := "msc_" + v + "_" + goos + "_" + goarch
	if goos == "windows" {
		return base + ".zip"
	}
	return base + ".tar.gz"
}
