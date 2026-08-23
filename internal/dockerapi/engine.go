package dockerapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/usererr"
)

// ErrNoContainer means no compose container exists for that service (stack never started).
var ErrNoContainer = errors.New("no container")

const composeServiceLabel = "com.docker.compose.service"

// Engine talks to the Docker Engine HTTP API over the default socket or DOCKER_HOST.
// Desktop and native Linux Engine share this API; there is no Desktop-specific client.
type Engine struct {
	http *http.Client
	base string
}

// NewEngine connects using DOCKER_HOST, the active Docker CLI context, or the Engine socket.
func NewEngine() (*Engine, error) {
	home, _ := os.UserHomeDir()
	host := ResolveHost(os.Getenv, home)
	u, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("docker host %q: %w", host, err)
	}
	httpClient, base, err := engineHTTPClient(u)
	if err != nil {
		return nil, err
	}
	return &Engine{http: httpClient, base: base}, nil
}

// Close implements io.Closer for API symmetry with the SDK client.
func (e *Engine) Close() error { return nil }

func engineHTTPClientUnix(u *url.URL) (*http.Client, string, error) {
	base := "http://docker"
	switch u.Scheme {
	case "unix":
		return &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", u.Path)
				},
			},
			Timeout: 0,
		}, base, nil
	case "tcp", "http", "https":
		base = strings.TrimRight(u.String(), "/")
		if u.Scheme == "tcp" {
			base = "http://" + u.Host
		}
		return &http.Client{Timeout: 0}, base, nil
	default:
		return nil, "", fmt.Errorf("unsupported DOCKER_HOST scheme %q", u.Scheme)
	}
}

type containerJSON struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	State  string            `json:"State"`
	Labels map[string]string `json:"Labels"`
}

func (e *Engine) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.base+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, usererr.Docker(err, "")
	}
	return resp, nil
}

func (e *Engine) post(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.base+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, usererr.Docker(err, "")
	}
	return resp, nil
}

// Ping hits the Engine _ping endpoint.
func (e *Engine) Ping(ctx context.Context) error {
	resp, err := e.get(ctx, "/_ping")
	if err != nil {
		return usererr.Docker(fmt.Errorf("docker ping: %w", err), "")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return dockerError("ping", resp)
	}
	return nil
}

// ListContainers returns compose-labeled containers (including stopped).
func (e *Engine) ListContainers(ctx context.Context) ([]Container, error) {
	resp, err := e.get(ctx, "/containers/json?all=true")
	if err != nil {
		return nil, fmt.Errorf("docker list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, dockerError("list containers", resp)
	}
	var list []containerJSON
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	out := make([]Container, 0, len(list))
	for _, c := range list {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		out = append(out, Container{
			Name:           name,
			ComposeService: c.Labels[composeServiceLabel],
			Running:        c.State == "running",
		})
	}
	return out, nil
}

func (e *Engine) find(ctx context.Context, composeService string) (string, error) {
	q := url.QueryEscape(`{"label":["` + composeServiceLabel + `=` + composeService + `"]}`)
	resp, err := e.get(ctx, "/containers/json?all=true&filters="+q)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", dockerError("find container", resp)
	}
	var list []containerJSON
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", fmt.Errorf("no container for compose service %q: %w", composeService, ErrNoContainer)
	}
	return list[0].ID, nil
}

// StopService stops the container for a compose service.
func (e *Engine) StopService(ctx context.Context, composeService string) error {
	id, err := e.find(ctx, composeService)
	if err != nil {
		return err
	}
	resp, err := e.post(ctx, "/containers/"+id+"/stop")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotModified {
		return dockerError("stop", resp)
	}
	return nil
}

// StartService starts the container for a compose service.
func (e *Engine) StartService(ctx context.Context, composeService string) error {
	id, err := e.find(ctx, composeService)
	if err != nil {
		return err
	}
	resp, err := e.post(ctx, "/containers/"+id+"/start")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotModified {
		return dockerError("start", resp)
	}
	return nil
}

// SignalContainer sends a signal (e.g. SIGHUP) to a container by name or ID.
func (e *Engine) SignalContainer(ctx context.Context, containerName, signal string) error {
	resp, err := e.post(ctx, "/containers/"+url.PathEscape(containerName)+"/kill?signal="+url.QueryEscape(signal))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return dockerError("signal", resp)
	}
	return nil
}

// Pull starts an image pull and returns the JSON-lines progress stream.
func (e *Engine) Pull(ctx context.Context, ref string) (io.ReadCloser, error) {
	image, tag := splitRef(ref)
	q := "/images/create?fromImage=" + url.QueryEscape(image)
	if tag != "" {
		q += "&tag=" + url.QueryEscape(tag)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.base+q, nil)
	if err != nil {
		return nil, err
	}
	if auth, err := RegistryAuthHeader(ref); err == nil && auth != "" {
		req.Header.Set("X-Registry-Auth", auth)
	}
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, usererr.Docker(err, "")
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, dockerError("pull", resp)
	}
	return resp.Body, nil
}

func splitRef(ref string) (image, tag string) {
	if i := strings.LastIndex(ref, ":"); i > 0 && !strings.Contains(ref[i:], "/") {
		return ref[:i], ref[i+1:]
	}
	return ref, "latest"
}

func dockerError(op string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if msg := parseDockerAPIMessage(body); msg != "" {
		return fmt.Errorf("docker %s: %s", op, msg)
	}
	return fmt.Errorf("docker %s: %s: %s", op, resp.Status, strings.TrimSpace(string(body)))
}

func parseDockerAPIMessage(body []byte) string {
	var v struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &v); err != nil || v.Message == "" {
		return ""
	}
	return strings.TrimSpace(v.Message)
}
