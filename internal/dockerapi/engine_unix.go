//go:build !windows

package dockerapi

import (
	"net/http"
	"net/url"
)

func engineHTTPClient(u *url.URL) (*http.Client, string, error) {
	return engineHTTPClientUnix(u)
}
