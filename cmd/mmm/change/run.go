package change

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/spf13/cobra"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

var errInvalidVersion = errors.New("invalid minecraft version")
var errLatestVersionRequired = errors.New("could not determine latest version")
var errVersionValidationUnavailable = errors.New("could not verify minecraft version")

type changeRunState struct {
	meta           config.Metadata
	cfg            models.ModsJSON
	lock           []models.ModInstall
	mode           interaction.ExecutionMode
	shouldContinue bool
}

func changeOptionsFromFlags(cmd *cobra.Command, args []string) (changeOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return changeOptions{}, err
	}
	unattended, err := cmd.Flags().GetBool("unattended")
	if err != nil {
		return changeOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return changeOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return changeOptions{}, err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return changeOptions{}, err
	}
	keepConfig, err := cmd.Flags().GetBool("keep-config")
	if err != nil {
		return changeOptions{}, err
	}
	pruneConfig, err := cmd.Flags().GetBool("prune-config")
	if err != nil {
		return changeOptions{}, err
	}
	disableSkipped, err := cmd.Flags().GetBool("disable-skipped")
	if err != nil {
		return changeOptions{}, err
	}

	forcePolicy, resolveErr := resolveForcePolicy(changeForcePolicyFlags{
		force:          force,
		keepConfig:     keepConfig,
		pruneConfig:    pruneConfig,
		disableSkipped: disableSkipped,
	})

	return changeOptions{
		ConfigPath:  configPath,
		GameVersion: resolveGameVersion(args),
		Unattended:  unattended,
		Quiet:       quiet,
		Debug:       debug,
		Force:       force,
		ForcePolicy: forcePolicy,
	}, resolveErr
}

func resolveGameVersion(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "latest"
}

func applyChangeCommandErrorPolicy(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
	}
	cmd.SilenceUsage = true
}

func runChange(ctx context.Context, cmd *cobra.Command, opts changeOptions, deps changeDeps) (changeResult, error) {
	runState, err := prepareChangeRunState(ctx, cmd, opts, deps)
	if err != nil {
		return changeResult{ExitCode: 1, Interactive: runState.mode.IsInteractive()}, err
	}
	if !runState.shouldContinue {
		return changeResult{ExitCode: 0, Interactive: runState.mode.IsInteractive()}, nil
	}

	targetVersion, noop, err := resolveTargetVersion(ctx, cmd, opts, deps, runState)
	if err != nil {
		return changeResult{ExitCode: 1, Interactive: runState.mode.IsInteractive()}, err
	}
	if noop {
		return changeResult{
			TargetVersion: targetVersion,
			ExitCode:      0,
			Interactive:   runState.mode.IsInteractive(),
		}, nil
	}

	return runChangeWithTarget(ctx, cmd, opts, deps, runState, targetVersion)
}

func prepareChangeRunState(ctx context.Context, cmd *cobra.Command, opts changeOptions, deps changeDeps) (changeRunState, error) {
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	runState := changeRunState{
		meta:           config.NewMetadata(opts.ConfigPath),
		mode:           mode,
		shouldContinue: true,
	}

	configState, err := ensureChangeConfig(ctx, cmd, opts, deps, runState)
	if err != nil {
		return runState, err
	}
	runState.cfg = configState.cfg
	runState.lock = configState.lock
	runState.shouldContinue = configState.shouldContinue
	return runState, nil
}

type changeConfigState struct {
	cfg            models.ModsJSON
	lock           []models.ModInstall
	shouldContinue bool
}

func ensureChangeConfig(ctx context.Context, cmd *cobra.Command, opts changeOptions, deps changeDeps, runState changeRunState) (changeConfigState, error) {
	configState, err := loadChangeConfig(ctx, deps, runState.meta)
	if err == nil {
		return configState, nil
	}

	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return changeConfigState{}, handleChangeFailure(cmd, deps, err)
	}

	return handleMissingChangeConfig(ctx, cmd, opts, deps, runState)
}

func loadChangeConfig(ctx context.Context, deps changeDeps, meta config.Metadata) (changeConfigState, error) {
	cfg, err := deps.readConfig(ctx, deps.fs, meta)
	if err != nil {
		return changeConfigState{}, err
	}
	lock, lockErr := deps.ensureLock(ctx, deps.fs, meta)
	if lockErr != nil {
		return changeConfigState{}, lockErr
	}
	return changeConfigState{
		cfg:            cfg,
		lock:           lock,
		shouldContinue: true,
	}, nil
}

