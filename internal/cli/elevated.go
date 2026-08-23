package cli

import (
	"fmt"
	"runtime"

	"github.com/SoheilHasankhani/msc-cli/internal/hostcerts"
	"github.com/spf13/cobra"
)

func newElevatedDoCmd() *cobra.Command {
	var payload string
	cmd := &cobra.Command{
		Use:    "__elevated-do",
		Short:  "Internal privileged helper (sudo / UAC)",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if payload == "" {
				return fmt.Errorf("__elevated-do requires --payload")
			}
			p, err := hostcerts.ReadPayloadFile(payload)
			if err != nil {
				return err
			}
			return hostcerts.ApplyElevated(p, runtime.GOOS, hostcerts.ExecRunner)
		},
	}
	cmd.Flags().StringVar(&payload, "payload", "", "JSON payload written by the non-elevated process")
	return cmd
}
