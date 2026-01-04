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
	"go.opentelemetry.io/otel/attribute"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/mmmignore"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

var runInteractiveInit = initCmd.RunInteractiveInit

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
	entriesCount, usedInteractive, runErr := runList(ctx, cmd, options.configPath, runListOptions{
		unattended: options.unattended,
		quiet:      options.quiet,
	}, deps)
	finishListSpan(span, runErr == nil)
	recordListTelemetry(deps.telemetry, entriesCount, usedInteractive, runErr)
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

type initRequest struct {
	configPath string
}

type initRunner func(context.Context, *cobra.Command, initRequest) error

type listDeps struct {
	fs        afero.Fs
	output    *output.Output
	logger    *logger.Logger
	telemetry func(telemetry.CommandTelemetry)
	runTea    func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
	runInit   initRunner
}

func defaultListDeps(cmd *cobra.Command, options listCommandOptions) listDeps {
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: options.quiet,
		Debug: options.debug,
	})
	return listDeps{
		fs:        common.FS,
		output:    common.Output,
		logger:    common.Logger,
		telemetry: telemetry.RecordCommand,
		runTea:    defaultRunTea,
		runInit: func(ctx context.Context, command *cobra.Command, request initRequest) error {
			return runInteractiveInit(ctx, command, initCmd.InteractiveInitDeps{
				FS:              common.FS,
				Output:          common.Output,
				Logger:          common.Logger,
				MinecraftClient: common.MinecraftClient,
				RunTea:          defaultRunTea,
			}, initCmd.InteractiveInitOptions{
				ConfigPath: request.configPath,
				Quiet:      options.quiet,
				Debug:      options.debug,
			})
		},
	}
}

func defaultRunTea(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	return program.Run()
}

var runTeaProgram = defaultRunTea

func finishListSpan(span *perf.Span, success bool) {
	span.SetAttributes(attribute.Bool("success", success))
	span.End()
}

func recordListTelemetry(telemetryRecorder func(telemetry.CommandTelemetry), entriesCount int, usedInteractive bool, err error) {
	payload := telemetry.CommandTelemetry{
		Command:     "list",
		Success:     err == nil,
		Error:       err,
		ExitCode:    0,
		Interactive: usedInteractive,
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

type listEntry struct {
	DisplayName string
	ID          string
	Platform    models.Platform
	Status      listEntryStatus
	FileName    string
}

type listEntryStatus int

const (
	listEntryMissing listEntryStatus = iota
	listEntryInstalled
	listEntryHashMismatch
)

type modsFolderReadError struct {
	path string
	err  error
}

func (err *modsFolderReadError) Error() string {
	return fmt.Sprintf("could not read %s: %s", err.path, err.err)
}

func (err *modsFolderReadError) Unwrap() error {
	return err.err
}

type runListOptions struct {
	unattended bool
	quiet      bool
}

type listConfigState struct {
	Config         models.ModsJSON
	Lock           []models.ModInstall
	ShouldContinue bool
}

func runList(ctx context.Context, cmd *cobra.Command, configPath string, options runListOptions, deps listDeps) (int, bool, error) {
	meta := config.NewMetadata(configPath)
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: options.unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	configState, usedInteractive, err := ensureListConfig(ctx, cmd, options, deps, meta, mode)
	if err != nil {
		return 0, usedInteractive, err
	}
	if !configState.ShouldContinue {
		return 0, usedInteractive, nil
	}

	invalidLockWarnings := invalidLockEntries(configState.Lock)
	if warningErr := writeInvalidLockWarnings(cmd, deps, invalidLockWarnings); warningErr != nil {
		return 0, usedInteractive, warningErr
	}

	entries, entriesErr := buildEntries(configState.Config, configState.Lock, meta, deps.fs)
	if entriesErr != nil {
		return 0, usedInteractive, handleListModsFolderFailure(cmd, deps, entriesErr)
	}
	colorMode := colorModeForWriter(cmd)
	listView := renderListView(entries, colorMode)
	if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{listView}); outputErr != nil {
		return 0, usedInteractive, outputErr
	}

	unmanagedFiles, unmanagedErr := listUnmanagedFiles(deps.fs, meta, configState.Config, configState.Lock)
	if unmanagedErr != nil {
		return 0, usedInteractive, handleListModsFolderPartialFailure(cmd, deps, unmanagedErr)
	}
	if len(unmanagedFiles) > 0 {
		noticeView := renderUnmanagedNotice(unmanagedFiles, colorMode)
		if noticeErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{"\n" + noticeView}); noticeErr != nil {
			return 0, usedInteractive, noticeErr
		}
	}

	return len(entries), usedInteractive, nil
}

