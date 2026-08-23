//go:build windows

package dockerapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	winio "github.com/Microsoft/go-winio"
)

func engineHTTPClient(u *url.URL) (*http.Client, string, error) {
	switch u.Scheme {
	case "npipe":
		pipe, err := npipePath(u)
		if err != nil {
			return nil, "", err
		}
		return &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return winio.DialPipeContext(ctx, pipe)
				},
			},
			Timeout: 0,
		}, "http://docker", nil
	default:
		return engineHTTPClientUnix(u)
	}
}

func npipePath(u *url.URL) (string, error) {
	if u.Scheme != "npipe" {
		return "", fmt.Errorf("not an npipe URL")
	}
	path := strings.TrimPrefix(u.Path, "//")
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "pipe/")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "", fmt.Errorf("npipe URL missing pipe name")
	}
	return `\\.\pipe\` + path, nil
}