func handleMissingChangeConfig(ctx context.Context, cmd *cobra.Command, opts changeOptions, deps changeDeps, runState changeRunState) (changeConfigState, error) {
	promptErr := configMissingPromptError(opts, cmd, runState.meta)
	if promptErr != nil {
		if outputErr := writeConfigMissingOutput(cmd, deps, runState.meta); outputErr != nil {
			return changeConfigState{}, outputErr
		}
		return changeConfigState{}, clierrors.MarkHandled(promptErr)
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, runState.meta)
	if err != nil {
		return changeConfigState{}, err
	}
	if canceled || !confirmed {
		return changeConfigState{shouldContinue: false}, nil
	}

	if initErr := runInteractiveInit(ctx, cmd, deps, opts, runState.meta); initErr != nil {
		if errors.Is(initErr, initCmd.ErrInitCanceled) {
			return changeConfigState{shouldContinue: false}, nil
		}
		return changeConfigState{}, initErr
	}

	configState, err := loadChangeConfig(ctx, deps, runState.meta)
	if err != nil {
		return changeConfigState{}, handleChangeFailure(cmd, deps, err)
	}
	return configState, nil
}

func configMissingPromptError(opts changeOptions, cmd *cobra.Command, meta config.Metadata) error {
	return interaction.CheckConfigInitGate(meta, interaction.ConfigInitGate{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
		UnattendedError: func(meta config.Metadata) error {
			return errors.New(i18n.T("cmd.change.error.config_missing", &i18n.Tvars{
				Data: &i18n.TData{"configPath": meta.ConfigPath},
			}))
		},
		NoTTYError: func(meta config.Metadata) error {
			return errors.New(i18n.T("cmd.change.error.config_missing", &i18n.Tvars{
				Data: &i18n.TData{"configPath": meta.ConfigPath},
			}))
		},
	})
}

func resolveTargetVersion(ctx context.Context, cmd *cobra.Command, opts changeOptions, deps changeDeps, runState changeRunState) (string, bool, error) {
	targetVersion := opts.GameVersion
	if strings.EqualFold(targetVersion, "latest") {
		latest, err := deps.latestVersion(ctx, deps.minecraftClient)
		if err != nil {
			return "", false, outputAndHandleChangeError(cmd, deps, i18n.T("cmd.change.error.latest_unavailable", nil), errLatestVersionRequired)
		}
		targetVersion = latest
	}

	if targetVersion == runState.cfg.GameVersion {
		if opts.Quiet {
			return targetVersion, true, nil
		}
		colorMode := colorModeForOutput(cmd.OutOrStdout())
		line := view.RenderModItemLine(view.ModItemLine{
			Label:  i18n.T("cmd.change.noop", &i18n.Tvars{Data: &i18n.TData{"version": targetVersion}}),
			Status: view.ModItemStatusSuccess,
		}, colorMode)
		if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{line}); outputErr != nil {
			return "", false, outputErr
		}
		return targetVersion, true, nil
	}

	valid, validationErr := deps.isValidVersion(ctx, targetVersion, deps.minecraftClient)
	if validationErr != nil {
		return "", false, outputAndHandleChangeError(cmd, deps, i18n.T("cmd.change.error.version_unavailable", nil), errVersionValidationUnavailable)
	}
	if !valid {
		message := i18n.T("cmd.change.error.invalid_version", &i18n.Tvars{Data: &i18n.TData{"version": targetVersion}})
		return "", false, outputAndHandleChangeError(cmd, deps, message, errInvalidVersion)
	}

	return targetVersion, false, nil
}

func runChangeWithTarget(ctx context.Context, cmd *cobra.Command, opts changeOptions, deps changeDeps, runState changeRunState, targetVersion string) (changeResult, error) {
	items, indexByKey := buildChangeItems(runState.cfg, changeItemOrderForMode(runState.mode))
	execInput := changeExecutionInput{
		meta:          runState.meta,
		cfg:           runState.cfg,
		lock:          runState.lock,
		targetVersion: targetVersion,
		force:         opts.Force,
		forcePolicy:   opts.ForcePolicy,
		deps:          deps,
		items:         items,
		indexByKey:    indexByKey,
	}

	var outcome changeOutcome
	var err error
	switch {
	case shouldRunInteractiveChange(opts, runState.mode):
		outcome, err = runInteractiveChange(ctx, cmd, execInput)
	case opts.Quiet:
		outcome, err = runQuietChange(ctx, cmd, deps, execInput)
	default:
		outcome, err = runNonInteractiveChange(ctx, cmd, deps, execInput, runState.mode)
	}
	if err != nil {
		return changeResultFromOutcome(targetVersion, items, runState.mode, outcome, 1), err
	}

	return changeResultFromOutcome(targetVersion, items, runState.mode, outcome, 0), nil
}

func changeItemOrderForMode(mode interaction.ExecutionMode) changeItemOrder {
	return changeItemOrderAlphabetical
}

func shouldRunInteractiveChange(opts changeOptions, mode interaction.ExecutionMode) bool {
	if opts.Unattended || opts.Quiet {
		return false
	}
	return mode.IsInteractive()
}

