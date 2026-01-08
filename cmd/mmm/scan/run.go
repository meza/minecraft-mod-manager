package scan

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func runScanByMode(
	ctx context.Context,
	cmd *cobra.Command,
	input scanExecutionInput,
	mode interaction.ExecutionMode,
	options scanOptions,
) (scanExecutionOutcome, error) {
	if mode == interaction.ExecutionModeUnattended && !view.SupportsControlSequences(cmd.OutOrStdout()) {
		mode = interaction.ExecutionModeNonTTY
	}
	switch {
	case options.Quiet:
		return runQuietScan(ctx, cmd, input, options)
	case mode == interaction.ExecutionModeUnattended:
		return runInteractiveScan(ctx, cmd, input, options)
	case mode.IsInteractive():
		return runInteractiveScan(ctx, cmd, input, options)
	default:
		return runScanTranscript(ctx, cmd, input, options)
	}
}

func runInteractiveScan(ctx context.Context, cmd *cobra.Command, input scanExecutionInput, options scanOptions) (scanExecutionOutcome, error) {
	items, indexByKey := buildScanItems(input.candidates)
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	model := newScanModel(scanModelInput{
		ctx:        ctx,
		colorMode:  colorMode,
		items:      items,
		indexByKey: indexByKey,
		execRunner: func(ctx context.Context, sender scanExecSender) scanExecutionOutcome {
			return runScanExecution(ctx, input, sender)
		},
	})
	outcome, err := runInteractiveScanProgram(cmd, model)
	if err != nil {
		if isContextCancellation(err) {
			return scanExecutionOutcome{}, nil
		}
		return outcome, err
	}
	if isContextCancellation(outcome.err) {
		return scanExecutionOutcome{}, nil
	}

	return handleInteractiveScanOutcome(ctx, cmd, input, options, colorMode, outcome)
}

func runInteractiveScanProgram(cmd *cobra.Command, model *scanModel) (scanExecutionOutcome, error) {
	if runScanProgram == nil {
		return scanExecutionOutcome{}, errors.New("missing bubble tea runner")
	}
	optionsList := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
	optionsList = append(optionsList, tea.WithAltScreen())
	result, err := runScanProgram(model, optionsList...)
	if err != nil {
		return scanExecutionOutcome{}, err
	}
	outcome, err := scanOutcomeFromModel(result)
	if err != nil {
		return scanExecutionOutcome{}, err
	}
	if outcome.err != nil {
		if isContextCancellation(outcome.err) {
			return outcome, nil
		}
		return outcome, outcome.err
	}
	return outcome, nil
}

func handleInteractiveScanOutcome(
	ctx context.Context,
	cmd *cobra.Command,
	input scanExecutionInput,
	options scanOptions,
	colorMode view.ColorMode,
	outcome scanExecutionOutcome,
) (scanExecutionOutcome, error) {
	if shouldPromptForAdoption(options, outcome.matches) {
		return handleInteractiveScanPrompt(ctx, cmd, input, colorMode, outcome)
	}
	if options.Add {
		return handleInteractiveScanAdd(ctx, cmd, input, colorMode, outcome)
	}
	if outputErr := writeScanResultsOutput(cmd, input, outcome); outputErr != nil {
		return outcome, outputErr
	}
	return outcome, nil
}

func handleInteractiveScanPrompt(
	ctx context.Context,
	cmd *cobra.Command,
	input scanExecutionInput,
	colorMode view.ColorMode,
	outcome scanExecutionOutcome,
) (scanExecutionOutcome, error) {
	promptResult, promptErr := runScanAdoptionPrompt(cmd, input.deps, outcome)
	if promptErr != nil {
		return outcome, promptErr
	}
	if promptResult.canceled {
		return outcome, nil
	}
	if !promptResult.confirmed {
		line := renderScanAdoptionCancelledLine(colorMode)
		if outputErr := writeScanTranscriptSections(cmd, input, []string{line}); outputErr != nil {
			return outcome, outputErr
		}
		return outcome, nil
	}
	return handleInteractivePromptConfirm(ctx, cmd, input, colorMode, outcome, promptResult.prompt)
}

func handleInteractivePromptConfirm(
	ctx context.Context,
	cmd *cobra.Command,
	input scanExecutionInput,
	colorMode view.ColorMode,
	outcome scanExecutionOutcome,
	prompt scanConfirmPromptModel,
) (scanExecutionOutcome, error) {
	added, persistErr := persistScanMatches(ctx, cmd, input, outcome.matches)
	if persistErr != nil {
		return outcome, persistErr
	}
	outcome.added = added
	if isFullAdoption(outcome) {
		return writeFullAdoptionOutput(cmd, input, colorMode, outcome)
	}
	sections := buildScanPromptTranscriptSectionsWithAdded(outcome, colorMode, prompt)
	if outputErr := writeScanTranscriptSections(cmd, input, sections); outputErr != nil {
		return outcome, outputErr
	}
	return outcome, nil
}

