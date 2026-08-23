package complete

import "slices"

// ComposeCommands are common docker compose subcommands offered after `msc compose`.
var ComposeCommands = []string{
	"attach",
	"build",
	"config",
	"cp",
	"create",
	"down",
	"events",
	"exec",
	"images",
	"kill",
	"logs",
	"ls",
	"pause",
	"port",
	"ps",
	"pull",
	"push",
	"restart",
	"rm",
	"run",
	"scale",
	"start",
	"stop",
	"top",
	"unpause",
	"up",
	"watch",
}

var composeServiceCommands = map[string]struct{}{
	"attach":  {},
	"build":   {},
	"down":    {},
	"exec":    {},
	"kill":    {},
	"logs":    {},
	"pause":   {},
	"pull":    {},
	"restart": {},
	"rm":      {},
	"run":     {},
	"scale":   {},
	"start":   {},
	"stop":    {},
	"top":     {},
	"unpause": {},
	"up":      {},
}

// ComposeTakesServices reports whether a compose subcommand accepts service names.
func ComposeTakesServices(subcommand string) bool {
	_, ok := composeServiceCommands[subcommand]
	return ok
}

// ComposeSubcommands returns a copy of ComposeCommands (sorted).
func ComposeSubcommands() []string {
	out := slices.Clone(ComposeCommands)
	slices.Sort(out)
	return out
}
