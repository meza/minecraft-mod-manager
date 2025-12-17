package mmm

import (
	"fmt"
	addCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/add"
	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	installCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	listCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/list"
	removeCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/remove"
	updateCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/update"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/version"
	"github.com/meza/minecraft-mod-manager/internal/constants"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"os"
	"strings"
)

func Command() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     constants.COMMAND_NAME,
		Short:   i18n.T("app.description"),
		Version: environment.AppVersion(),
	}

	rootCmd.PersistentFlags().StringP("config", "c", "./modlist.json", "An alternative JSON file containing the configuration")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all output")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug messages")
	rootCmd.PersistentFlags().Bool("perf", false, "Write a performance log (mmm-perf.json) when the command exits")
	rootCmd.PersistentFlags().String("perf-out-dir", "", "Directory to write mmm-perf.json (defaults to the config file directory)")

	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.AddCommand(addCmd.Command())
	rootCmd.AddCommand(initCmd.Command())
	rootCmd.AddCommand(installCmd.Command())
	rootCmd.AddCommand(listCmd.Command())
	rootCmd.AddCommand(removeCmd.Command())
	rootCmd.AddCommand(updateCmd.Command())
	rootCmd.AddCommand(version.Command())

	translateDefaultHelpFacilities(rootCmd)
	appendHelpURLFooter(rootCmd)
	fixFlagUsageAlignment(rootCmd)

	return rootCmd
}

func translateDefaultHelpFacilities(rootCmd *cobra.Command) {
	subcommands := rootCmd.Commands()
	allCommands := make([]*cobra.Command, 0, len(subcommands)+1)
	allCommands = append(allCommands, rootCmd)
	allCommands = append(allCommands, subcommands...)

	for _, cmd := range allCommands {
		cmd.InitDefaultHelpFlag()
		flags := cmd.Flags()
		flags.Lookup("help").Usage = i18n.T("cmd.help.template", i18n.Tvars{
			Data: &i18n.TData{"command": cmd.Name()},
		})
	}

	rootCmd.InitDefaultHelpCmd()
	helpCmd, _, e := rootCmd.Find([]string{"help"})

	if e == nil {
		helpCmd.Short = i18n.T("cmd.help.usage.short")
		helpCmd.Long = i18n.T("cmd.help.usage.long", i18n.Tvars{
			Data: &i18n.TData{"appName": rootCmd.Name()},
		})
		helpCmd.Run = func(c *cobra.Command, args []string) {
			cmd, _, e := c.Root().Find(args)
			if cmd == nil || e != nil {
				c.PrintErrln(i18n.T("cmd.help.error", i18n.Tvars{
					Data: &i18n.TData{"topic": fmt.Sprintf("%#q", args)},
				}) + "\n")
				cobra.CheckErr(c.Root().Usage())
			} else {
				cmd.InitDefaultHelpFlag()    // make possible 'help' flag to be shown
				cmd.InitDefaultVersionFlag() // make possible 'version' flag to be shown
				cobra.CheckErr(cmd.Help())
			}
		}
	}
}

func fixFlagUsageAlignment(rootCmd *cobra.Command) {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	usageTemplate := rootCmd.UsageTemplate()
	usageTemplate = strings.ReplaceAll(usageTemplate, ".FlagUsages", fmt.Sprintf(".FlagUsagesWrapped %d", width))
	rootCmd.SetUsageTemplate(usageTemplate)
}

func appendHelpURLFooter(rootCmd *cobra.Command) {
	template := strings.TrimRight(rootCmd.HelpTemplate(), "\n")
	footer := i18n.T("cmd.help.more_info", i18n.Tvars{
		Data: &i18n.TData{"helpUrl": environment.HelpURL()},
	})

	rootCmd.SetHelpTemplate(template + "\n\n" + footer + "\n")
}

func Execute() error {
	return Command().Execute()
}
