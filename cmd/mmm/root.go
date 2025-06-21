package mmm

import (
	"fmt"
	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/version"
	"github.com/meza/minecraft-mod-manager/internal/constants"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/tui"
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
		RunE:    runRoot,
	}
	cobra.MousetrapHelpText = "" // allow the app to run in windows by clicking the exe

	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.AddCommand(initCmd.Command())
	rootCmd.AddCommand(version.Command())

	translateDefaultHelpFacilities(rootCmd)
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

func Execute() error {
	return Command().Execute()
}

func runRoot(_ *cobra.Command, _ []string) error {
	_, err := tui.RunApp().Run()
	return err
}
