package test

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

type testConfigState struct {
	cfg            models.ModsJSON
	shouldContinue bool
}

type targetVersionResolution struct {
	targetVersion  string
	shouldContinue bool
	exitCode       int
}

func runTest(ctx context.Context, cmd *cobra.Command, opts testOptions, deps testDeps) (Result, error) {
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	meta := config.NewMetadata(opts.ConfigPath)
	configState, err := ensureTestConfig(ctx, cmd, opts, deps, meta)
	if err != nil {
		return Result{ExitCode: 1, Interactive: mode.IsInteractive()}, err
	}
	if !configState.shouldContinue {
		return Result{ExitCode: 0, Interactive: mode.IsInteractive()}, nil
	}

	resolution, err := resolveTargetVersion(ctx, cmd, opts, deps, configState.cfg, mode)
	if err != nil {
		return resolutionResult(resolution, mode), err
	}
	if !resolution.shouldContinue {
		return resolutionResult(resolution, mode), nil
	}

	targetVersion := resolution.targetVersion
	if len(configState.cfg.Mods) == 0 {
		return handleEmptyModList(cmd, deps, opts, targetVersion, mode)
	}

	execInput := buildTestExecutionInput(configState.cfg, targetVersion, deps)
	outcome, err := runTestByMode(ctx, cmd, opts, mode, execInput)
	if err != nil {
		return Result{TargetVersion: targetVersion, ExitCode: 1, Interactive: mode.IsInteractive()}, err
	}

	return buildTestResult(outcome, targetVersion, mode)
}

func RunWithDeps(ctx context.Context, cmd *cobra.Command, opts Options, deps Deps) (Result, error) {
	return runTest(ctx, cmd, opts, deps)
}

func shouldRunInteractiveTest(opts testOptions, mode interaction.ExecutionMode) bool {
	if opts.Unattended || opts.Quiet {
		return false
	}
	return mode.IsInteractive()
}

func runInteractiveTest(ctx context.Context, cmd *cobra.Command, input testExecutionInput) (testExecutionOutcome, error) {
	model := newTestModel(testModelInput{
		ctx:               ctx,
		targetVersion:     input.targetVersion,
		colorMode:         colorModeForOutput(cmd.OutOrStdout()),
		items:             input.items,
		indexByKey:        input.indexByKey,
		showCompatibility: true,
		execRunner: func(ctx context.Context, sender testExecSender) testExecutionOutcome {
			return runTestExecution(ctx, input, sender)
		},
	})

	if runTestProgram == nil {
		return testExecutionOutcome{}, errors.New("missing bubble tea runner")
	}

	options := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
	options = append(options, tea.WithAltScreen())
	result, err := runTestProgram(model, options...)
	if err != nil {
		return testExecutionOutcome{}, err
	}

	typed, ok := result.(*testModel)
	if !ok {
		return testExecutionOutcome{}, errors.New("unexpected test model")
	}

	if outputErr := writeInteractiveTestTranscript(cmd, input, typed); outputErr != nil {
		return typed.outcome, outputErr
	}

	if typed.outcome.err != nil {
		return typed.outcome, typed.outcome.err
	}
	return typed.outcome, nil
}

func buildTestExecutionInput(cfg models.ModsJSON, targetVersion string, deps testDeps) testExecutionInput {
	items, indexByKey := buildTestItems(cfg)
	return testExecutionInput{
		cfg:           cfg,
		targetVersion: targetVersion,
		items:         items,
		indexByKey:    indexByKey,
		deps:          deps,
	}
}

func runTestByMode(
	ctx context.Context,
	cmd *cobra.Command,
	opts testOptions,
	mode interaction.ExecutionMode,
	input testExecutionInput,
) (testExecutionOutcome, error) {
	switch {
	case shouldRunInteractiveTest(opts, mode):
		return runInteractiveTest(ctx, cmd, input)
	case opts.Quiet:
		return runQuietTest(ctx, cmd, input)
	default:
		return runNonInteractiveTest(ctx, cmd, input)
	}
}

