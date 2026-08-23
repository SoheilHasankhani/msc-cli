package nginxcfg

import (
	"strconv"
	"strings"
)

// UpstreamVar is the nginx variable for a compose service: $doctor_upstream.
func UpstreamVar(composeService string) string {
	n := strings.TrimSpace(composeService)
	n = strings.ReplaceAll(n, ".", "_")
	n = strings.ReplaceAll(n, "-", "_")
	return n + "_upstream"
}

// DockerHost is the compose DNS name used in Docker Mode.
// Dots become hyphens so identity.api resolves as identity-api.
func DockerHost(composeService string) string {
	return strings.ReplaceAll(strings.TrimSpace(composeService), ".", "-")
}

// DockerUpstream is the proxy_pass value for Docker Mode.
func DockerUpstream(composeService string) string {
	return "http://" + DockerHost(composeService)
}

// SourceUpstream is the proxy_pass value for Source Mode.
func SourceUpstream(port int) string {
	return "http://host.docker.internal:" + strconv.Itoa(port)
}

// HostCandidates are nginx upstream hostnames we accept for a compose service.
func HostCandidates(composeService string) []string {
	out := []string{composeService, DockerHost(composeService)}
	if strings.Contains(composeService, ".") {
		out = append(out, strings.ReplaceAll(composeService, ".", "_"))
	}
	return uniq(out)
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
