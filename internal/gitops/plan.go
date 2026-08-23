package gitops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/manifest"
)

// RepoStatus is one Manifest repo after access + filesystem checks.
type RepoStatus struct {
	Name       string
	URL        string
	Dest       string
	Cloned     bool
	Accessible bool
	Warning    string
}

// Plan is the sync decision set. Denied (never cloned) repos are omitted from
// user-facing lists; revoked (cloned, access gone) are kept and warned.
type Plan struct {
	Available []RepoStatus
	Cloned    []RepoStatus
	Denied    []RepoStatus
	Revoked   []RepoStatus
}

// BuildPlan classifies each Manifest repo. access maps SSH URL → accessible.
func BuildPlan(m *manifest.Manifest, clonesDir string, access map[string]bool) (Plan, error) {
	var plan Plan
	if m == nil {
		return plan, fmt.Errorf("manifest is required")
	}
	for _, repo := range m.Repos {
		url, err := SSHURL(m.GitHost.BaseURL, repo.Git)
		if err != nil {
			return Plan{}, fmt.Errorf("repo %s: %w", repo.Name, err)
		}
		dest := filepath.Join(clonesDir, repo.Name)
		st := RepoStatus{
			Name:       repo.Name,
			URL:        url,
			Dest:       dest,
			Cloned:     isClone(dest),
			Accessible: access[url],
		}
		switch {
		case st.Cloned && !st.Accessible:
			st.Warning = revokedWarning(repo)
			plan.Revoked = append(plan.Revoked, st)
			plan.Cloned = append(plan.Cloned, st)
		case st.Cloned:
			plan.Cloned = append(plan.Cloned, st)
		case st.Accessible:
			plan.Available = append(plan.Available, st)
		default:
			plan.Denied = append(plan.Denied, st)
		}
	}
	return plan, nil
}

func isClone(dest string) bool {
	info, err := os.Stat(dest)
	if err != nil || !info.IsDir() {
		return false
	}
	_, err = os.Stat(filepath.Join(dest, ".git"))
	return err == nil
}

func revokedWarning(repo manifest.RepoDef) string {
	var b strings.Builder
	b.WriteString("access was revoked; the local clone is kept (never deleted) and cannot be pulled")
	if len(repo.Services) > 0 {
		fmt.Fprintf(&b, "; if it is in Source Mode, consider: switch %s --to docker", repo.Services[0].ComposeService)
	}
	return b.String()
}

// SelectUpdate splits accessible repos into clone and pull targets for a default sync.
func (p Plan) SelectUpdate(names []string) (toClone, toPull []RepoStatus, err error) {
	byAvail := map[string]RepoStatus{}
	for _, r := range p.Available {
		byAvail[r.Name] = r
	}
	byCloned := map[string]RepoStatus{}
	for _, r := range p.Cloned {
		byCloned[r.Name] = r
	}
	if len(names) == 0 {
		toClone = append(toClone, p.Available...)
		for _, r := range p.Cloned {
			if r.Accessible {
				toPull = append(toPull, r)
			}
		}
		if len(toClone) == 0 && len(toPull) == 0 {
			return nil, nil, fmt.Errorf("no accessible repos to sync")
		}
		return toClone, toPull, nil
	}
	for _, name := range names {
		if r, ok := byAvail[name]; ok {
			toClone = append(toClone, r)
			continue
		}
		if r, ok := byCloned[name]; ok {
			if !r.Accessible {
				msg := r.Warning
				if msg == "" {
					msg = "no access"
				}
				return nil, nil, fmt.Errorf("repo %q cannot be synced: %s", name, msg)
			}
			toPull = append(toPull, r)
			continue
		}
		for _, r := range p.Denied {
			if r.Name == name {
				return nil, nil, fmt.Errorf("repo %q is not accessible", name)
			}
		}
		return nil, nil, fmt.Errorf("repo %q is unknown", name)
	}
	return toClone, toPull, nil
}

// SelectPull returns cloned, accessible repos to fast-forward.
func (p Plan) SelectPull(names []string, all bool) ([]RepoStatus, error) {
	byName := map[string]RepoStatus{}
	for _, r := range p.Cloned {
		byName[r.Name] = r
	}
	var pullable []RepoStatus
	for _, r := range p.Cloned {
		if r.Accessible {
			pullable = append(pullable, r)
		}
	}
	if all || len(names) == 0 {
		if len(pullable) == 0 {
			return nil, fmt.Errorf("no cloned repos to pull (run: msc sync first)")
		}
		return pullable, nil
	}
	var out []RepoStatus
	for _, name := range names {
		r, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("repo %q is not cloned — run: msc sync %s", name, name)
		}
		if !r.Accessible {
			msg := r.Warning
			if msg == "" {
				msg = "no access to pull updates"
			}
			return nil, fmt.Errorf("repo %q cannot be pulled: %s", name, msg)
		}
		out = append(out, r)
	}
	return out, nil
}

// Select returns repos to clone. all=true takes every Available repo.
func (p Plan) Select(names []string, all bool) ([]RepoStatus, error) {
	if all {
		if len(p.Available) == 0 {
			return nil, fmt.Errorf("no accessible repos left to clone")
		}
		return p.Available, nil
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("select repos to clone, or pass --all")
	}
	byName := map[string]RepoStatus{}
	for _, r := range p.Available {
		byName[r.Name] = r
	}
	var out []RepoStatus
	for _, name := range names {
		r, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("repo %q is not available to clone (unknown, already cloned, or no access)", name)
		}
		out = append(out, r)
	}
	return out, nil
}