func buildTestResult(outcome testExecutionOutcome, targetVersion string, mode interaction.ExecutionMode) (Result, error) {
	exitCode, resultErr := evaluateTestOutcome(outcome)
	return Result{
		TargetVersion:   targetVersion,
		ExitCode:        exitCode,
		UnsupportedMods: toUnsupportedMods(outcome.items),
		Interactive:     mode.IsInteractive(),
	}, resultErr
}

func resolutionResult(resolution targetVersionResolution, mode interaction.ExecutionMode) Result {
	return Result{
		TargetVersion: resolution.targetVersion,
		ExitCode:      resolution.exitCode,
		Interactive:   mode.IsInteractive(),
	}
}

func handleEmptyModList(
	cmd *cobra.Command,
	deps testDeps,
	opts testOptions,
	targetVersion string,
	mode interaction.ExecutionMode,
) (Result, error) {
	if !opts.Quiet {
		colorMode := colorModeForOutput(cmd.OutOrStdout())
		lines := []string{renderSuccessLine(colorMode, i18n.T("cmd.test.success", &i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}))}
		if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), lines); outputErr != nil {
			return Result{ExitCode: 0, TargetVersion: targetVersion, Interactive: mode.IsInteractive()}, outputErr
		}
	}
	return Result{ExitCode: 0, TargetVersion: targetVersion, Interactive: mode.IsInteractive()}, nil
}

func runQuietTest(ctx context.Context, cmd *cobra.Command, input testExecutionInput) (testExecutionOutcome, error) {
	outcome := runTestExecution(ctx, input, testExecSender{})
	if outcome.err != nil {
		return outcome, outcome.err
	}

	colorMode := colorModeForOutput(cmd.OutOrStdout())
	lines := buildQuietTestLines(outcome.items, colorMode)
	if len(lines) == 0 {
		return outcome, nil
	}
	if outputErr := runOutputLines(cmd, input.deps, cmd.OutOrStdout(), []string{strings.Join(lines, "\n")}); outputErr != nil {
		return outcome, outputErr
	}
	return outcome, nil
}

func runNonInteractiveTest(
	ctx context.Context,
	cmd *cobra.Command,
	input testExecutionInput,
) (testExecutionOutcome, error) {
	outcome := runTestExecution(ctx, input, testExecSender{})
	if outcome.err != nil {
		return outcome, outcome.err
	}

	colorMode := colorModeForOutput(cmd.OutOrStdout())
	sections := buildTestSections(testViewInput{
		targetVersion: input.targetVersion,
		items:         outcome.items,
		colorMode:     colorMode,
	})

	if outputErr := runOutputLines(cmd, input.deps, cmd.OutOrStdout(), sections); outputErr != nil {
		return outcome, outputErr
	}

	return outcome, nil
}

func evaluateTestOutcome(outcome testExecutionOutcome) (int, error) {
	if outcome.err != nil {
		return 1, outcome.err
	}
	for _, item := range outcome.items {
		if item.Status == testItemStatusUnsupported || item.Status == testItemStatusInconclusive {
			return 1, clierrors.MarkHandled(errUnsupportedMods)
		}
	}
	return 0, nil
}

func testOutcomeFromModel(result tea.Model) error {
	switch typed := result.(type) {
	case *testModel:
		return typed.outcome.err
	default:
		return errors.New("unexpected test model")
	}
}

func writeInteractiveTestTranscript(cmd *cobra.Command, input testExecutionInput, model *testModel) error {
	if model == nil {
		return nil
	}

	sections := buildTestSections(testViewInput{
		targetVersion: model.targetVersion,
		items:         model.items,
		colorMode:     model.colorMode,
	})
	return runOutputLines(cmd, input.deps, cmd.OutOrStdout(), sections)
}

