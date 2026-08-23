package manifest

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SuggestInput is enough to draft a Manifest without asking the developer yet.
type SuggestInput struct {
	DisplayName  string
	Command      string
	GitHostBase  string
	LocalDomain  string
	ComposeYAML  string
	CloneDirs    []string
	DefaultGroup string
}

var infraServices = map[string]bool{
	"nginx": true, "redis": true, "rabbitmq": true, "elasticsearch": true,
	"clickhouse": true, "otel": true, "otel-collector": true, "signoz": true,
	"kibana": true, "wiremock": true, "postgres": true, "postgresql": true,
	"mysql": true, "mongo": true, "mongodb": true, "jaeger": true,
	"grafana": true, "prometheus": true, "zipkin": true, "init-clickhouse": true,
}

var skipCloneDirs = map[string]bool{
	"config": true, "certificates": true, "nginx": true, "compose": true,
}

// Suggest builds a Manifest from compose service names and clone directory names.
func Suggest(in SuggestInput) *Manifest {
	services := composeServiceNames(in.ComposeYAML)
	group := in.DefaultGroup
	if group == "" {
		group = in.Command
	}

	m := &Manifest{
		Brand:       BrandInfo{DisplayName: in.DisplayName, Command: in.Command},
		GitHost:     GitHostInfo{BaseURL: in.GitHostBase},
		LocalDomain: in.LocalDomain,
	}
	port := 5000

	for _, dir := range in.CloneDirs {
		if skipCloneDirs[dir] || strings.HasPrefix(dir, ".") {
			continue
		}
		svc := matchCompose(dir, services)
		if svc == "" {
			svc = dir
		}
		m.Repos = append(m.Repos, RepoDef{
			Name: dir,
			Git:  group + "/" + dir,
			Services: []ServiceDef{{
				ComposeService: svc,
				Path:           ".",
				SourcePort:     port,
			}},
		})
		port++
	}

	if len(in.CloneDirs) == 0 {
		for _, svc := range services {
			if infraServices[svc] {
				continue
			}
			name := strings.ReplaceAll(svc, ".", "-")
			m.Repos = append(m.Repos, RepoDef{
				Name: name,
				Git:  group + "/" + name,
				Services: []ServiceDef{{
					ComposeService: svc,
					Path:           ".",
					SourcePort:     port,
				}},
			})
			port++
		}
	}

	m.ApplyDefaults()
	return m
}

func composeServiceNames(raw string) []string {
	var doc struct {
		Services map[string]yaml.Node `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		return nil
	}
	names := make([]string, 0, len(doc.Services))
	for name := range doc.Services {
		names = append(names, name)
	}
	return names
}

func matchCompose(dir string, services []string) string {
	folded := strings.ToLower(dir)
	for _, svc := range services {
		s := strings.ToLower(svc)
		if s == folded || strings.ReplaceAll(s, ".", "-") == folded || strings.ReplaceAll(s, ".", "_") == folded {
			return svc
		}
		if strings.TrimSuffix(folded, "-api") == strings.ReplaceAll(s, ".", "-") {
			return svc
		}
	}
	return ""
}

// CommitReminder is the post-wizard message. The CLI never auto-commits.
func CommitReminder(relPath string) string {
	return fmt.Sprintf("Manifest written to %s but not committed. Review it, then:\n  git add %s && git commit -m \"Add msc project manifest\" && git push", relPath, relPath)
}