func ensureListConfig(ctx context.Context, cmd *cobra.Command, options runListOptions, deps listDeps, meta config.Metadata, mode interaction.ExecutionMode) (listConfigState, bool, error) {
	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err == nil {
		lock, lockErr := readLockRequired(ctx, deps.fs, meta)
		if lockErr != nil {
			return listConfigState{}, mode == interaction.ExecutionModeInteractive, handleListFailure(cmd, deps, lockErr)
		}
		return listConfigState{Config: cfg, Lock: lock, ShouldContinue: true}, mode == interaction.ExecutionModeInteractive, nil
	}

	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return listConfigState{}, mode == interaction.ExecutionModeInteractive, handleListFailure(cmd, deps, err)
	}

	promptErr := configMissingPromptError(options, cmd, meta)
	if promptErr != nil {
		if outputErr := writeConfigMissingOutput(cmd, deps, meta); outputErr != nil {
			return listConfigState{}, mode == interaction.ExecutionModeInteractive, outputErr
		}
		return listConfigState{}, mode == interaction.ExecutionModeInteractive, clierrors.MarkHandled(promptErr)
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, meta)
	if err != nil {
		return listConfigState{}, true, err
	}
	if canceled || !confirmed {
		return listConfigState{ShouldContinue: false}, true, nil
	}

	if deps.runInit == nil {
		return listConfigState{}, true, errors.New("missing init runner")
	}
	if runErr := deps.runInit(ctx, cmd, initRequest{configPath: meta.ConfigPath}); runErr != nil {
		if errors.Is(runErr, initCmd.ErrInitCanceled) {
			return listConfigState{ShouldContinue: false}, true, nil
		}
		return listConfigState{}, true, runErr
	}

	cfg, err = config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return listConfigState{}, true, handleListFailure(cmd, deps, err)
	}
	lock, lockErr := readLockRequired(ctx, deps.fs, meta)
	if lockErr != nil {
		return listConfigState{}, true, handleListFailure(cmd, deps, lockErr)
	}

	return listConfigState{Config: cfg, Lock: lock, ShouldContinue: true}, true, nil
}

func configMissingPromptError(options runListOptions, cmd *cobra.Command, meta config.Metadata) error {
	return interaction.CheckConfigInitGate(meta, interaction.ConfigInitGate{
		Unattended: options.unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
		UnattendedError: func(meta config.Metadata) error {
			return errors.New(i18n.T("cmd.list.error.config_missing", &i18n.Tvars{
				Data: &i18n.TData{"configPath": meta.ConfigPath},
			}))
		},
		NoTTYError: func(meta config.Metadata) error {
			return errors.New(i18n.T("cmd.list.error.config_missing", &i18n.Tvars{
				Data: &i18n.TData{"configPath": meta.ConfigPath},
			}))
		},
	})
}

type lockMissingError struct {
	message string
}

func (err *lockMissingError) Error() string {
	return err.message
}

