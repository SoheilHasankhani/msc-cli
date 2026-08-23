package dockerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/SoheilHasankhani/msc-cli/internal/usererr"
)

// ComposeService is one enabled compose service with its resolved image and container name.
type ComposeService struct {
	Name          string
	ContainerName string
	Image         string
}

type composeConfigJSON struct {
	Services map[string]composeServiceJSON `json:"services"`
}

type composeServiceJSON struct {
	ContainerName string `json:"container_name"`
	Image         string `json:"image"`
}

// Services lists compose services for the selected profile(s) via `docker compose config --format json`.
func (e ExecCompose) Services(ctx context.Context, workDir, composeFile string, opts ComposeRunOpts) ([]ComposeService, error) {
	var out bytes.Buffer
	cmd := e.cmd(ctx, workDir, composeFile, opts.Profiles, "config", "--format", "json")
	cmd.Stdout = &out
	var buf bytes.Buffer
	if e.Stderr != nil {
		cmd.Stderr = io.MultiWriter(e.Stderr, &buf)
	} else {
		cmd.Stderr = &buf
	}
	if err := cmd.Run(); err != nil {
		return nil, usererr.Compose(fmt.Errorf("docker compose config --format json: %w", err), buf.String())
	}
	return parseComposeServicesJSON(out.Bytes())
}

func parseComposeServicesJSON(data []byte) ([]ComposeService, error) {
	var cfg composeConfigJSON
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse compose config json: %w", err)
	}
	out := make([]ComposeService, 0, len(cfg.Services))
	for name, svc := range cfg.Services {
		container := svc.ContainerName
		if container == "" {
			container = name
		}
		out = append(out, ComposeService{
			Name:          name,
			ContainerName: container,
			Image:         svc.Image,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
