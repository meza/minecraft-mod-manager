package prune

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfiles"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type pruneOptions struct {
	ConfigPath string
	Unattended bool
	Quiet      bool
	Debug      bool
	Force      bool
	LockSync   locksync.PolicyFlags
}

type initRequest struct {
	configPath string
}

type initRunner func(context.Context, *cobra.Command, initRequest) error

type pruneDeps struct {
	fs        afero.Fs
	logger    *logger.Logger
	output    *output.Output
	telemetry func(telemetry.CommandTelemetry)
	runTea    func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
	runInit   initRunner
}

var runInteractiveInit = initCmd.RunInteractiveInit

var errPromptDisabled = errors.New("prune aborted: prompt disabled")

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: i18n.T("cmd.prune.short", nil),
		RunE:  runPruneCommand,
	}

	cmd.Flags().BoolP("force", "f", false, i18n.T("cmd.prune.flag.force", nil))

	return cmd
}

func runPruneCommand(cmd *cobra.Command, _ []string) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.prune")

	options, err := pruneOptionsFromFlags(cmd)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	deps := defaultPruneDeps(cmd, options)
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: options.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	deletedCount, runErr := runPrune(ctx, cmd, options, deps)
	span.SetAttributes(attribute.Bool("success", runErr == nil))
	span.End()

	recordPruneTelemetry(deps.telemetry, options, deletedCount, mode, runErr)
	applyPruneCommandErrorPolicy(cmd, runErr)
	return runErr
}

func applyPruneCommandErrorPolicy(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
	}
	cmd.SilenceUsage = true
}

func pruneOptionsFromFlags(cmd *cobra.Command) (pruneOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return pruneOptions{}, err
	}
	unattended, err := cmd.Flags().GetBool("unattended")
	if err != nil {
		return pruneOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return pruneOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return pruneOptions{}, err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return pruneOptions{}, err
	}
	lockSync, err := locksync.PolicyFlagsFromFlags(cmd.Flags())
	if err != nil {
		return pruneOptions{}, err
	}

	return pruneOptions{
		ConfigPath: configPath,
		Unattended: unattended,
		Quiet:      quiet,
		Debug:      debug,
		Force:      force,
		LockSync:   lockSync,
	}, nil
}

func defaultPruneDeps(cmd *cobra.Command, options pruneOptions) pruneDeps {
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: options.Quiet,
		Debug: options.Debug,
	})
	return pruneDeps{
		fs:        common.FS,
		logger:    common.Logger,
		output:    common.Output,
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
				Quiet:      options.Quiet,
				Debug:      options.Debug,
			})
		},
	}
}

func defaultRunTea(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	return program.Run()
}

var runTeaProgram = defaultRunTea

func recordPruneTelemetry(record func(telemetry.CommandTelemetry), options pruneOptions, deletedCount int, mode interaction.ExecutionMode, err error) {
	payload := telemetry.CommandTelemetry{
		Command:       "prune",
		Success:       err == nil,
		Error:         err,
		ExitCode:      0,
		Interactive:   mode.IsInteractive(),
		ExecutionMode: mode.String(),
		Arguments: map[string]interface{}{
			"force": options.Force,
		},
	}
	if err != nil {
		payload.ExitCode = 1
	} else {
		payload.Extra = map[string]interface{}{
			"deletedCount": deletedCount,
		}
	}
	record(payload)
}

type pruneConfigState struct {
	Config         models.ModsJSON
	Lock           []models.ModInstall
	ShouldContinue bool
}

func runPrune(ctx context.Context, cmd *cobra.Command, options pruneOptions, deps pruneDeps) (int, error) {
	meta := config.NewMetadata(options.ConfigPath)
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: options.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	configState, err := ensurePruneConfig(ctx, cmd, options, deps, meta)
	if err != nil {
		return 0, err
	}
	if !configState.ShouldContinue {
		return 0, nil
	}

	unmanagedFiles, err := modfiles.ListUnmanagedFiles(deps.fs, meta, configState.Config, configState.Lock)
	if err != nil {
		return 0, handlePruneFailure(cmd, deps, err)
	}
	if len(unmanagedFiles) == 0 {
		if outputErr := writeNoUnmanagedOutput(cmd, deps); outputErr != nil {
			return 0, outputErr
		}
		return 0, nil
	}

	sortedUnmanaged := sortUnmanagedFiles(unmanagedFiles)
	colorMode := colorModeForWriter(cmd)

	if shouldUseConfirmDeleteFlow(options, mode) {
		deletedCount, err := runInteractiveConfirmDelete(cmd, deps, colorMode, sortedUnmanaged)
		return deletedCount, err
	}

	shouldDelete, promptErr := confirmPrune(cmd, deps, options, mode, colorMode, sortedUnmanaged)
	if promptErr != nil {
		return 0, promptErr
	}
	if !shouldDelete {
		return 0, nil
	}

	deletedCount, deleteErr := executePruneDeletion(cmd, deps, options, mode, colorMode, sortedUnmanaged)
	return deletedCount, deleteErr
}

