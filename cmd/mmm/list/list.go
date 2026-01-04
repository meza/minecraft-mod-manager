package list

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
	"go.opentelemetry.io/otel/attribute"
)

var listWriteString = func(builder *strings.Builder, value string) error {
	_, err := builder.WriteString(value)
	return err
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   i18n.T("cmd.list.short", nil),
		RunE:    runListCommand,
	}

	return cmd
}

type listCommandOptions struct {
	configPath string
	unattended bool
	quiet      bool
	debug      bool
}

func runListCommand(cmd *cobra.Command, _ []string) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.list")

	options, err := listOptionsFromFlags(cmd)
	if err != nil {
		finishListSpan(span, false)
		return err
	}

	deps := defaultListDeps(cmd, options)
	entriesCount, usedTUI, runErr := runList(ctx, cmd, options.configPath, runListOptions{
		unattended: options.unattended,
		quiet:      options.quiet,
	}, deps)
	finishListSpan(span, runErr == nil)
	recordListTelemetry(deps.telemetry, entriesCount, usedTUI, runErr)
	applyListCommandErrorPolicy(cmd, runErr)

	return runErr
}

func applyListCommandErrorPolicy(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
	}
	cmd.SilenceUsage = true
}

func listOptionsFromFlags(cmd *cobra.Command) (listCommandOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return listCommandOptions{}, err
	}
	unattended, err := cmd.Flags().GetBool("unattended")
	if err != nil {
		return listCommandOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return listCommandOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return listCommandOptions{}, err
	}

	return listCommandOptions{
		configPath: configPath,
		unattended: unattended,
		quiet:      quiet,
		debug:      debug,
	}, nil
}

func defaultListDeps(cmd *cobra.Command, options listCommandOptions) listDeps {
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: options.quiet,
		Debug: options.debug,
	})
	return listDeps{
		fs:            common.FS,
		logger:        common.Logger,
		output:        common.Output,
		telemetry:     telemetry.RecordCommand,
		programRunner: defaultProgramRunner,
	}
}

func finishListSpan(span *perf.Span, success bool) {
	span.SetAttributes(attribute.Bool("success", success))
	span.End()
}

func recordListTelemetry(telemetryRecorder func(telemetry.CommandTelemetry), entriesCount int, usedTUI bool, err error) {
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
	telemetryRecorder(payload)
}

type listDeps struct {
	fs            afero.Fs
	logger        *logger.Logger
	output        *output.Output
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
	Status      listEntryStatus
	FileName    string
}

type listDisplayMode int

const (
	listDisplayCLI listDisplayMode = iota
	listDisplayTUI
)

type listEntryStatus int

const (
	listEntryMissing listEntryStatus = iota
	listEntryInstalled
	listEntryHashMismatch
)

func (mode listDisplayMode) UseTUI() bool {
	return mode == listDisplayTUI
}

type runListOptions struct {
	unattended bool
	quiet      bool
}

func runList(ctx context.Context, cmd *cobra.Command, configPath string, options runListOptions, deps listDeps) (int, bool, error) {
	meta := config.NewMetadata(configPath)

	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return 0, false, err
	}

	lock, err := readLockRequired(ctx, deps.fs, meta)
	if err != nil {
		return 0, false, handleLockReadError(err, deps.output)
	}
	if err := logInvalidLockEntries(lock, deps.output); err != nil {
		return 0, false, err
	}

	entries := buildEntries(cfg, lock, meta, deps.fs)
	useTUI := tui.SupportsPrompting(cmd.InOrStdin(), cmd.OutOrStdout()) && !options.unattended
	colorize := useTUI || tui.IsTerminalWriter(cmd.OutOrStdout())
	colorMode := tui.ColorDisabled
	if colorize {
		colorMode = tui.ColorEnabled
	}
	view := renderListView(entries, colorMode)
	displayMode := listDisplayCLI
	if useTUI {
		displayMode = listDisplayTUI
	}

	if err := renderList(ctx, cmd, entries, view, deps, displayMode); err != nil {
		return 0, useTUI, err
	}
	return len(entries), useTUI, nil
}

type lockMissingError struct {
	message string
}

func (err *lockMissingError) Error() string {
	return err.message
}

func handleLockReadError(err error, out *output.Output) error {
	var lockMissing *lockMissingError
	if !errors.As(err, &lockMissing) {
		return err
	}
	if outputErr := out.Error(lockMissing.Error()); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func readLockRequired(ctx context.Context, fs afero.Fs, meta config.Metadata) ([]models.ModInstall, error) {
	lock, err := config.ReadLock(ctx, fs, meta)
	if err == nil {
		return lock, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, &lockMissingError{
			message: i18n.T("cmd.list.error.lock_missing", &i18n.Tvars{
				Data: &i18n.TData{
					"lock_path":       meta.LockPath(),
					"install_command": "mmm install",
				},
			}),
		}
	}
	return nil, err
}

func buildEntries(cfg models.ModsJSON, lock []models.ModInstall, meta config.Metadata, fs afero.Fs) []listEntry {
	entries := make([]listEntry, 0, len(cfg.Mods))

	for _, mod := range cfg.Mods {
		displayName := strings.TrimSpace(mod.Name)
		if displayName == "" {
			displayName = mod.ID
		}

		status := entryStatus(mod, lock, meta, cfg, fs)
		entry := listEntry{
			DisplayName: displayName,
			ID:          mod.ID,
			Platform:    mod.Type,
			Status:      status.Status,
			FileName:    status.FileName,
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].DisplayName) < strings.ToLower(entries[j].DisplayName)
	})

	return entries
}

