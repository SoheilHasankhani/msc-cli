package main

import (
	"fmt"
	"os"

	"github.com/SoheilHasankhani/msc-cli/internal/cli"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, ui.Error(err.Error()))
		os.Exit(1)
	}
}
