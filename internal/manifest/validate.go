package manifest

import (
	"fmt"
	"net/url"
	"regexp"
)

var commandRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// Validate reports the first schema problem on m.
func (m *Manifest) Validate() error {
	if m == nil {
		return fmt.Errorf("manifest is nil")
	}
	if m.Brand.DisplayName == "" {
		return fmt.Errorf("brand.display_name is required")
	}
	if m.Brand.Command == "" {
		return fmt.Errorf("brand.command is required")
	}
	if !commandRE.MatchString(m.Brand.Command) {
		return fmt.Errorf("brand.command %q is not a valid command name", m.Brand.Command)
	}
	if m.GitHost.BaseURL == "" {
		return fmt.Errorf("git_host.base_url is required")
	}
	if err := validateHTTPURL(m.GitHost.BaseURL); err != nil {
		return fmt.Errorf("git_host.base_url: %w", err)
	}
	if m.LocalDomain == "" {
		return fmt.Errorf("local_domain is required")
	}

	repoNames := make(map[string]struct{}, len(m.Repos))
	composeNames := make(map[string]struct{})
	ports := make(map[int]struct{})

	for i, repo := range m.Repos {
		if repo.Name == "" {
			return fmt.Errorf("repos[%d].name is required", i)
		}
		if _, dup := repoNames[repo.Name]; dup {
			return fmt.Errorf("duplicate repo name %q", repo.Name)
		}
		repoNames[repo.Name] = struct{}{}
		if repo.Git == "" {
			return fmt.Errorf("repos[%d].git is required", i)
		}
		for j, svc := range repo.Services {
			if svc.ComposeService == "" {
				return fmt.Errorf("repos[%d].services[%d].compose_service is required", i, j)
			}
			if _, dup := composeNames[svc.ComposeService]; dup {
				return fmt.Errorf("duplicate compose_service %q", svc.ComposeService)
			}
			composeNames[svc.ComposeService] = struct{}{}
			if svc.Path == "" {
				return fmt.Errorf("repos[%d].services[%d].path is required", i, j)
			}
			if svc.SourcePort < 1 || svc.SourcePort > 65535 {
				return fmt.Errorf("repos[%d].services[%d].source_port %d is out of range", i, j, svc.SourcePort)
			}
			if _, dup := ports[svc.SourcePort]; dup {
				return fmt.Errorf("duplicate source_port %d", svc.SourcePort)
			}
			ports[svc.SourcePort] = struct{}{}
		}
	}
	return nil
}

func validateHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%q must be an http(s) URL", raw)
	}
	if u.Host == "" {
		return fmt.Errorf("%q is missing a host", raw)
	}
	return nil
}
