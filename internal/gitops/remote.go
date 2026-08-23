package gitops

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// ParseIdentity turns a git remote into (remote URL, Git host https base).
func ParseIdentity(remote string) (gitRemote, gitHostBase string, err error) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", "", fmt.Errorf("git remote is empty")
	}
	if strings.HasPrefix(remote, "git@") {
		// git@host:group/repo.git
		rest := strings.TrimPrefix(remote, "git@")
		host, _, ok := strings.Cut(rest, ":")
		if !ok || host == "" {
			return "", "", fmt.Errorf("invalid SSH remote %q", remote)
		}
		return remote, "https://" + host, nil
	}
	u, err := url.Parse(remote)
	if err != nil || u.Host == "" {
		return "", "", fmt.Errorf("invalid git remote %q", remote)
	}
	scheme := u.Scheme
	if scheme == "ssh" {
		scheme = "https"
	}
	if scheme == "" {
		scheme = "https"
	}
	return remote, scheme + "://" + u.Host, nil
}

// OriginURL returns `git remote get-url origin` for a working tree.
func OriginURL(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "get-url", "origin")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git remote get-url origin: %s", strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(out.String()), nil
}