func handleListFailure(cmd *cobra.Command, deps listDeps, err error) error {
	if outputErr := writeListFailureOutput(cmd, deps, err); outputErr != nil {
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

func buildEntries(cfg models.ModsJSON, lock []models.ModInstall, meta config.Metadata, fs afero.Fs) ([]listEntry, error) {
	entries := make([]listEntry, 0, len(cfg.Mods))

	for _, mod := range cfg.Mods {
		displayName := strings.TrimSpace(mod.Name)
		if displayName == "" {
			displayName = mod.ID
		}

		status, statusErr := entryStatus(mod, lock, meta, cfg, fs)
		if statusErr != nil {
			return nil, statusErr
		}
		entry := listEntry{
			DisplayName: displayName,
			ID:          mod.ID,
			Platform:    mod.Type,
			Status:      status.Status,
			FileName:    status.FileName,
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i int, j int) bool {
		return strings.ToLower(entries[i].DisplayName) < strings.ToLower(entries[j].DisplayName)
	})

	return entries, nil
}

func isInstalled(mod models.Mod, lock []models.ModInstall, meta config.Metadata, cfg models.ModsJSON, fs afero.Fs) bool {
	status, err := entryStatus(mod, lock, meta, cfg, fs)
	if err != nil {
		return false
	}
	return status.Status == listEntryInstalled
}

type entryStatusResult struct {
	Status   listEntryStatus
	FileName string
}

func entryStatus(mod models.Mod, lock []models.ModInstall, meta config.Metadata, cfg models.ModsJSON, fs afero.Fs) (entryStatusResult, error) {
	for _, install := range lock {
		if install.ID != mod.ID || install.Type != mod.Type {
			continue
		}

		normalizedFileName, ok := normalizeLockFileName(install.FileName)
		if !ok {
			return entryStatusResult{Status: listEntryMissing}, nil
		}

		path := filepath.Join(meta.ModsFolderPath(cfg), normalizedFileName)
		exists, err := afero.Exists(fs, path)
		if err != nil {
			return entryStatusResult{}, &modsFolderReadError{path: meta.ModsFolderPath(cfg), err: err}
		}
		if !exists {
			return entryStatusResult{Status: listEntryMissing}, nil
		}

		expectedHash := strings.TrimSpace(install.Hash)
		if expectedHash == "" {
			return entryStatusResult{Status: listEntryMissing}, nil
		}

		actualHash, err := sha1ForFile(fs, path)
		if err != nil {
			return entryStatusResult{}, &modsFolderReadError{path: meta.ModsFolderPath(cfg), err: err}
		}
		if !strings.EqualFold(expectedHash, actualHash) {
			return entryStatusResult{Status: listEntryHashMismatch, FileName: normalizedFileName}, nil
		}

		return entryStatusResult{Status: listEntryInstalled, FileName: normalizedFileName}, nil
	}

	return entryStatusResult{Status: listEntryMissing}, nil
}

func normalizeLockFileName(fileName string) (string, bool) {
	normalizedFileName, err := modfilename.Normalize(fileName)
	if err != nil {
		return "", false
	}
	return normalizedFileName, true
}

func invalidLockEntries(lock []models.ModInstall) []string {
	warnings := []string{}
	for _, install := range lock {
		if _, err := modfilename.Normalize(install.FileName); err != nil {
			name := strings.TrimSpace(install.Name)
			if name == "" {
				name = install.ID
			}
			warnings = append(warnings, i18n.T("cmd.list.error.invalid_filename_lock", &i18n.Tvars{
				Data: &i18n.TData{
					"name": name,
					"file": modfilename.Display(install.FileName),
				},
			}))
		}
	}
	return warnings
}

func writeInvalidLockWarnings(cmd *cobra.Command, deps listDeps, warnings []string) error {
	if len(warnings) == 0 {
		return nil
	}
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), warnings)
}

func renderListView(entries []listEntry, colorMode view.ColorMode) string {
	var builder strings.Builder

	if len(entries) == 0 {
		empty := i18n.T("cmd.list.empty", nil)
		empty = view.RenderIfColorEnabled(colorMode, view.PlaceholderStyle, empty)
		return empty
	}

	header := i18n.T("cmd.list.header", nil)
	header = view.RenderIfColorEnabled(colorMode, view.TitleStyle, header)
	if err := view.WriteString(&builder, header); err != nil {
		return ""
	}
	if err := view.WriteString(&builder, "\n"); err != nil {
		return ""
	}

	for index, entry := range entries {
		if index > 0 {
			if err := view.WriteString(&builder, "\n"); err != nil {
				return ""
			}
		}
		if err := view.WriteString(&builder, renderEntry(entry, colorMode)); err != nil {
			return ""
		}
	}

	return builder.String()
}

func renderEntry(entry listEntry, colorMode view.ColorMode) string {
	icon := view.ErrorIcon(colorMode)
	key := "cmd.list.entry.missing"

	id := view.RenderIfColorEnabled(colorMode, view.ParenStyle, entry.ID)
	platform := view.RenderIfColorEnabled(colorMode, view.ParenStyle, entry.Platform.String())
	data := i18n.TData{
		"name":     entry.DisplayName,
		"id":       id,
		"platform": platform,
	}

	switch entry.Status {
	case listEntryInstalled:
		icon = view.SuccessIcon(colorMode)
		key = "cmd.list.entry.installed"
	case listEntryHashMismatch:
		key = "cmd.list.entry.hash_mismatch"
		data["fix_command"] = "mmm install"
	}

	message := i18n.T(key, &i18n.Tvars{
		Data: &data,
	})

	return fmt.Sprintf("%s %s", icon, message)
}

func listUnmanagedFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall) ([]string, error) {
	candidates, err := listJarFiles(fs, meta, cfg)
	if err != nil {
		return nil, err
	}

	unmanaged := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if fileIsManaged(candidate, lock) {
			continue
		}
		unmanaged = append(unmanaged, candidate)
	}
	return unmanaged, nil
}

func listJarFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON) ([]string, error) {
	allEntries, err := afero.ReadDir(fs, meta.ModsFolderPath(cfg))
	if err != nil {
		return nil, &modsFolderReadError{path: meta.ModsFolderPath(cfg), err: err}
	}

	candidates := make([]string, 0, len(allEntries))
	for _, entry := range allEntries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
			continue
		}
		candidates = append(candidates, filepath.Join(meta.ModsFolderPath(cfg), entry.Name()))
	}

	patterns, err := mmmignore.ListPatterns(fs, meta.Dir())
	if err != nil {
		return nil, &modsFolderReadError{path: filepath.Join(meta.Dir(), ".mmmignore"), err: err}
	}

	filtered := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if mmmignore.IsIgnored(meta.ModsFolderPath(cfg), candidate, patterns) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered, nil
}