func shouldUseConfirmDeleteFlow(options pruneOptions, mode interaction.ExecutionMode) bool {
	return mode == interaction.ExecutionModeInteractive && !options.Force && !options.Quiet
}

func runInteractiveConfirmDelete(
	cmd *cobra.Command,
	deps pruneDeps,
	colorMode view.ColorMode,
	unmanaged []string,
) (int, error) {
	outcome, err := runConfirmDeleteFlow(cmd, deps, colorMode, unmanaged, func() ([]pruneFileResult, error) {
		return deleteUnmanagedFiles(deps, unmanaged)
	})
	if err != nil {
		return 0, err
	}
	if outcome.canceled || !outcome.confirmed {
		return 0, nil
	}

	deletedCount := countDeleted(outcome.results)
	if outcome.deleteErr != nil {
		return deletedCount, clierrors.MarkHandled(outcome.deleteErr)
	}
	return deletedCount, nil
}

func confirmPrune(
	cmd *cobra.Command,
	deps pruneDeps,
	options pruneOptions,
	mode interaction.ExecutionMode,
	colorMode view.ColorMode,
	unmanaged []string,
) (bool, error) {
	if options.Force {
		return true, nil
	}
	if options.Quiet {
		listView := renderUnmanagedList(colorMode, unmanaged)
		if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{listView}); outputErr != nil {
			return false, outputErr
		}
		return false, nil
	}
	return shouldDeleteUnmanaged(cmd, deps, mode, colorMode, unmanaged)
}

func executePruneDeletion(
	cmd *cobra.Command,
	deps pruneDeps,
	options pruneOptions,
	mode interaction.ExecutionMode,
	colorMode view.ColorMode,
	unmanaged []string,
) (int, error) {
	if options.Force && mode == interaction.ExecutionModeInteractive && !options.Quiet {
		results, deleteErr, runErr := runForceInteractiveDeletion(cmd, deps, colorMode, unmanaged)
		deletedCount := countDeleted(results)
		if runErr != nil {
			return deletedCount, runErr
		}
		if deleteErr != nil {
			return deletedCount, clierrors.MarkHandled(deleteErr)
		}
		return deletedCount, nil
	}

	results, deleteErr := deleteUnmanagedFiles(deps, unmanaged)
	deletedCount := countDeleted(results)
	if deleteErr != nil {
		if outputErr := writeDeleteFailedOutput(cmd, deps, colorMode, results); outputErr != nil {
			return deletedCount, outputErr
		}
		return deletedCount, clierrors.MarkHandled(deleteErr)
	}

	if options.Quiet {
		return deletedCount, nil
	}

	if outputErr := writeDeleteSuccessOutput(cmd, deps, colorMode, results); outputErr != nil {
		return deletedCount, outputErr
	}
	return deletedCount, nil
}

func runForceInteractiveDeletion(
	cmd *cobra.Command,
	deps pruneDeps,
	colorMode view.ColorMode,
	unmanaged []string,
) (results []pruneFileResult, deleteErr error, runErr error) {
	runTea := deps.runTea
	if runTea == nil {
		runTea = runTeaProgram
	}

	model := newPruneForceModel(colorMode, unmanaged, func() ([]pruneFileResult, error) {
		return deleteUnmanagedFiles(deps, unmanaged)
	})
	result, err := runTea(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return nil, nil, err
	}
	results, deleteErr, resultErr := pruneForceResultFromModel(result)
	return results, deleteErr, resultErr
}

func ensurePruneConfig(
	ctx context.Context,
	cmd *cobra.Command,
	options pruneOptions,
	deps pruneDeps,
	meta config.Metadata,
) (pruneConfigState, error) {
	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err == nil {
		return runPruneLockSync(ctx, cmd, options, deps, meta, cfg)
	}

	return handlePruneConfigReadError(ctx, cmd, options, deps, meta, err)
}

