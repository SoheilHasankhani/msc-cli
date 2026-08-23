package complete

import (
	"sort"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
	"github.com/SoheilHasankhani/msc-cli/internal/state"
)

// Services is the union of cache keys and Manifest compose_service names, sorted.
func Services(m *manifest.Manifest, cache *state.Snapshot) []string {
	set := map[string]struct{}{}
	if m != nil {
		for _, repo := range m.Repos {
			for _, svc := range repo.Services {
				if n := strings.TrimSpace(svc.ComposeService); n != "" {
					set[n] = struct{}{}
				}
			}
		}
	}
	if cache != nil {
		for name := range cache.Services {
			if n := strings.TrimSpace(name); n != "" {
				set[n] = struct{}{}
			}
		}
	}
	return sorted(set)
}

// Repos returns sorted Manifest repo names.
func Repos(m *manifest.Manifest) []string {
	set := map[string]struct{}{}
	if m != nil {
		for _, repo := range m.Repos {
			if n := strings.TrimSpace(repo.Name); n != "" {
				set[n] = struct{}{}
			}
		}
	}
	return sorted(set)
}

// FilterPrefix keeps names that start with prefix.
func FilterPrefix(names []string, prefix string) []string {
	if prefix == "" {
		return names
	}
	var out []string
	for _, n := range names {
		if strings.HasPrefix(n, prefix) {
			out = append(out, n)
		}
	}
	return out
}

func sorted(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
