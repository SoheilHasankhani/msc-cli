package nginxcfg

import (
	"context"
	"fmt"
	"strings"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/usererr"
)

// Reload sends SIGHUP to the running nginx compose service (never a restart).
// Missing or stopped nginx is a no-op so switch works before the stack is up.
func Reload(ctx context.Context, c dockerapi.Client, composeService string) error {
	if composeService == "" {
		composeService = DefaultNginxService
	}
	list, err := c.ListContainers(ctx)
	if err != nil {
		return err
	}
	for _, ctr := range list {
		if ctr.ComposeService != composeService {
			continue
		}
		if !ctr.Running {
			return nil
		}
		name := strings.TrimPrefix(ctr.Name, "/")
		if err := c.SignalContainer(ctx, name, "HUP"); err != nil {
			return usererr.NginxReload(fmt.Errorf("reload nginx (%s): %w", name, err))
		}
		return nil
	}
	return nil
}