func handleInteractiveScanAdd(
	ctx context.Context,
	cmd *cobra.Command,
	input scanExecutionInput,
	colorMode view.ColorMode,
	outcome scanExecutionOutcome,
) (scanExecutionOutcome, error) {
	added, persistErr := persistScanMatches(ctx, cmd, input, outcome.matches)
	if persistErr != nil {
		return outcome, persistErr
	}
	outcome.added = added
	if isFullAdoption(outcome) {
		return writeFullAdoptionOutput(cmd, input, colorMode, outcome)
	}
	sections := buildScanResultSectionsWithoutMatches(outcome, colorMode)
	sections = appendScanAddedSection(sections, outcome.added, colorMode)
	if outputErr := writeScanTranscriptSections(cmd, input, sections); outputErr != nil {
		return outcome, outputErr
	}
	return outcome, nil
}

func runScanTranscript(ctx context.Context, cmd *cobra.Command, input scanExecutionInput, options scanOptions) (scanExecutionOutcome, error) {
	items, indexByKey := buildScanItems(input.candidates)
	model := newScanTranscriptModel(
		ctx,
		colorModeForOutput(cmd.OutOrStdout()),
		items,
		indexByKey,
		cmd.OutOrStdout(),
		options.Quiet,
		options.Add,
		func(ctx context.Context, sender scanExecSender) scanExecutionOutcome {
			return runScanExecution(ctx, input, sender)
		},
	)

	if runScanTranscriptProgram == nil {
		return scanExecutionOutcome{}, errors.New("missing bubble tea runner")
	}

	result, err := runScanTranscriptProgram(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return scanExecutionOutcome{}, err
	}

	outcome, err := scanOutcomeFromModel(result)
	if err != nil {
		return scanExecutionOutcome{}, err
	}
	if outcome.err != nil {
		return outcome, outcome.err
	}
	if !options.Add {
		return outcome, nil
	}
	added, persistErr := persistScanMatches(ctx, cmd, input, outcome.matches)
	if persistErr != nil {
		return outcome, persistErr
	}
	outcome.added = added
	if outputErr := writeScanAddedSectionOutput(cmd, input, outcome.added); outputErr != nil {
		return outcome, outputErr
	}
	return outcome, nil
}

func runQuietScan(ctx context.Context, cmd *cobra.Command, input scanExecutionInput, options scanOptions) (scanExecutionOutcome, error) {
	outcome := runScanExecution(ctx, input, scanExecSender{})
	if outcome.err != nil {
		return outcome, outcome.err
	}
	if options.Add {
		added, persistErr := persistScanMatches(ctx, cmd, input, outcome.matches)
		if persistErr != nil {
			return outcome, persistErr
		}
		outcome.added = added
	}
	if outputErr := writeScanQuietOutput(cmd, input, outcome, options); outputErr != nil {
		return outcome, outputErr
	}
	return outcome, nil
}

func shouldPromptForAdoption(options scanOptions, matches []scanMatch) bool {
	if options.Add || options.Unattended {
		return false
	}
	return len(matches) > 0
}

func runScanAdoptionPrompt(cmd *cobra.Command, deps scanDeps, outcome scanExecutionOutcome) (scanAdoptionPromptModel, error) {
	sections := buildScanResultSections(scanResultsViewInput{
		matches:   outcome.matches,
		unknown:   outcome.unknown,
		unsure:    outcome.unsure,
		colorMode: colorModeForOutput(cmd.OutOrStdout()),
	})
	question := i18n.T("cmd.scan.prompt.add", nil)
	model := newScanAdoptionPromptModel(sections, question)

	runTea := deps.runTea
	if runTea == nil {
		runTea = runTeaProgram
	}
	if runTea == nil {
		return scanAdoptionPromptModel{}, errors.New("missing bubble tea runner")
	}
	options := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
	if view.SupportsControlSequences(cmd.OutOrStdout()) {
		options = append(options, tea.WithAltScreen())
	}
	result, err := runTea(model, options...)
	if err != nil {
		return scanAdoptionPromptModel{}, err
	}
	return scanAdoptionPromptResult(result)
}

func writeScanResultsOutput(cmd *cobra.Command, input scanExecutionInput, outcome scanExecutionOutcome) error {
	sections := buildScanResultSections(scanResultsViewInput{
		matches:   outcome.matches,
		unknown:   outcome.unknown,
		unsure:    outcome.unsure,
		colorMode: colorModeForOutput(cmd.OutOrStdout()),
	})
	return writeScanTranscriptSections(cmd, input, sections)
}

func writeScanTranscriptSections(cmd *cobra.Command, input scanExecutionInput, sections []string) error {
	if len(sections) == 0 {
		return nil
	}
	return runOutputLines(cmd, input.deps, cmd.OutOrStdout(), sections)
}

