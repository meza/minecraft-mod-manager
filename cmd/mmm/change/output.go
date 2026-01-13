package change

import (
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

func writeChangeOutcome(cmd *cobra.Command, deps changeDeps, outcome changeOutcome, targetVersion string, mode interaction.ExecutionMode) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	spin := view.NewSpinner()
	var spinner *view.Spinner
	if mode != interaction.ExecutionModeNonTTY {
		spinner = &spin
	}

	sections := buildChangeSections(changeViewInput{
		stage:       outcome.Stage,
		target:      targetVersion,
		items:       outcome.Items,
		colorMode:   colorMode,
		spinner:     spinner,
		forcePolicy: outcome.ForcePolicy,
	})
	if mode == interaction.ExecutionModeNonTTY && outcome.Stage == changeStageCompatibilityFailed {
		sections = buildNonTTYCompatibilityFailureSections(changeViewInput{
			stage:     outcome.Stage,
			target:    targetVersion,
			items:     outcome.Items,
			colorMode: colorMode,
		})
	}
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), sections)
}

func writeQuietChangeOutcome(cmd *cobra.Command, deps changeDeps, outcome changeOutcome, targetVersion string) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	lines := buildQuietChangeLines(outcome, targetVersion, colorMode)
	if len(lines) == 0 {
		return nil
	}
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), lines)
}

func writeChangePolicyFlagError(cmd *cobra.Command, deps changeDeps, err changePolicyFlagError) error {
	colorMode := colorModeForOutput(cmd.OutOrStdout())
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.change.error.invalid_flags", nil))

	bodyLines := []string{}
	switch err.kind {
	case changePolicyFlagErrorRequiresForce:
		bodyLines = append(bodyLines, i18n.T("cmd.change.error.force_policy_requires_force", nil))
	default:
		bodyLines = append(bodyLines, i18n.T("cmd.change.error.force_policy_multiple", nil))
		bodyLines = append(bodyLines, i18n.T("cmd.change.error.force_policy_choose", nil))
	}

	body := strings.Join(bodyLines, "\n")
	return runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{headline, body})
}

func buildNonTTYCompatibilityFailureSections(input changeViewInput) []string {
	filtered := filterCompatibilityFailureFinalItems(input.items)
	if len(filtered) == 0 {
		return []string{renderCompatibilityErrorFooterNonTTY(input)}
	}
	lines := make([]string, 0, len(filtered))
	for _, item := range filtered {
		lines = append(lines, renderNonTTYCompatibilityLine(input, item))
	}
	return []string{strings.Join(lines, "\n"), renderCompatibilityErrorFooterNonTTY(input)}
}

func renderNonTTYCompatibilityLine(input changeViewInput, item changeItem) string {
	label := renderModLabel(input, item)
	status := view.ModItemStatusPending
	suffix := ""

	switch item.CompatStatus {
	case changeCompatSupported:
		status = view.ModItemStatusPending
	case changeCompatUnsupported:
		status = view.ModItemStatusError
		suffix = i18n.T("cmd.change.item.unsupported", &i18n.Tvars{
			Data: &i18n.TData{"version": input.target},
		})
	}

	return view.RenderModItemLine(view.ModItemLine{
		Label:  label,
		Suffix: suffix,
		Status: status,
	}, input.colorMode)
}

func renderCompatibilityErrorFooterNonTTY(input changeViewInput) string {
	headline := renderFinalErrorLine(input.colorMode, i18n.T("cmd.change.error.compatibility_failed", &i18n.Tvars{
		Data: &i18n.TData{"version": input.target},
	}))
	body := i18n.T("cmd.change.error.no_changes", nil)
	return strings.Join([]string{headline, body}, "\n")
}

func buildQuietChangeLines(outcome changeOutcome, targetVersion string, colorMode view.ColorMode) []string {
	switch outcome.Stage {
	case changeStageSuccess:
		if countSkipped(outcome.Items) == 0 {
			return nil
		}
		return []string{renderQuietSkippedSection(outcome.Items, targetVersion, colorMode, outcome.ForcePolicy)}
	case changeStageCompatibilityFailed:
		headline := renderFinalErrorLine(colorMode, i18n.T("cmd.change.error.compatibility_failed_quiet", &i18n.Tvars{
			Data: &i18n.TData{"version": targetVersion},
		}))
		list := renderQuietCompatibilityFailures(outcome.Items, targetVersion, colorMode)
		return pruneEmptySections([]string{headline, list})
	case changeStageDownloadFailed:
		headline := renderFinalErrorLine(colorMode, i18n.T("cmd.change.error.download_failed", nil))
		list := renderQuietDownloadFailures(outcome.Items, colorMode)
		return pruneEmptySections([]string{headline, list})
	case changeStageSwitchFailed:
		headline := renderFinalErrorLine(colorMode, i18n.T("cmd.change.error.switch_failed", nil))
		list := renderQuietSwitchFailures(outcome.Items, colorMode)
		return pruneEmptySections([]string{headline, list})
	default:
		return nil
	}
}

func renderQuietCompatibilityFailures(items []changeItem, targetVersion string, colorMode view.ColorMode) string {
	return renderQuietFailures(items, colorMode, func(item changeItem) bool {
		return item.CompatStatus == changeCompatUnsupported
	}, func(item changeItem) string {
		return i18n.T("cmd.change.item.unsupported", &i18n.Tvars{Data: &i18n.TData{"version": targetVersion}})
	})
}

func renderQuietDownloadFailures(items []changeItem, colorMode view.ColorMode) string {
	return renderQuietFailures(items, colorMode, func(item changeItem) bool {
		return item.DownloadStatus == changeDownloadFailed
	}, func(item changeItem) string {
		return i18n.T("cmd.change.item.download_failed", &i18n.Tvars{Data: &i18n.TData{"reason": item.ErrorReason}})
	})
}

func renderQuietSwitchFailures(items []changeItem, colorMode view.ColorMode) string {
	return renderQuietFailures(items, colorMode, func(item changeItem) bool {
		return item.SwitchStatus == changeSwitchFailed
	}, func(item changeItem) string {
		return i18n.T("cmd.change.item.switch_failed", &i18n.Tvars{Data: &i18n.TData{"reason": item.ErrorReason}})
	})
}

func renderQuietFailures(items []changeItem, colorMode view.ColorMode, include func(changeItem) bool, suffix func(changeItem) string) string {
	lines := make([]string, 0)
	for _, item := range items {
		if !include(item) {
			continue
		}
		label := renderModLabel(changeViewInput{colorMode: colorMode}, item)
		lines = append(lines, view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Suffix: suffix(item),
			Status: view.ModItemStatusError,
		}, colorMode))
	}
	return strings.Join(lines, "\n")
}

func renderQuietSkippedSection(items []changeItem, targetVersion string, colorMode view.ColorMode, policy changeForcePolicy) string {
	lines := []string{i18n.T(skippedSectionHeader(policy), nil)}
	for _, item := range items {
		if !item.Skipped {
			continue
		}
		label := renderModLabel(changeViewInput{colorMode: colorMode, target: targetVersion}, item)
		lines = append(lines, view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Suffix: i18n.T("cmd.change.item.unsupported", &i18n.Tvars{Data: &i18n.TData{"version": targetVersion}}),
			Status: view.ModItemStatusError,
		}, colorMode))
	}
	return strings.Join(lines, "\n")
}