func isInstalled(mod models.Mod, lock []models.ModInstall, meta config.Metadata, cfg models.ModsJSON, fs afero.Fs) bool {
	return entryStatus(mod, lock, meta, cfg, fs).Status == listEntryInstalled
}

type entryStatusResult struct {
	Status   listEntryStatus
	FileName string
}

func entryStatus(mod models.Mod, lock []models.ModInstall, meta config.Metadata, cfg models.ModsJSON, fs afero.Fs) entryStatusResult {
	for _, install := range lock {
		if install.ID != mod.ID || install.Type != mod.Type {
			continue
		}

		normalizedFileName, err := modfilename.Normalize(install.FileName)
		if err != nil {
			return entryStatusResult{Status: listEntryMissing}
		}

		path := filepath.Join(meta.ModsFolderPath(cfg), normalizedFileName)
		exists, err := afero.Exists(fs, path)
		if err != nil {
			return entryStatusResult{Status: listEntryMissing}
		}
		if !exists {
			return entryStatusResult{Status: listEntryMissing}
		}

		expectedHash := strings.TrimSpace(install.Hash)
		if expectedHash == "" {
			return entryStatusResult{Status: listEntryMissing}
		}

		actualHash, err := sha1ForFile(fs, path)
		if err != nil {
			return entryStatusResult{Status: listEntryMissing}
		}
		if !strings.EqualFold(expectedHash, actualHash) {
			return entryStatusResult{Status: listEntryHashMismatch, FileName: normalizedFileName}
		}

		return entryStatusResult{Status: listEntryInstalled, FileName: normalizedFileName}
	}

	return entryStatusResult{Status: listEntryMissing}
}

func logInvalidLockEntries(lock []models.ModInstall, out *output.Output) error {
	for _, install := range lock {
		if _, err := modfilename.Normalize(install.FileName); err != nil {
			name := strings.TrimSpace(install.Name)
			if name == "" {
				name = install.ID
			}
			if err := out.Error(i18n.T("cmd.list.error.invalid_filename_lock", &i18n.Tvars{
				Data: &i18n.TData{
					"name": name,
					"file": modfilename.Display(install.FileName),
				},
			})); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderListView(entries []listEntry, colorMode tui.ColorMode) string {
	var builder strings.Builder

	if len(entries) == 0 {
		empty := i18n.T("cmd.list.empty", nil)
		empty = tui.RenderIfColorEnabled(colorMode, tui.PlaceholderStyle, empty)
		return empty
	}

	header := i18n.T("cmd.list.header", nil)
	header = tui.RenderIfColorEnabled(colorMode, tui.TitleStyle, header)
	if err := listWriteString(&builder, header); err != nil {
		return ""
	}

	for _, entry := range entries {
		if err := appendListEntry(&builder, entry, colorMode); err != nil {
			return ""
		}
	}

	return builder.String()
}

func appendListEntry(builder *strings.Builder, entry listEntry, colorMode tui.ColorMode) error {
	if err := listWriteString(builder, "\n"); err != nil {
		return err
	}
	if err := listWriteString(builder, renderEntry(entry, colorMode)); err != nil {
		return err
	}
	return nil
}

func renderEntry(entry listEntry, colorMode tui.ColorMode) string {
	icon := tui.ErrorIcon(colorMode)
	key := "cmd.list.entry.missing"
	name := entry.DisplayName
	id := entry.ID
	data := i18n.TData{
		"name": name,
		"id":   id,
	}
	switch entry.Status {
	case listEntryInstalled:
		icon = tui.SuccessIcon(colorMode)
		key = "cmd.list.entry.installed"
	case listEntryHashMismatch:
		key = "cmd.list.entry.hash_mismatch"
		fixCommand := "mmm install"
		fix := i18n.T("cmd.list.entry.hash_mismatch.fix", &i18n.Tvars{
			Data: &i18n.TData{
				"fix_command": fixCommand,
			},
		})
		fix = tui.RenderIfColorEnabled(colorMode, tui.PlaceholderStyle.PaddingLeft(0), fix)
		data["file"] = modfilename.Display(entry.FileName)
		data["platform"] = entry.Platform.String()
		data["fix"] = fix
	}

	id = tui.RenderIfColorEnabled(colorMode, tui.ParenStyle, id)
	data["id"] = id

	message := i18n.T(key, &i18n.Tvars{
		Data: &data,
	})

	return fmt.Sprintf("%s %s", icon, message)
}

func renderList(ctx context.Context, cmd *cobra.Command, entries []listEntry, view string, deps listDeps, displayMode listDisplayMode) error {
	if !displayMode.UseTUI() {
		return deps.output.Log(view, output.LogForce)
	}

	_, tuiSpan := perf.StartSpan(ctx, "interaction.list.session")
	model := newModel(view, tuiSpan)
	if err := deps.programRunner(model, tui.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...); err != nil {
		tuiSpan.SetAttributes(attribute.Bool("success", false))
		tuiSpan.End()
		return err
	}
	tuiSpan.SetAttributes(attribute.Bool("success", true))
	tuiSpan.End()
	if len(entries) == 0 {
		return deps.output.Log(view, output.LogForce)
	}
	return nil
}

func sha1ForFile(fs afero.Fs, path string) (hash string, returnErr error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	hasher := sha1.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
