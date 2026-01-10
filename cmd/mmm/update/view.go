package update

import (
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type updateRunningViewInput struct {
	items        []updateItem
	colorMode    view.ColorMode
	spinnerFrame string
}

type updateResultsViewInput struct {
	items     []updateItem
	colorMode view.ColorMode
}

type updateErrorViewInput struct {
	colorMode  view.ColorMode
	lockPath   string
	configPath string
}

func renderUpdateRunningView(input updateRunningViewInput) string {
	sections := buildUpdateRunningSections(input)
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderUpdateResultsView(input updateResultsViewInput) string {
	sections := buildUpdateResultSections(input)
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderUpdateInstallFailureView(colorMode view.ColorMode) string {
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.update.error.install_failed", nil))
	hint := i18n.T("cmd.update.error.install_failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return view.RenderViewSections([]string{headline, hint}, view.SectionSeparatorParagraph)
}

func renderUpdateWriteLockFailureView(input updateErrorViewInput) string {
	headline := renderFinalErrorLine(input.colorMode, i18n.T("cmd.update.error.write_lock", &i18n.Tvars{
		Data: &i18n.TData{"path": input.lockPath},
	}))
	hint := i18n.T("cmd.update.error.write_hint", nil)
	if input.colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return view.RenderViewSections([]string{headline, hint}, view.SectionSeparatorParagraph)
}

func renderUpdateWriteConfigFailureView(input updateErrorViewInput) string {
	headline := renderFinalErrorLine(input.colorMode, i18n.T("cmd.update.error.write_config", &i18n.Tvars{
		Data: &i18n.TData{"path": input.configPath},
	}))
	hint := i18n.T("cmd.update.error.write_hint", nil)
	if input.colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return view.RenderViewSections([]string{headline, hint}, view.SectionSeparatorParagraph)
}

func renderFinalErrorLine(colorMode view.ColorMode, message string) string {
	line := fmt.Sprintf("%s %s", view.FinalErrorIcon(colorMode), message)
	if colorMode.Enabled() {
		return view.ErrorStyle.Render(line)
	}
	return line
}

func buildUpdateRunningSections(input updateRunningViewInput) []string {
	upToDate := filterUpdateItems(input.items, updateItemStatusUpToDate)
	updated := filterUpdateItems(input.items, updateItemStatusUpdated)
	skipped := filterUpdateItems(input.items, updateItemStatusSkipped)
	failed := filterUpdateItems(input.items, updateItemStatusFailed)
	updating := make([]updateItem, 0, len(input.items))
	for _, item := range input.items {
		if item.Status == updateItemStatusPending || item.Status == updateItemStatusUpdating || item.Status == updateItemStatusDownloading {
			updating = append(updating, item)
		}
	}

	sections := make([]string, 0, 5)
	if len(upToDate) > 0 {
		sections = append(sections, renderUpdateSection(i18n.T("cmd.update.section.up_to_date", nil), input, upToDate))
	}
	if len(updated) > 0 {
		sections = append(sections, renderUpdateSection(i18n.T("cmd.update.section.updated", nil), input, updated))
	}
	if len(skipped) > 0 {
		sections = append(sections, renderUpdateSection(i18n.T("cmd.update.section.skipped", nil), input, skipped))
	}
	if len(failed) > 0 {
		sections = append(sections, renderUpdateSection(i18n.T("cmd.update.section.failed", nil), input, failed))
	}
	updatingHeader := i18n.T("cmd.update.section.updating", nil)
	if len(upToDate)+len(updated)+len(skipped)+len(failed) > 0 {
		updatingHeader = fmt.Sprintf("%s:", i18n.T("cmd.update.section.updating_label", nil))
	}
	sections = append(sections, renderUpdateSection(updatingHeader, input, updating))

	return pruneEmptySections(sections)
}

func buildUpdateResultSections(input updateResultsViewInput) []string {
	upToDate := filterUpdateItems(input.items, updateItemStatusUpToDate)
	updated := filterUpdateItems(input.items, updateItemStatusUpdated)
	skipped := filterUpdateItems(input.items, updateItemStatusSkipped)
	failed := filterUpdateItems(input.items, updateItemStatusFailed)

	sections := make([]string, 0, 6)
	if len(upToDate) > 0 {
		sections = append(sections, renderUpdateSection(i18n.T("cmd.update.section.up_to_date", nil), updateRunningViewInput{
			items:     upToDate,
			colorMode: input.colorMode,
		}, upToDate))
	}
	if len(updated) > 0 {
		sections = append(sections, renderUpdateSection(i18n.T("cmd.update.section.updated", nil), updateRunningViewInput{
			items:     updated,
			colorMode: input.colorMode,
		}, updated))
	}
	if len(skipped) > 0 {
		sections = append(sections, renderUpdateSection(i18n.T("cmd.update.section.skipped", nil), updateRunningViewInput{
			items:     skipped,
			colorMode: input.colorMode,
		}, skipped))
	}
	if len(failed) > 0 {
		sections = append(sections, renderUpdateSection(i18n.T("cmd.update.section.failed", nil), updateRunningViewInput{
			items:     failed,
			colorMode: input.colorMode,
		}, failed))
	}

	sections = append(sections, renderUpdateSummarySections(input)...)
	return pruneEmptySections(sections)
}

func renderUpdateSection(header string, input updateRunningViewInput, items []updateItem) string {
	if len(items) == 0 {
		return ""
	}
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, header)
	for _, item := range items {
		lines = append(lines, renderUpdateItemLine(input, item))
	}
	return strings.Join(lines, "\n")
}

func renderUpdateSummarySections(input updateResultsViewInput) []string {
	hasFailure := hasUpdateStatus(input.items, updateItemStatusFailed)
	if hasFailure {
		headline := renderFinalErrorLine(input.colorMode, i18n.T("cmd.update.summary.incomplete", nil))
		hint := i18n.T("cmd.update.summary.incomplete_hint", nil)
		if input.colorMode.Enabled() {
			hint = view.CtaStyle.Render(hint)
		}
		return []string{headline, hint}
	}
	return []string{renderUpdateSuccessSummary(input.colorMode)}
}

func renderUpdateSuccessSummary(colorMode view.ColorMode) string {
	return fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), i18n.T("cmd.update.summary.success", nil))
}

func renderUpdateTranscriptSummary(colorMode view.ColorMode, items []updateItem) []string {
	if hasUpdateStatus(items, updateItemStatusFailed) {
		headline := renderFinalErrorLine(colorMode, i18n.T("cmd.update.summary.incomplete", nil))
		hint := i18n.T("cmd.update.summary.incomplete_hint", nil)
		if colorMode.Enabled() {
			hint = view.CtaStyle.Render(hint)
		}
		return []string{headline, "", hint}
	}
	return []string{renderUpdateSuccessSummary(colorMode)}
}

func renderUpdateItemLine(input updateRunningViewInput, item updateItem) string {
	label := view.RenderModLabel(input.colorMode, item.DisplayName, item.Mod.ID, string(item.Mod.Type))
	suffix := ""
	switch item.Status {
	case updateItemStatusFailed:
		suffix = i18n.T("cmd.update.item.failed", &i18n.Tvars{
			Data: &i18n.TData{"reason": item.FailReason},
		})
	case updateItemStatusSkipped:
		suffix = i18n.T("cmd.update.item.skipped", nil)
	}

	icon := renderUpdateItemIcon(input, item)
	message := label
	if strings.TrimSpace(suffix) != "" {
		message = fmt.Sprintf("%s %s", label, suffix)
	}
	lines := []string{fmt.Sprintf("%s %s", icon, message)}
	if item.Status == updateItemStatusDownloading && item.Progress != nil {
		bar := view.NewProgressBar()
		lines = append(lines,
			view.RenderProgressBar(bar, item.Progress.ratio),
			view.ProgressPercentLine(item.Progress.ratio, item.Progress.downloaded, item.Progress.total),
		)
	}
	return strings.Join(lines, "\n")
}

func renderUpdateItemIcon(input updateRunningViewInput, item updateItem) string {
	switch item.Status {
	case updateItemStatusUpdated, updateItemStatusUpToDate:
		return view.SuccessIcon(input.colorMode)
	case updateItemStatusFailed:
		return view.ErrorIcon(input.colorMode)
	case updateItemStatusSkipped:
		return view.PinnedIcon(input.colorMode)
	case updateItemStatusDownloading:
		return view.DownloadIcon(input.colorMode)
	case updateItemStatusUpdating:
		frame := strings.TrimSpace(input.spinnerFrame)
		if frame == "" {
			return view.PendingIcon(input.colorMode)
		}
		return view.RenderIfColorEnabled(input.colorMode, view.QuestionStyle, frame)
	case updateItemStatusPending:
		return view.PendingIcon(input.colorMode)
	default:
		return view.PendingIcon(input.colorMode)
	}
}

func filterUpdateItems(items []updateItem, status updateItemStatus) []updateItem {
	filtered := make([]updateItem, 0, len(items))
	for _, item := range items {
		if item.Status == status {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func hasUpdateStatus(items []updateItem, status updateItemStatus) bool {
	for _, item := range items {
		if item.Status == status {
			return true
		}
	}
	return false
}

func pruneEmptySections(sections []string) []string {
	pruned := make([]string, 0, len(sections))
	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}
		pruned = append(pruned, section)
	}
	return pruned
}
