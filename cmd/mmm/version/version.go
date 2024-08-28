package version

import (
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/constants"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/i18n"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	versionCmd := &cobra.Command{
		Use: "version",
		Short: i18n.T("cmd.version.short", map[string]any{
			"appName": constants.APP_NAME,
		}),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(environment.AppVersion())
		},
	}

	return versionCmd
}
