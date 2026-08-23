package dockerapi

import "context"

// Running implements state.ContainerLookup.
func Running(ctx context.Context, c Client, composeService string) (bool, error) {
	list, err := c.ListContainers(ctx)
	if err != nil {
		return false, err
	}
	for _, ctr := range list {
		if ctr.ComposeService == composeService {
			return ctr.Running, nil
		}
	}
	return false, nil
}

// Lookup adapts Client to state.ContainerLookup.
type Lookup struct {
	Client Client
}

// Running reports whether composeService has a running container.
func (l Lookup) Running(ctx context.Context, composeService string) (bool, error) {
	return Running(ctx, l.Client, composeService)
}
