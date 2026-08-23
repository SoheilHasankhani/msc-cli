package manifest

import "fmt"

// LookupService finds a service by compose_service or a unique single-service repo name.
func (m *Manifest) LookupService(name string) (ServiceDef, error) {
	if m == nil {
		return ServiceDef{}, fmt.Errorf("manifest is nil")
	}
	var found []ServiceDef
	seen := map[string]bool{}
	add := func(s ServiceDef) {
		if seen[s.ComposeService] {
			return
		}
		seen[s.ComposeService] = true
		found = append(found, s)
	}
	for _, repo := range m.Repos {
		if repo.Name == name && len(repo.Services) > 1 {
			return ServiceDef{}, fmt.Errorf("repo %q has %d services; use a compose_service name (e.g. %s)", repo.Name, len(repo.Services), repo.Services[0].ComposeService)
		}
		for _, svc := range repo.Services {
			if svc.ComposeService == name {
				add(svc)
			}
		}
		if repo.Name == name && len(repo.Services) == 1 {
			add(repo.Services[0])
		}
	}
	if len(found) == 0 {
		return ServiceDef{}, fmt.Errorf("unknown service %q; use a compose_service or a unique repo name from the Manifest", name)
	}
	if len(found) > 1 {
		return ServiceDef{}, fmt.Errorf("ambiguous service name %q", name)
	}
	return found[0], nil
}
