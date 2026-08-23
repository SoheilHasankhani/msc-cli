package hostcerts

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var serverNameRE = regexp.MustCompile(`(?i)server_name\s+([^;]+);`)

// BeginMarker / EndMarker delimit one project's additive hosts block.
func BeginMarker(project string) string { return "# msc-begin " + project }
func EndMarker(project string) string   { return "# msc-end " + project }

// SystemHostsPath is the OS hosts file. goos is runtime.GOOS or a test override.
func SystemHostsPath(goos string) string {
	if goos == "windows" {
		return `C:\Windows\System32\drivers\etc\hosts`
	}
	return "/etc/hosts"
}

// CollectHostnames returns a sorted unique list from local_domain and nginx server_name lines.
func CollectHostnames(localDomain, nginxContent string) []string {
	domain := strings.TrimSpace(localDomain)
	set := map[string]struct{}{}
	if domain != "" {
		set[domain] = struct{}{}
	}
	for _, m := range serverNameRE.FindAllStringSubmatch(nginxContent, -1) {
		for _, part := range strings.Fields(m[1]) {
			name := strings.TrimSpace(part)
			if !publicHostname(name, domain) {
				continue
			}
			set[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func publicHostname(name, localDomain string) bool {
	if name == "" || name == "_" {
		return false
	}
	if strings.HasSuffix(name, "_internal") || strings.Contains(name, "metrics_status") {
		return false
	}
	if strings.Contains(name, ".") {
		return true
	}
	return localDomain != "" && name == localDomain
}

// Missing returns wanted names that do not appear anywhere in the hosts file.
func Missing(hostsContent string, names []string) []string {
	var miss []string
	for _, n := range names {
		if !containsName(hostsContent, n) {
			miss = append(miss, n)
		}
	}
	return miss
}

func containsName(hostsContent, name string) bool {
	for _, line := range strings.Split(hostsContent, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		for _, field := range strings.Fields(trim) {
			if field == name {
				return true
			}
		}
	}
	return false
}

// UpsertBlock replaces or appends the marked block for project. Other blocks are untouched.
func UpsertBlock(hostsContent, project string, names []string) string {
	begin, end := BeginMarker(project), EndMarker(project)
	block := renderBlock(project, names)
	start := strings.Index(hostsContent, begin)
	if start < 0 {
		body := strings.TrimRight(hostsContent, "\n")
		if body != "" {
			body += "\n"
		}
		return body + "\n" + block
	}
	rest := hostsContent[start:]
	relEnd := strings.Index(rest, end)
	if relEnd < 0 {
		return hostsContent[:start] + block
	}
	stop := start + relEnd + len(end)
	for stop < len(hostsContent) && (hostsContent[stop] == '\n' || hostsContent[stop] == '\r') {
		stop++
	}
	return hostsContent[:start] + block + hostsContent[stop:]
}

func renderBlock(project string, names []string) string {
	uniq := append([]string(nil), names...)
	sort.Strings(uniq)
	line := "127.0.0.1"
	if len(uniq) == 0 {
		line += " " + project
	} else {
		line += " " + strings.Join(uniq, " ")
	}
	return fmt.Sprintf("%s\n%s\n%s\n", BeginMarker(project), line, EndMarker(project))
}