func ensureTestConfig(
	ctx context.Context,
	cmd *cobra.Command,
	opts testOptions,
	deps testDeps,
	meta config.Metadata,
) (testConfigState, error) {
	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err == nil {
		return testConfigState{cfg: cfg, shouldContinue: true}, nil
	}

	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return testConfigState{}, err
	}

	promptErr := configMissingPromptError(opts, cmd, meta)
	if promptErr != nil {
		if outputErr := writeConfigMissingOutput(cmd, deps, meta); outputErr != nil {
			return testConfigState{}, outputErr
		}
		return testConfigState{}, clierrors.MarkHandled(promptErr)
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, meta)
	if err != nil {
		return testConfigState{}, err
	}
	if canceled || !confirmed {
		return testConfigState{shouldContinue: false}, nil
	}

	if deps.runInit == nil {
		return testConfigState{}, errors.New("missing init runner")
	}
	if runErr := deps.runInit(ctx, cmd, initRequest{configPath: meta.ConfigPath}); runErr != nil {
		if errors.Is(runErr, initCmd.ErrInitCanceled) {
			return testConfigState{shouldContinue: false}, nil
		}
		return testConfigState{}, runErr
	}

	cfg, err = config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return testConfigState{}, err
	}
	return testConfigState{cfg: cfg, shouldContinue: true}, nil
}

func configMissingPromptError(opts testOptions, cmd *cobra.Command, meta config.Metadata) error {
	return interaction.CheckConfigInitGate(meta, interaction.ConfigInitGate{
		Unattended: opts.Unattended,
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

func writeConfigMissingOutput(cmd *cobra.Command, deps testDeps, meta config.Metadata) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	hint := i18n.T("cmd.config.error.missing_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	lines := []string{
		renderFinalErrorLine(colorMode, i18n.T("cmd.config.error.missing", &i18n.Tvars{
			Data: &i18n.TData{"configPath": meta.ConfigPath},
		})),
		hint,
	}
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), lines)
}

func resolveTargetVersion(
	ctx context.Context,
	cmd *cobra.Command,
	opts testOptions,
	deps testDeps,
	cfg models.ModsJSON,
	mode interaction.ExecutionMode,
) (targetVersionResolution, error) {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	targetVersion := opts.GameVersion

	resolvedVersion, resolution, err := resolveLatestVersion(ctx, cmd, deps, mode, colorMode, targetVersion)
	if err != nil || resolution != nil {
		return resolutionOrError(resolution, err)
	}
	targetVersion = resolvedVersion

	resolvedTarget, resolution, err := validateTargetVersion(ctx, cmd, deps, mode, colorMode, targetVersion)
	if err != nil || resolution != nil {
		return resolutionOrError(resolution, err)
	}
	targetVersion = resolvedTarget

	sameVersionResolution, handled, err := handleSameVersion(cmd, deps, opts, colorMode, targetVersion, cfg.GameVersion)
	if handled || err != nil {
		return sameVersionResolution, err
	}

	return targetVersionResolution{targetVersion: targetVersion, shouldContinue: true, exitCode: 0}, nil
}

func resolveLatestVersion(
	ctx context.Context,
	cmd *cobra.Command,
	deps testDeps,
	mode interaction.ExecutionMode,
	colorMode view.ColorMode,
	targetVersion string,
) (string, *targetVersionResolution, error) {
	if !strings.EqualFold(targetVersion, "latest") {
		return targetVersion, nil, nil
	}

	latest, err := deps.latestVersion(ctx, deps.minecraftClient)
	if err == nil {
		return latest, nil, nil
	}

	if mode.IsInteractive() {
		headline := renderFinalErrorLine(colorMode, i18n.T("cmd.minecraft.version.latest_unavailable", nil))
		version, canceled, promptErr := runTestVersionPrompt(ctx, cmd, deps, testVersionPromptInput{
			headline: headline,
			question: i18n.T("cmd.test.prompt.version.latest_question", nil),
		})
		if promptErr != nil {
			return "", &targetVersionResolution{exitCode: 1}, promptErr
		}
		if canceled {
			return "", &targetVersionResolution{shouldContinue: false, exitCode: 0}, nil
		}
		return version, nil, nil
	}

	resolution, outputErr := handleLatestUnavailableNonInteractive(cmd, deps, colorMode)
	return "", &resolution, outputErr
}

