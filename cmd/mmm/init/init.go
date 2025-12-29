package init

import (
	"fmt"
	"io"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

func Command() *cobra.Command {
	return commandWithRunner(runInitCommand)
}

func defaultRunTea(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	return program.Run()
}

var registerFlagCompletion = func(cmd *cobra.Command, flagName string, completionFunc cobra.CompletionFunc) error {
	return cmd.RegisterFlagCompletionFunc(flagName, completionFunc)
}

var completionWarnWriter io.Writer = os.Stderr

func commandWithRunner(runner initRunner) *cobra.Command {
	var loader loaderFlag

	cmd := &cobra.Command{
		Use:   "init",
		Short: i18n.T("cmd.init.short", nil),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			ctx, span := perf.StartSpan(cmd.Context(), "app.command.init")
			defer func() {
				span.SetAttributes(attribute.Bool("success", err == nil))
				span.End()
			}()

			options, err := readInitOptions(cmd, loader.value)
			if err != nil {
				return err
			}
			promptMode := tui.PromptEnabled
			if options.NonInteractive {
				promptMode = tui.PromptDisabled
			}
			promptAllowed := tui.ShouldPrompt(promptMode, cmd.InOrStdin(), cmd.OutOrStdout())

			common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
				Quiet:           options.Quiet,
				Debug:           options.Debug,
				MinecraftClient: http.DefaultClient,
			})
			deps := newInitDeps(cmd, common, promptAllowed)
			meta := config.NewMetadata(options.ConfigPath)

			err = runner(ctx, cmd, options, deps, meta)
			return err
		},
	}

	addInitFlags(cmd, &loader)
	if !registerInitCompletions(cmd) {
		return cmd
	}

	return cmd
}

func addInitFlags(cmd *cobra.Command, loader *loaderFlag) {
	cmd.Flags().VarP(loader, "loader", "l", i18n.T("cmd.init.usage.loader", &i18n.Tvars{
		Data: &i18n.TData{"loaders": getAllLoaders()},
	}))
	cmd.Flags().StringSliceP("release-types", "r", []string{"release"}, i18n.T("cmd.init.usage.release-types", &i18n.Tvars{
		Data: &i18n.TData{"releaseTypes": getAllReleaseTypes()},
	}))
	cmd.Flags().StringP("game-version", "g", "latest", i18n.T("cmd.init.usage.game-version", nil))
	cmd.Flags().StringP("mods-folder", "m", "mods", i18n.T("cmd.init.usage.mods-folder", nil))
}

func registerInitCompletions(cmd *cobra.Command) bool {
	if err := registerFlagCompletion(cmd, "loader", completeLoaders); err != nil {
		if _, writeErr := fmt.Fprintf(completionWarnWriter, "warning: failed to register loader completion: %v\n", err); writeErr != nil {
			return false
		}
	}
	if err := registerFlagCompletion(cmd, "release-types", completeReleaseTypes); err != nil {
		if _, writeErr := fmt.Fprintf(completionWarnWriter, "warning: failed to register release type completion: %v\n", err); writeErr != nil {
			return false
		}
	}
	return true
}
