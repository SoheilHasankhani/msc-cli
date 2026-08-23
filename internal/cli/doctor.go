package cli

import (
	"context"
	"fmt"

	"github.com/SoheilHasankhani/msc-cli/internal/doctor"
	"github.com/SoheilHasankhani/msc-cli/internal/elevate"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var fix, noElevate bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Report (and optionally repair) the developer machine and project setup",
		Long:  "Checks Git, SSH, Docker, and — when a project is selected — hosts, the local CA, and host-gateway. --fix applies only safe repairs. It never installs Docker, Git, or .NET. Hosts-file and OS trust-store writes re-invoke this binary under sudo / UAC.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ren := ui.For(cmd.OutOrStdout())
			opt := doctor.Options{Fix: fix, NoElevate: noElevate, Elevate: elevate.NewProcess()}
			if name, err := cmd.Flags().GetString("project"); err == nil && name != "" {
				p, err := resolveProject(cmd)
				if err != nil {
					return err
				}
				opt.Project = p
			}
			var report doctor.Report
			var err error
			run := func(ctx context.Context) error {
				report, err = doctor.Run(ctx, opt)
				return err
			}
			if fix {
				err = ui.WithSpinner(cmd.Context(), cmd.ErrOrStderr(), "Applying doctor fixes", run)
			} else {
				err = run(cmd.Context())
			}
			if err != nil {
				return err
			}
			if _, err := fmt.Fprint(cmd.OutOrStdout(), doctor.Format(report, ren)); err != nil {
				return err
			}
			if report.HasFail() {
				return fmt.Errorf("doctor found problems")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "apply safe repairs (overlay, cert files, hosts block, OS/NSS CA trust)")
	cmd.Flags().BoolVar(&noElevate, "no-elevate", false, "with --fix, skip sudo/UAC (still write overlay and cert files)")
	return cmd
}
