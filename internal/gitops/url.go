package gitops

import (
	"fmt"
	"net/url"
	"strings"
)

// SSHURL builds an SSH remote. Full git@ / ssh:// URLs are passed through.
func SSHURL(gitHostBase, repo string) (string, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "", fmt.Errorf("repo git path is required")
	}
	if strings.HasPrefix(repo, "git@") || strings.HasPrefix(repo, "ssh://") {
		return repo, nil
	}
	if gitHostBase == "" {
		return "", fmt.Errorf("git_host.base_url is required")
	}
	u, err := url.Parse(gitHostBase)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid git_host.base_url %q", gitHostBase)
	}
	path := strings.TrimPrefix(repo, "/")
	if !strings.HasSuffix(path, ".git") {
		path += ".git"
	}
	return "git@" + u.Host + ":" + path, nil
}
