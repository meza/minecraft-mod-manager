package list

import (
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
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   i18n.T("cmd.list.short"),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			details := perf.PerformanceDetails{}
			region := perf.StartRegionWithDetails("app.command.list", &details)
			defer func() {
				details["success"] = err == nil
				region.End()
			}()

			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			quiet, err := cmd.Flags().GetBool("quiet")
			if err != nil {
				return err
			}
			debug, err := cmd.Flags().GetBool("debug")
			if err != nil {
				return err
			}

			log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quiet, debug)
			deps := listDeps{
				fs:        afero.NewOsFs(),
				logger:    log,
				telemetry: telemetry.CaptureCommand,
			}

			err = runList(cmd, configPath, quiet, deps)
			return err
		},
	}

	return cmd
}

type listDeps struct {
	fs        afero.Fs
	logger    *logger.Logger
	telemetry func(telemetry.CommandTelemetry)
}

type listEntry struct {
	DisplayName string
	ID          string
	Platform    models.Platform
	Installed   bool
}

func runList(cmd *cobra.Command, configPath string, quiet bool, deps listDeps) error {
	meta := config.NewMetadata(configPath)

	cfg, err := config.ReadConfig(deps.fs, meta)
	if err != nil {
		return err
	}

	lock, err := readLockOrEmpty(deps.fs, meta)
	if err != nil {
		return err
	}

	entries := buildEntries(cfg, lock, meta, deps.fs)
	useTUI := tui.ShouldUseTUI(quiet, cmd.InOrStdin(), cmd.OutOrStdout())
	colorize := useTUI || tui.IsTerminalWriter(cmd.OutOrStdout())
	view := renderListView(entries, colorize)

	if useTUI {
		model := newModel(view)
		program := tea.NewProgram(model, tui.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
		if _, err := program.Run(); err != nil {
			return err
		}
		if len(entries) == 0 {
			deps.logger.Log(view, true)
		}
	} else {
		deps.logger.Log(view, true)
	}

	deps.telemetry(telemetry.CommandTelemetry{
		Command: "list",
		Success: true,
		Extra: map[string]interface{}{
			"numberOfMods": len(entries),
		},
	})

	return nil
}

func readLockOrEmpty(fs afero.Fs, meta config.Metadata) ([]models.ModInstall, error) {
	lockPath := meta.LockPath()
	exists, err := afero.Exists(fs, lockPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []models.ModInstall{}, nil
	}

	return config.ReadLock(fs, meta)
}

func buildEntries(cfg models.ModsJson, lock []models.ModInstall, meta config.Metadata, fs afero.Fs) []listEntry {
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

func isInstalled(mod models.Mod, lock []models.ModInstall, meta config.Metadata, cfg models.ModsJson, fs afero.Fs) bool {
	for _, install := range lock {
		if install.Id != mod.ID || install.Type != mod.Type {
			continue
		}

		if install.FileName == "" {
			return false
		}

		path := filepath.Join(meta.ModsFolderPath(cfg), install.FileName)
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