func fileIsManaged(filePath string, lock []models.ModInstall) bool {
	fileName := filepath.Base(filePath)
	for _, install := range lock {
		if install.FileName == fileName {
			return true
		}
	}
	return false
}

func renderUnmanagedNotice(files []string, colorMode view.ColorMode) string {
	if len(files) == 0 {
		return ""
	}

	var builder strings.Builder

	header := i18n.T("cmd.list.unmanaged.header", nil)
	header = view.RenderIfColorEnabled(colorMode, view.TitleStyle, header)
	if err := view.WriteString(&builder, header); err != nil {
		return ""
	}
	if err := view.WriteString(&builder, "\n"); err != nil {
		return ""
	}

	icon := view.ErrorIcon(colorMode)
	for index, file := range files {
		if index > 0 {
			if err := view.WriteString(&builder, "\n"); err != nil {
				return ""
			}
		}
		entry := fmt.Sprintf("%s %s", icon, filepath.Base(file))
		if err := view.WriteString(&builder, entry); err != nil {
			return ""
		}
	}

	if err := view.WriteString(&builder, "\n\n"); err != nil {
		return ""
	}

	description := i18n.T("cmd.list.unmanaged.description", nil)
	if err := view.WriteString(&builder, description); err != nil {
		return ""
	}
	if err := view.WriteString(&builder, "\n"); err != nil {
		return ""
	}

	cta := i18n.T("cmd.list.unmanaged.cta", nil)
	if colorMode.Enabled() {
		cta = view.CtaStyle.Render(cta)
	}
	if err := view.WriteString(&builder, cta); err != nil {
		return ""
	}

	return builder.String()
}

func writeListFailureOutput(cmd *cobra.Command, deps listDeps, err error) error {
	colorMode := colorModeForWriter(cmd)
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.list.error.failed", &i18n.Tvars{
		Data: &i18n.TData{"reason": err.Error()},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.list.error.failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func handleListModsFolderFailure(cmd *cobra.Command, deps listDeps, err error) error {
	var readErr *modsFolderReadError
	if !errors.As(err, &readErr) {
		return handleListFailure(cmd, deps, err)
	}
	if outputErr := writeListModsFolderFailure(cmd, deps, readErr.path, readErr.err); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func handleListModsFolderPartialFailure(cmd *cobra.Command, deps listDeps, err error) error {
	var readErr *modsFolderReadError
	if !errors.As(err, &readErr) {
		return handleListFailure(cmd, deps, err)
	}
	if outputErr := writeListModsFolderPartialFailure(cmd, deps, readErr.path, readErr.err); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func writeListModsFolderFailure(cmd *cobra.Command, deps listDeps, path string, readErr error) error {
	colorMode := colorModeForWriter(cmd)
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.list.error.mods_folder", &i18n.Tvars{
		Data: &i18n.TData{
			"modsFolder": path,
			"reason":     readErr.Error(),
		},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.list.error.mods_folder_hint", nil)
	hintSecondary := i18n.T("cmd.list.error.mods_folder_hint_secondary", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
		hintSecondary = view.CtaStyle.Render(hintSecondary)
	}

	combinedHint := hint + "\n" + hintSecondary
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, combinedHint})
}

func writeListModsFolderPartialFailure(cmd *cobra.Command, deps listDeps, path string, readErr error) error {
	colorMode := colorModeForWriter(cmd)
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.list.error.mods_folder_partial", &i18n.Tvars{
		Data: &i18n.TData{
			"path":   path,
			"reason": readErr.Error(),
		},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	notice := i18n.T("cmd.list.error.mods_folder_partial_notice", nil)
	hint := i18n.T("cmd.list.error.mods_folder_partial_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	combinedHeadline := headline + "\n" + notice
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{combinedHeadline, hint})
}

func writeConfigMissingOutput(cmd *cobra.Command, deps listDeps, meta config.Metadata) error {
	colorMode := colorModeForWriter(cmd)
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.list.error.config_missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.list.error.config_missing_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

func colorModeForWriter(cmd *cobra.Command) view.ColorMode {
	if cmd == nil {
		return view.ColorDisabled
	}
	if !view.SupportsColor(cmd.OutOrStdout()) {
		return view.ColorDisabled
	}
	return view.ColorEnabled
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
