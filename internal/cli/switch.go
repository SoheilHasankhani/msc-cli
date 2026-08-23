package cli

import (
	"context"
	"fmt"

	"github.com/SoheilHasankhani/msc-cli/internal/dockerapi"
	"github.com/SoheilHasankhani/msc-cli/internal/project"
	"github.com/SoheilHasankhani/msc-cli/internal/switchsvc"
	"github.com/SoheilHasankhani/msc-cli/internal/ui"
	"github.com/spf13/cobra"
)

func newSwitchCmd() *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:               "switch SERVICE",
		Short:             "Route a service to Source Mode or Docker Mode",
		Long:              "Without --to, flips the live mode (docker ↔ source) from nginx + Docker. --to source|docker sets the mode explicitly. Writes the generated nginx upstream overlay, stops or starts the compose service, and reloads nginx with SIGHUP. In Source Mode the process must listen on 0.0.0.0, not 127.0.0.1.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			ren := ui.For(cmd.OutOrStdout())
			p, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			mode, err := switchsvc.ParseMode(to)
			if err != nil {
				return err
			}
			engine, err := dockerapi.NewEngine()
			if err != nil {
				return err
			}
			defer engine.Close()

			var res switchsvc.Result
			err = ui.WithSpinner(cmd.Context(), cmd.ErrOrStderr(), "Switching "+args[0], func(ctx context.Context) error {
				var runErr error
				res, runErr = switchsvc.Run(ctx, switchsvc.Options{
					Project: p,
					Name:    args[0],
					To:      mode,
					Docker:  engine,
					Compose: dockerapi.ExecCompose{},
				})
				return runErr
			})
			if err != nil {
				return err
			}

			if err := recordSwitch(cmd.Context(), p, engine, res); err != nil {
				return err
			}

			var msg string
			if res.From != "" {
				msg = fmt.Sprintf("switched %s %s → %s", res.Service.ComposeService, res.From, res.Mode)
			} else {
				msg = fmt.Sprintf("switched %s to %s", res.Service.ComposeService, res.Mode)
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.SuccessLine(ren, msg, res.Reminder))
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "target mode: source or docker (omit to toggle)")
	_ = cmd.RegisterFlagCompletionFunc("to", completeSwitchModes)
	return cmd
}

func recordSwitch(ctx context.Context, p *project.Context, c dockerapi.Client, res switchsvc.Result) error {
	return syncProjectCache(ctx, p, c, res.Service.ComposeService)
}