func writeScanAllManaged(cmd *cobra.Command, deps scanDeps) error {
	if cmd == nil {
		return nil
	}
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	line := fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), i18n.T("cmd.scan.all_managed", nil))
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{line})
}

func writeScanAddedSectionOutput(cmd *cobra.Command, input scanExecutionInput, added []scanMatch) error {
	if len(added) == 0 {
		return nil
	}
	section := renderScanAddedSection(scanAddedViewInput{added: added, colorMode: colorModeForOutput(cmd.OutOrStdout())})
	return writeScanTranscriptSections(cmd, input, []string{"", section})
}

func writeScanQuietOutput(cmd *cobra.Command, input scanExecutionInput, outcome scanExecutionOutcome, options scanOptions) error {
	if options.Add {
		return nil
	}
	viewText := renderScanQuietView(scanResultsViewInput{
		unknown:   outcome.unknown,
		unsure:    outcome.unsure,
		colorMode: colorModeForOutput(cmd.OutOrStdout()),
	})
	if strings.TrimSpace(viewText) == "" {
		return nil
	}
	return runOutputLines(cmd, input.deps, cmd.OutOrStdout(), []string{viewText})
}

func writeScanFailureOutput(cmd *cobra.Command, deps scanDeps, err error) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{renderScanFailureLine(colorMode, err)})
}

func writeInteractiveScanTranscript(cmd *cobra.Command, input scanExecutionInput, model *scanModel) error {
	if model == nil {
		return nil
	}
	sections := buildScanResultSectionsForTranscript(model.outcome, model.colorMode)
	return writeScanTranscriptSections(cmd, input, sections)
}

func isFullAdoption(outcome scanExecutionOutcome) bool {
	return len(outcome.matches) > 0 && len(outcome.unknown) == 0 && len(outcome.unsure) == 0
}

func writeFullAdoptionOutput(
	cmd *cobra.Command,
	input scanExecutionInput,
	colorMode view.ColorMode,
	outcome scanExecutionOutcome,
) (scanExecutionOutcome, error) {
	if len(outcome.added) == 0 {
		return outcome, nil
	}
	section := renderScanAddedSection(scanAddedViewInput{added: outcome.added, colorMode: colorMode})
	if outputErr := writeScanTranscriptSections(cmd, input, []string{section}); outputErr != nil {
		return outcome, outputErr
	}
	return outcome, nil
}

func writeConfigMissingOutput(cmd *cobra.Command, deps scanDeps, meta config.Metadata) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.scan.error.config_missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.scan.error.config_missing_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, hint})
}

func handleScanFailure(cmd *cobra.Command, deps scanDeps, err error) error {
	if clierrors.IsHandled(err) {
		return err
	}
	if outputErr := writeScanFailureOutput(cmd, deps, err); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func buildScanResultSectionsForTranscript(outcome scanExecutionOutcome, colorMode view.ColorMode) []string {
	return buildScanResultSections(scanResultsViewInput{
		matches:   outcome.matches,
		unknown:   outcome.unknown,
		unsure:    outcome.unsure,
		colorMode: colorMode,
	})
}

func buildScanResultSectionsWithoutMatches(outcome scanExecutionOutcome, colorMode view.ColorMode) []string {
	return buildScanResultSections(scanResultsViewInput{
		unknown:   outcome.unknown,
		unsure:    outcome.unsure,
		colorMode: colorMode,
	})
}

func buildScanPromptTranscriptSections(outcome scanExecutionOutcome, colorMode view.ColorMode, prompt scanConfirmPromptModel) []string {
	sections := buildScanResultSectionsForTranscript(outcome, colorMode)
	return append(sections, prompt.View())
}

func buildScanPromptTranscriptSectionsWithAdded(outcome scanExecutionOutcome, colorMode view.ColorMode, prompt scanConfirmPromptModel) []string {
	sections := buildScanResultSectionsWithoutMatches(outcome, colorMode)
	sections = append(sections, prompt.View())
	return appendScanAddedSection(sections, outcome.added, colorMode)
}

func appendScanAddedSection(sections []string, added []scanMatch, colorMode view.ColorMode) []string {
	if len(added) == 0 {
		return sections
	}
	return append(sections, renderScanAddedSection(scanAddedViewInput{
		added:     added,
		colorMode: colorMode,
	}))
}

func colorModeForOutput(writer io.Writer) view.ColorMode {
	if writer == nil {
		return view.ColorDisabled
	}
	if !view.SupportsColor(writer) || !view.SupportsControlSequences(writer) {
		return view.ColorDisabled
	}
	return view.ColorEnabled
}

func resolvePreferredPlatform(value string) (models.Platform, error) {
	preferPlatform := normalizePlatform(value)
	if preferPlatform != models.MODRINTH && preferPlatform != models.CURSEFORGE {
		return "", errors.New(i18n.T("cmd.scan.error.unknown_platform", &i18n.Tvars{
			Data: &i18n.TData{"platform": value},
		}))
	}
	return preferPlatform, nil
}