func changeResultFromOutcome(targetVersion string, items []changeItem, mode interaction.ExecutionMode, outcome changeOutcome, exitCode int) changeResult {
	return changeResult{
		TargetVersion:  targetVersion,
		TotalMods:      len(items),
		SkippedMods:    countSkipped(outcome.Items),
		DownloadedMods: countDownloaded(outcome.Items),
		ExitCode:       exitCode,
		Interactive:    mode.IsInteractive(),
	}
}

func outputAndHandleChangeError(cmd *cobra.Command, deps changeDeps, message string, err error) error {
	if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{message}); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func runInteractiveChange(ctx context.Context, cmd *cobra.Command, input changeExecutionInput) (changeOutcome, error) {
	policySelections := make(chan changeForcePolicy, 1)
	input.policySelection = policySelections
	model := newChangeModel(changeModelInput{
		ctx:             ctx,
		target:          input.targetVersion,
		colorMode:       colorModeForOutput(cmd.OutOrStdout()),
		items:           input.items,
		indexByKey:      input.indexByKey,
		forcePolicy:     input.forcePolicy,
		policySelection: policySelections,
		execRunner: func(ctx context.Context, sender httpclient.Sender) changeOutcome {
			return runChangeExecution(ctx, sender, input)
		},
	})

	if runChangeProgram == nil {
		return changeOutcome{}, errors.New("missing bubble tea runner")
	}

	options := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
	options = append(options, tea.WithMouseCellMotion(), tea.WithAltScreen())
	result, err := runChangeProgram(model, options...)
	if err != nil {
		return changeOutcome{}, err
	}

	typed, ok := result.(*changeModel)
	if !ok {
		return changeOutcome{}, errors.New("unexpected change model")
	}

	if outputErr := writeInteractiveTranscriptIfNeeded(cmd, input, typed); outputErr != nil {
		return typed.outcome, outputErr
	}

	if typed.outcome.Err != nil {
		return typed.outcome, clierrors.MarkHandled(typed.outcome.Err)
	}
	return typed.outcome, nil
}

func runNonInteractiveChange(ctx context.Context, cmd *cobra.Command, deps changeDeps, input changeExecutionInput, mode interaction.ExecutionMode) (changeOutcome, error) {
	outcome := runChangeExecution(ctx, nil, input)
	if outputErr := writeChangeOutcome(cmd, deps, outcome, input.targetVersion, mode); outputErr != nil {
		return outcome, outputErr
	}
	if outcome.Err != nil {
		return outcome, clierrors.MarkHandled(outcome.Err)
	}
	return outcome, nil
}

func runQuietChange(ctx context.Context, cmd *cobra.Command, deps changeDeps, input changeExecutionInput) (changeOutcome, error) {
	outcome := runChangeExecution(ctx, nil, input)
	if outputErr := writeQuietChangeOutcome(cmd, deps, outcome, input.targetVersion); outputErr != nil {
		return outcome, outputErr
	}
	if outcome.Err != nil {
		return outcome, clierrors.MarkHandled(outcome.Err)
	}
	return outcome, nil
}

func handleChangeFailure(cmd *cobra.Command, deps changeDeps, err error) error {
	if outputErr := writeChangeFailureOutput(cmd, deps, err); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func writeChangeFailureOutput(cmd *cobra.Command, deps changeDeps, err error) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.change.error.failed", &i18n.Tvars{
		Data: &i18n.TData{"reason": err.Error()},
	}))
	hint := i18n.T("cmd.change.error.failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func writeConfigMissingOutput(cmd *cobra.Command, deps changeDeps, meta config.Metadata) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.change.error.config_missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))

	hint := i18n.T("cmd.change.error.config_missing_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func writeInteractiveTranscriptIfNeeded(cmd *cobra.Command, input changeExecutionInput, model *changeModel) error {
	if model == nil {
		return nil
	}

	sections := buildChangeSections(changeViewInput{
		stage:            model.stage,
		target:           model.target,
		items:            model.items,
		colorMode:        model.colorMode,
		spinner:          &model.spinner,
		forcePolicy:      model.forcePolicy,
		waitingForPolicy: model.policyPrompt != nil,
	})
	if model.policyAnswer != "" {
		sections = append(sections, model.policyAnswer)
	}
	content := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	if strings.TrimSpace(content) == "" {
		return nil
	}

	return runOutputLines(cmd, input.deps, cmd.OutOrStdout(), sections)
}

func countSkipped(items []changeItem) int {
	count := 0
	for _, item := range items {
		if item.Skipped {
			count++
		}
	}
	return count
}

func countDownloaded(items []changeItem) int {
	count := 0
	for _, item := range items {
		if item.DownloadStatus == changeDownloadSucceeded {
			count++
		}
	}
	return count
}