func runPruneLockSync(
	ctx context.Context,
	cmd *cobra.Command,
	options pruneOptions,
	deps pruneDeps,
	meta config.Metadata,
	cfg models.ModsJSON,
) (pruneConfigState, error) {
	lock, lockErr := readLockRequired(ctx, deps.fs, meta)
	if lockErr != nil {
		return pruneConfigState{}, handleLockReadError(cmd, deps, lockErr)
	}
	syncOutcome, syncErr := locksync.RunLockSyncGate(locksync.GateInput{
		Ctx:         ctx,
		Fs:          deps.fs,
		Meta:        meta,
		Config:      cfg,
		Lock:        lock,
		Mode:        resolvePruneMode(cmd, options),
		CommandName: cmd.Name(),
		ColorMode:   colorModeForWriter(cmd),
		In:          cmd.InOrStdin(),
		Out:         cmd.OutOrStdout(),
		RunTea:      deps.runTea,
		PolicyFlags: options.LockSync,
		Force:       options.Force,
	})
	if syncErr != nil {
		return pruneConfigState{}, handlePruneFailure(cmd, deps, syncErr)
	}
	if !syncOutcome.ShouldContinue {
		return pruneConfigState{ShouldContinue: false}, nil
	}
	return pruneConfigState{Config: syncOutcome.Config, Lock: syncOutcome.Lock, ShouldContinue: true}, nil
}

func resolvePruneMode(cmd *cobra.Command, options pruneOptions) interaction.ExecutionMode {
	return interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: options.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})
}

func handlePruneConfigReadError(
	ctx context.Context,
	cmd *cobra.Command,
	options pruneOptions,
	deps pruneDeps,
	meta config.Metadata,
	err error,
) (pruneConfigState, error) {
	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return pruneConfigState{}, handlePruneFailure(cmd, deps, err)
	}

	return handlePruneConfigMissing(ctx, cmd, options, deps, meta)
}

func handlePruneConfigMissing(
	ctx context.Context,
	cmd *cobra.Command,
	options pruneOptions,
	deps pruneDeps,
	meta config.Metadata,
) (pruneConfigState, error) {
	promptErr := configMissingPromptError(options, cmd, meta)
	if promptErr != nil {
		if outputErr := writeConfigMissingOutput(cmd, deps, meta); outputErr != nil {
			return pruneConfigState{}, outputErr
		}
		return pruneConfigState{}, clierrors.MarkHandled(promptErr)
	}

	confirmed, canceled, promptErr := runConfigInitPrompt(cmd, deps, meta)
	if promptErr != nil {
		return pruneConfigState{}, promptErr
	}
	if canceled || !confirmed {
		return pruneConfigState{ShouldContinue: false}, nil
	}

	if deps.runInit == nil {
		return pruneConfigState{}, errors.New("missing init runner")
	}
	if runErr := deps.runInit(ctx, cmd, initRequest{configPath: meta.ConfigPath}); runErr != nil {
		if errors.Is(runErr, initCmd.ErrInitCanceled) {
			return pruneConfigState{ShouldContinue: false}, nil
		}
		return pruneConfigState{}, runErr
	}

	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return pruneConfigState{}, handlePruneFailure(cmd, deps, err)
	}
	return runPruneLockSync(ctx, cmd, options, deps, meta, cfg)
}

