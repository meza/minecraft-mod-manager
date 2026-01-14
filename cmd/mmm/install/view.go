package install

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func renderInstallRunningView(colorMode view.ColorMode, items []installItem, footerLine string) string {
	header := i18n.T("cmd.install.header.running", nil)
	lines := renderInstallItemLines(colorMode, items)
	if strings.TrimSpace(footerLine) != "" {
		lines = append(lines, footerLine)
	}
	return strings.Join(append([]string{header}, lines...), "\n")
}

func renderInstallSuccessView(colorMode view.ColorMode, items []installItem) string {
	lines := renderInstallItemLines(colorMode, items)
	sections := []string{
		strings.Join(lines, "\n"),
		renderInstallSuccessSummary(colorMode),
	}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderInstallDownloadFailedViewWithHint(colorMode view.ColorMode, items []installItem) string {
	return renderInstallDownloadFailedViewWithSummary(colorMode, items, renderInstallDownloadFailureSummaryWithHint(colorMode))
}

func renderInstallDownloadFailedViewWithSummary(colorMode view.ColorMode, items []installItem, summaryLines []string) string {
	header := i18n.T("cmd.install.header.running", nil)
	lines := renderInstallItemLines(colorMode, items)
	sections := []string{
		strings.Join(append([]string{header}, lines...), "\n"),
		strings.Join(summaryLines, "\n"),
	}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderInstallWriteLockFailedView(colorMode view.ColorMode, items []installItem, lockPath string) string {
	header := i18n.T("cmd.install.header.running", nil)
	lines := renderInstallItemLines(colorMode, items)
	sections := []string{
		strings.Join(append([]string{header}, lines...), "\n"),
		strings.Join(renderInstallWriteLockSummary(colorMode, lockPath), "\n"),
	}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderInstallWriteConfigFailedView(colorMode view.ColorMode, items []installItem, configPath string) string {
	header := i18n.T("cmd.install.header.running", nil)
	lines := renderInstallItemLines(colorMode, items)
	sections := []string{
		strings.Join(append([]string{header}, lines...), "\n"),
		strings.Join(renderInstallWriteConfigSummary(colorMode, configPath), "\n"),
	}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderInstallExecutionFailedView(colorMode view.ColorMode, items []installItem, err error) string {
	header := i18n.T("cmd.install.header.running", nil)
	lines := renderInstallItemLines(colorMode, items)
	sections := []string{
		strings.Join(append([]string{header}, lines...), "\n"),
		strings.Join(renderInstallExecutionFailureSummary(colorMode, err), "\n"),
	}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderInstallCanceledView(colorMode view.ColorMode, items []installItem) string {
	header := i18n.T("cmd.install.header.running", nil)
	lines := renderInstallItemLines(colorMode, items)
	sections := []string{
		strings.Join(append([]string{header}, lines...), "\n"),
		strings.Join(renderInstallCanceledSummary(colorMode), "\n"),
	}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderInstallItemLines(colorMode view.ColorMode, items []installItem) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, renderInstallItemLine(colorMode, item))
	}
	return lines
}

func renderInstallItemLine(colorMode view.ColorMode, item installItem) string {
	label := view.RenderModLabel(colorMode, item.DisplayName, item.Mod.ID, string(item.Mod.Type))
	status := view.ModItemStatusPending
	suffix := ""
	var progress *view.ProgressDetails

	switch item.Status {
	case installItemDownloading:
		status = view.ModItemStatusDownloading
		if item.Progress != nil {
			progress = &view.ProgressDetails{
				Bar:        view.NewProgressBar(),
				Ratio:      item.Progress.ratio,
				Downloaded: item.Progress.downloaded,
				Total:      item.Progress.total,
			}
		}
	case installItemSuccess:
		status = view.ModItemStatusSuccess
	case installItemFailed:
		status = view.ModItemStatusError
		if strings.TrimSpace(item.FailureReason) != "" {
			suffix = i18n.T("cmd.download.item.failed", &i18n.Tvars{
				Data: &i18n.TData{"reason": item.FailureReason},
			})
		}
	case installItemAborted:
		status = view.ModItemStatusError
		suffix = i18n.T("cmd.install.item.aborted", nil)
	}

	return view.RenderModItemLine(view.ModItemLine{
		Label:    label,
		Suffix:   suffix,
		Status:   status,
		Progress: progress,
	}, colorMode)
}

func renderInstallSuccessSummary(colorMode view.ColorMode) string {
	return fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), i18n.T("cmd.install.summary.success", nil))
}

func renderInstallDownloadFailureSummary(colorMode view.ColorMode) []string {
	lines := []string{
		renderFinalErrorLine(colorMode, i18n.T("cmd.install.summary.download_failed", nil)),
		i18n.T("cmd.install.summary.partial", &i18n.Tvars{
			Data: &i18n.TData{"errorIcon": view.ErrorIcon(colorMode)},
		}),
	}
	return lines
}

func renderInstallDownloadFailureSummaryWithHint(colorMode view.ColorMode) []string {
	lines := renderInstallDownloadFailureSummary(colorMode)
	hint := i18n.T("cmd.install.summary.download_failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return append(lines, "", hint)
}

func renderInstallWriteLockSummary(colorMode view.ColorMode, lockPath string) []string {
	hint := i18n.T("cmd.install.summary.write_failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return []string{
		renderFinalErrorLine(colorMode, i18n.T("cmd.install.summary.write_failed_abort", &i18n.Tvars{
			Data: &i18n.TData{"file": filepath.Base(lockPath)},
		})),
		hint,
	}
}

func renderInstallWriteConfigSummary(colorMode view.ColorMode, configPath string) []string {
	hint := i18n.T("cmd.install.summary.write_failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return []string{
		renderFinalErrorLine(colorMode, i18n.T("cmd.install.summary.write_failed_abort", &i18n.Tvars{
			Data: &i18n.TData{"file": filepath.Base(configPath)},
		})),
		hint,
	}
}

func renderInstallExecutionFailureSummary(colorMode view.ColorMode, err error) []string {
	reason := ""
	if err != nil {
		reason = err.Error()
	}
	headline := renderFinalErrorLine(colorMode, i18n.T("cmd.install.error.failed", &i18n.Tvars{
		Data: &i18n.TData{"reason": reason},
	}))
	hint := i18n.T("cmd.install.error.failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}
	return []string{headline, hint}
}

func renderInstallCanceledSummary(colorMode view.ColorMode) []string {
	return []string{
		renderFinalErrorLine(colorMode, i18n.T("cmd.install.summary.canceled", nil)),
		i18n.T("cmd.install.summary.partial", &i18n.Tvars{
			Data: &i18n.TData{"errorIcon": view.ErrorIcon(colorMode)},
		}),
	}
}

func renderFinalErrorLine(colorMode view.ColorMode, message string) string {
	line := fmt.Sprintf("%s %s", view.FinalErrorIcon(colorMode), message)
	if colorMode.Enabled() {
		return view.ErrorStyle.Render(line)
	}
	return line
}
