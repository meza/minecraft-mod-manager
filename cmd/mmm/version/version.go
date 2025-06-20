package version

import (
	"github.com/meza/minecraft-mod-manager/internal/constants"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/i18n"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	versionCmd := &cobra.Command{
		Use: "version",
		Short: i18n.T("cmd.version.short", i18n.Tvars{
			Data: &i18n.TData{"appName": constants.APP_NAME},
		}),
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println(environment.AppVersion())
		},
	}

	return versionCmd
}