func configMissingPromptError(options pruneOptions, cmd *cobra.Command, meta config.Metadata) error {
	return interaction.CheckConfigInitGate(meta, interaction.ConfigInitGate{
		Unattended: options.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
		UnattendedError: func(meta config.Metadata) error {
			return errors.New(i18n.T("cmd.config.error.missing", &i18n.Tvars{
				Data: &i18n.TData{"configPath": meta.ConfigPath},
			}))
		},
		NoTTYError: func(meta config.Metadata) error {
			return errors.New(i18n.T("cmd.config.error.missing", &i18n.Tvars{
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

func readLockRequired(ctx context.Context, fs afero.Fs, meta config.Metadata) ([]models.ModInstall, error) {
	lock, err := config.ReadLock(ctx, fs, meta)
	if err == nil {
		return lock, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, &lockMissingError{
			message: i18n.T("cmd.prune.error.lock_missing", &i18n.Tvars{
				Data: &i18n.TData{"lockPath": meta.LockPath()},
			}),
		}
	}
	return nil, err
}

func handleLockReadError(cmd *cobra.Command, deps pruneDeps, err error) error {
	var lockMissing *lockMissingError
	if !errors.As(err, &lockMissing) {
		return handlePruneFailure(cmd, deps, err)
	}
	if outputErr := writeLockMissingOutput(cmd, deps, lockMissing); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

type pruneFileStatus int

const (
	pruneFileStatusUnmanaged pruneFileStatus = iota
	pruneFileStatusDeleting
	pruneFileStatusDeleted
	pruneFileStatusFailed
)

type pruneFileResult struct {
	Path   string
	Status pruneFileStatus
	Err    error
}

func shouldDeleteUnmanaged(
	cmd *cobra.Command,
	deps pruneDeps,
	mode interaction.ExecutionMode,
	colorMode view.ColorMode,
	unmanagedFiles []string,
) (bool, error) {
	if mode != interaction.ExecutionModeInteractive {
		listView := renderUnmanagedList(colorMode, unmanagedFiles)
		refusalView := renderUnattendedRefusalSummary(colorMode)
		combined := view.RenderViewSections([]string{listView, refusalView}, view.SectionSeparatorParagraph)
		if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{combined}); outputErr != nil {
			return false, outputErr
		}
		return false, clierrors.MarkHandled(errPromptDisabled)
	}

	confirmed, canceled, err := runDeletePrompt(cmd, deps, colorMode, unmanagedFiles)
	if err != nil {
		return false, err
	}
	if canceled || !confirmed {
		return false, nil
	}
	return true, nil
}

func deleteUnmanagedFiles(deps pruneDeps, unmanaged []string) ([]pruneFileResult, error) {
	results := make([]pruneFileResult, 0, len(unmanaged))
	var deleteErr error

	for _, filePath := range unmanaged {
		if err := removeFileForce(deps.fs, filePath); err != nil {
			if deleteErr == nil {
				deleteErr = err
			}
			results = append(results, pruneFileResult{Path: filePath, Status: pruneFileStatusFailed, Err: err})
			continue
		}
		results = append(results, pruneFileResult{Path: filePath, Status: pruneFileStatusDeleted})
	}

	if deleteErr != nil {
		return results, deleteErr
	}
	return results, nil
}

func removeFileForce(fs afero.Fs, path string) error {
	if err := fs.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func sortUnmanagedFiles(files []string) []string {
	sorted := append([]string{}, files...)
	sort.Slice(sorted, func(leftIndex int, rightIndex int) bool {
		left := strings.ToLower(filepath.Base(sorted[leftIndex]))
		right := strings.ToLower(filepath.Base(sorted[rightIndex]))
		if left != right {
			return left < right
		}
		return sorted[leftIndex] < sorted[rightIndex]
	})
	return sorted
}

func countDeleted(results []pruneFileResult) int {
	count := 0
	for _, result := range results {
		if result.Status == pruneFileStatusDeleted {
			count++
		}
	}
	return count
}

func handlePruneFailure(cmd *cobra.Command, deps pruneDeps, err error) error {
	if outputErr := writePruneFailureOutput(cmd, deps, err); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func writePruneFailureOutput(cmd *cobra.Command, deps pruneDeps, err error) error {
	colorMode := colorModeForWriter(cmd)
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.prune.error.failed", &i18n.Tvars{
		Data: &i18n.TData{"reason": err.Error()},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.prune.error.failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func writeConfigMissingOutput(cmd *cobra.Command, deps pruneDeps, meta config.Metadata) error {
	colorMode := colorModeForWriter(cmd)
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.config.error.missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.config.error.missing_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func writeLockMissingOutput(cmd *cobra.Command, deps pruneDeps, lockMissing *lockMissingError) error {
	colorMode := colorModeForWriter(cmd)
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), lockMissing.Error())
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.prune.error.lock_missing_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func writeNoUnmanagedOutput(cmd *cobra.Command, deps pruneDeps) error {
	colorMode := colorModeForWriter(cmd)
	hint := i18n.T("cmd.prune.summary.success_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{i18n.T("cmd.prune.no_unmanaged", nil), hint})
}

func writeDeletingOutput(cmd *cobra.Command, deps pruneDeps, colorMode view.ColorMode, unmanaged []string) error {
	listView := renderDeletingList(colorMode, unmanaged)
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{listView})
}

func writeDeleteFailedOutput(cmd *cobra.Command, deps pruneDeps, colorMode view.ColorMode, results []pruneFileResult) error {
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{renderDeleteFailedView(colorMode, results)})
}

func writeDeleteSuccessOutput(cmd *cobra.Command, deps pruneDeps, colorMode view.ColorMode, results []pruneFileResult) error {
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{renderDeleteSuccessView(colorMode, results)})
}

func renderUnmanagedList(colorMode view.ColorMode, unmanaged []string) string {
	header := i18n.T("cmd.prune.header.unmanaged", nil)
	lines := renderPruneFileResults(colorMode, toPruneResults(unmanaged, pruneFileStatusUnmanaged))
	return strings.Join(append([]string{header}, lines...), "\n")
}

func renderDeletingList(colorMode view.ColorMode, unmanaged []string) string {
	header := i18n.T("cmd.prune.header.deleting", nil)
	lines := renderPruneFileResults(colorMode, toPruneResults(unmanaged, pruneFileStatusDeleting))
	return strings.Join(append([]string{header}, lines...), "\n")
}

func renderDeleteFailedView(colorMode view.ColorMode, results []pruneFileResult) string {
	header := i18n.T("cmd.prune.header.deleting", nil)
	lines := renderPruneFileResults(colorMode, results)
	sections := []string{
		strings.Join(append([]string{header}, lines...), "\n"),
		renderDeleteFailureSummary(colorMode),
	}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderDeleteSuccessView(colorMode view.ColorMode, results []pruneFileResult) string {
	header := i18n.T("cmd.prune.header.deleted", nil)
	lines := renderPruneFileResults(colorMode, results)
	sections := []string{
		strings.Join(append([]string{header}, lines...), "\n"),
		renderDeleteSuccessSummary(colorMode),
	}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderUnattendedRefusalSummary(colorMode view.ColorMode) string {
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.prune.error.unattended_refuse", nil))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.prune.error.unattended_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return strings.Join([]string{headline, hint}, "\n")
}

func toPruneResults(files []string, status pruneFileStatus) []pruneFileResult {
	results := make([]pruneFileResult, 0, len(files))
	for _, filePath := range files {
		results = append(results, pruneFileResult{Path: filePath, Status: status})
	}
	return results
}

func renderPruneFileResults(colorMode view.ColorMode, results []pruneFileResult) []string {
	lines := make([]string, 0, len(results))
	for _, result := range results {
		lines = append(lines, renderPruneFileLine(colorMode, result))
	}
	return lines
}

func renderPruneFileLine(colorMode view.ColorMode, result pruneFileResult) string {
	status := view.ModItemStatusError
	switch result.Status {
	case pruneFileStatusDeleting:
		status = view.ModItemStatusPending
	case pruneFileStatusDeleted:
		status = view.ModItemStatusSuccess
	}

	suffix := ""
	if result.Status == pruneFileStatusFailed && result.Err != nil {
		suffix = i18n.T("cmd.file.delete_failed", &i18n.Tvars{
			Data: &i18n.TData{"reason": result.Err.Error()},
		})
	}

	return view.RenderModItemLine(view.ModItemLine{
		Label:  filepath.Base(result.Path),
		Suffix: suffix,
		Status: status,
	}, colorMode)
}

func renderDeleteSuccessSummary(colorMode view.ColorMode) string {
	lines := []string{
		fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), i18n.T("cmd.prune.summary.success", nil)),
		i18n.T("cmd.prune.summary.success_hint", nil),
	}
	if colorMode.Enabled() {
		lines[1] = view.CtaStyle.Render(lines[1])
	}
	return strings.Join(lines, "\n")
}

func renderDeleteFailureSummary(colorMode view.ColorMode) string {
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.prune.summary.incomplete", nil))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.prune.summary.incomplete_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return strings.Join([]string{headline, hint}, "\n")
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

func colorModeForWriter(cmd *cobra.Command) view.ColorMode {
	if cmd == nil {
		return view.ColorDisabled
	}
	if !view.SupportsColor(cmd.OutOrStdout()) || !view.SupportsControlSequences(cmd.OutOrStdout()) {
		return view.ColorDisabled
	}
	return view.ColorEnabled
}