func validateTargetVersion(
	ctx context.Context,
	cmd *cobra.Command,
	deps testDeps,
	mode interaction.ExecutionMode,
	colorMode view.ColorMode,
	targetVersion string,
) (string, *targetVersionResolution, error) {
	for {
		valid, validationErr := deps.isValidVersion(ctx, targetVersion, deps.minecraftClient)
		if validationErr != nil {
			resolution, err := handleVersionValidationUnavailable(cmd, deps, colorMode)
			return "", &resolution, err
		}
		if valid {
			return targetVersion, nil, nil
		}

		resolvedInvalid, resolution, resolveErr := resolveInvalidVersion(ctx, cmd, deps, mode, colorMode, targetVersion)
		if resolveErr != nil || resolution != nil {
			return "", resolution, resolveErr
		}
		targetVersion = resolvedInvalid
	}
}

func handleLatestUnavailableNonInteractive(
	cmd *cobra.Command,
	deps testDeps,
	colorMode view.ColorMode,
) (targetVersionResolution, error) {
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.minecraft.version.latest_unavailable", nil))
	hint := i18n.T("cmd.test.error.latest_unavailable_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint}); outputErr != nil {
		return targetVersionResolution{exitCode: 1}, outputErr
	}
	return targetVersionResolution{exitCode: 1}, clierrors.MarkHandled(errLatestVersionRequired)
}

func handleVersionValidationUnavailable(
	cmd *cobra.Command,
	deps testDeps,
	colorMode view.ColorMode,
) (targetVersionResolution, error) {
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.minecraft.version.unavailable", nil))
	if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline}); outputErr != nil {
		return targetVersionResolution{exitCode: 1}, outputErr
	}
	return targetVersionResolution{exitCode: 1}, clierrors.MarkHandled(errVersionValidationUnavailable)
}

func resolveInvalidVersion(
	ctx context.Context,
	cmd *cobra.Command,
	deps testDeps,
	mode interaction.ExecutionMode,
	colorMode view.ColorMode,
	targetVersion string,
) (string, *targetVersionResolution, error) {
	if mode.IsInteractive() {
		headline := renderFinalErrorLine(colorMode, i18n.T("cmd.test.error.invalid_version", &i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}))
		version, canceled, promptErr := runTestVersionPrompt(ctx, cmd, deps, testVersionPromptInput{
			headline:     headline,
			question:     i18n.T("cmd.test.prompt.version.invalid_question", nil),
			initialValue: targetVersion,
			initialError: i18n.T("cmd.minecraft.version.invalid", nil),
		})
		if promptErr != nil {
			return "", &targetVersionResolution{exitCode: 1}, promptErr
		}
		if canceled {
			return "", &targetVersionResolution{shouldContinue: false, exitCode: 0}, nil
		}
		return version, nil, nil
	}

	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.test.error.invalid_version", &i18n.Tvars{
		Data: &i18n.TData{"version": targetVersion},
	}))
	hint := i18n.T("cmd.test.error.invalid_version_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint}); outputErr != nil {
		return "", &targetVersionResolution{exitCode: 1}, outputErr
	}
	return "", &targetVersionResolution{exitCode: 1}, clierrors.MarkHandled(errInvalidVersion)
}

func handleSameVersion(
	cmd *cobra.Command,
	deps testDeps,
	opts testOptions,
	colorMode view.ColorMode,
	targetVersion string,
	configVersion string,
) (targetVersionResolution, bool, error) {
	if targetVersion != configVersion {
		return targetVersionResolution{}, false, nil
	}

	if !opts.Quiet {
		line := renderSuccessLine(colorMode, i18n.T("cmd.test.same_version", &i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}))
		if outputErr := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{line}); outputErr != nil {
			return targetVersionResolution{exitCode: 0}, true, outputErr
		}
	}
	return targetVersionResolution{targetVersion: targetVersion, shouldContinue: false, exitCode: 0}, true, nil
}

func resolutionOrError(resolution *targetVersionResolution, err error) (targetVersionResolution, error) {
	if resolution != nil {
		return *resolution, err
	}
	return targetVersionResolution{exitCode: 1}, err
}

func toUnsupportedMods(items []testItem) []models.Mod {
	if len(items) == 0 {
		return nil
	}

	unsupported := make([]models.Mod, 0, len(items))
	for _, item := range items {
		if item.Status != testItemStatusUnsupported {
			continue
		}
		unsupported = append(unsupported, item.Mod)
	}
	return unsupported
}
