package version

import (
	"github.com/meza/minecraft-mod-manager/internal/constants"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/perf"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

func Command() *cobra.Command {
	versionCmd := &cobra.Command{
		Use: "version",
		Short: i18n.T("cmd.version.short", i18n.Tvars{
			Data: &i18n.TData{"appName": constants.APP_NAME},
		}),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			_, span := perf.StartSpan(cmd.Context(), "app.command.version")
			defer func() {
				span.SetAttributes(attribute.Bool("success", err == nil))
				span.End()
			}()

			cmd.Println(environment.AppVersion())
			return nil
		},
	}

	return versionCmd
}
