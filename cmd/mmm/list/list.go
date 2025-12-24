package list

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"go.opentelemetry.io/otel/attribute"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   i18n.T("cmd.list.short"),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			ctx, span := perf.StartSpan(cmd.Context(), "app.command.list")

			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}
			quiet, err := cmd.Flags().GetBool("quiet")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}
			debug, err := cmd.Flags().GetBool("debug")
			if err != nil {
				span.SetAttributes(attribute.Bool("success", false))
				span.End()
				return err
			}

			log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quiet, debug)
			deps := listDeps{
				fs:            afero.NewOsFs(),
				logger:        log,
				telemetry:     telemetry.RecordCommand,
				programRunner: defaultProgramRunner,
			}

			entriesCount, usedTUI, err := runList(ctx, cmd, configPath, quiet, deps)
			span.SetAttributes(attribute.Bool("success", err == nil))
			span.End()

			payload := telemetry.CommandTelemetry{
				Command:     "list",
				Success:     err == nil,
				Error:       err,
				ExitCode:    0,
				Interactive: usedTUI,
			}
			if err != nil {
				payload.ExitCode = 1
			} else {
				payload.Extra = map[string]interface{}{
					"numberOfMods": entriesCount,
				}
			}
			deps.telemetry(payload)

			return err
		},
	}

	return cmd
}

type listDeps struct {
	fs            afero.Fs
	logger        *logger.Logger
	telemetry     func(telemetry.CommandTelemetry)
	programRunner func(model tea.Model, options ...tea.ProgramOption) error
}

func defaultProgramRunner(model tea.Model, options ...tea.ProgramOption) error {
	program := tea.NewProgram(model, options...)
	_, err := program.Run()
	return err
}

type listEntry struct {
	DisplayName string
	ID          string
	Platform    models.Platform
	Installed   bool
}

func runList(ctx context.Context, cmd *cobra.Command, configPath string, quiet bool, deps listDeps) (int, bool, error) {
	meta := config.NewMetadata(configPath)

	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return 0, false, err
	}

	lock, err := readLockOrEmpty(ctx, deps.fs, meta)
	if err != nil {
		return 0, false, err
	}
	logInvalidLockEntries(lock, deps.logger)

	entries := buildEntries(cfg, lock, meta, deps.fs)
	useTUI := tui.ShouldUseTUI(quiet, cmd.InOrStdin(), cmd.OutOrStdout())
	colorize := useTUI || tui.IsTerminalWriter(cmd.OutOrStdout())
	view := renderListView(entries, colorize)

	if err := renderList(ctx, cmd, entries, view, deps, useTUI); err != nil {
		return 0, useTUI, err
	}
	return len(entries), useTUI, nil
}

func readLockOrEmpty(ctx context.Context, fs afero.Fs, meta config.Metadata) ([]models.ModInstall, error) {
	lockPath := meta.LockPath()
	exists, err := afero.Exists(fs, lockPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []models.ModInstall{}, nil
	}

	return config.ReadLock(ctx, fs, meta)
}

func buildEntries(cfg models.ModsJSON, lock []models.ModInstall, meta config.Metadata, fs afero.Fs) []listEntry {
	entries := make([]listEntry, 0, len(cfg.Mods))

	for _, mod := range cfg.Mods {
		displayName := strings.TrimSpace(mod.Name)
		if displayName == "" {
			displayName = mod.ID
		}

		entry := listEntry{
			DisplayName: displayName,
			ID:          mod.ID,
			Platform:    mod.Type,
			Installed:   isInstalled(mod, lock, meta, cfg, fs),
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].DisplayName) < strings.ToLower(entries[j].DisplayName)
	})

	return entries
}

func isInstalled(mod models.Mod, lock []models.ModInstall, meta config.Metadata, cfg models.ModsJSON, fs afero.Fs) bool {
	for _, install := range lock {
		if install.ID != mod.ID || install.Type != mod.Type {
			continue
		}

		normalizedFileName, err := modfilename.Normalize(install.FileName)
		if err != nil {
			return false
		}

		path := filepath.Join(meta.ModsFolderPath(cfg), normalizedFileName)
		exists, err := afero.Exists(fs, path)
		if err != nil {
			return false
		}
		if exists {
			return true
		}
	}

	return false
}

func logInvalidLockEntries(lock []models.ModInstall, log *logger.Logger) {
	for _, install := range lock {
		if _, err := modfilename.Normalize(install.FileName); err != nil {
			name := strings.TrimSpace(install.Name)
			if name == "" {
				name = install.ID
			}
			log.Error(i18n.T("cmd.list.error.invalid_filename_lock", i18n.Tvars{
				Data: &i18n.TData{
					"name": name,
					"file": modfilename.Display(install.FileName),
				},
			}))
		}
	}
}

func renderListView(entries []listEntry, colorize bool) string {
	var builder strings.Builder

	if len(entries) == 0 {
		empty := i18n.T("cmd.list.empty")
		if colorize {
			empty = tui.PlaceholderStyle.Render(empty)
		}
		return empty
	}

	header := i18n.T("cmd.list.header")
	if colorize {
		header = tui.TitleStyle.Render(header)
	}
	builder.WriteString(header)

	for _, entry := range entries {
		builder.WriteString("\n")
		builder.WriteString(renderEntry(entry, colorize))
	}

	return builder.String()
}

func renderEntry(entry listEntry, colorize bool) string {
	icon := "✗"
	key := "cmd.list.entry.missing"
	name := entry.DisplayName
	id := entry.ID
	if entry.Installed {
		icon = "✓"
		key = "cmd.list.entry.installed"
	}

	if colorize {
		if entry.Installed {
			icon = tui.QuestionStyle.Render(icon)
		} else {
			icon = tui.ErrorStyle.Render(icon)
		}
		id = tui.PlaceholderStyle.Copy().PaddingLeft(0).Render(id)
	}

	message := i18n.T(key, i18n.Tvars{
		Data: &i18n.TData{
			"name": name,
			"id":   id,
		},
	})

	return fmt.Sprintf("%s %s", icon, message)
}

func renderList(ctx context.Context, cmd *cobra.Command, entries []listEntry, view string, deps listDeps, useTUI bool) error {
	if !useTUI {
		deps.logger.Log(view, true)
		return nil
	}

	_, tuiSpan := perf.StartSpan(ctx, "tui.list.session")
	model := newModel(view, tuiSpan)
	if err := deps.programRunner(model, tui.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...); err != nil {
		tuiSpan.SetAttributes(attribute.Bool("success", false))
		tuiSpan.End()
		return err
	}
	tuiSpan.SetAttributes(attribute.Bool("success", true))
	tuiSpan.End()
	if len(entries) == 0 {
		deps.logger.Log(view, true)
	}
	return nil
}
