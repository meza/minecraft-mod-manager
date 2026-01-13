package remove

import (
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func renderRemoveLabel(colorMode view.ColorMode, mod models.Mod) string {
	idPart := view.RenderIfColorEnabled(colorMode, view.ParenStyle, mod.ID)
	return fmt.Sprintf("%s (%s)", mod.Name, idPart)
}

func renderRemoveItemLine(colorMode view.ColorMode, item removeItem, spinner *view.Spinner) string {
	status := view.ModItemStatusPending
	suffix := ""
	line := view.ModItemLine{
		Label: renderRemoveLabel(colorMode, item.Mod),
	}

	switch item.Status {
	case removeItemPending:
		status = view.ModItemStatusSpinning
		line.Spinner = spinner
	case removeItemSuccess:
		status = view.ModItemStatusSuccess
	case removeItemFailed:
		status = view.ModItemStatusError
		suffix = i18n.T("cmd.remove.item.delete_failed", &i18n.Tvars{
			Data: &i18n.TData{"reason": item.FailureReason},
		})
	}

	line.Status = status
	line.Suffix = suffix
	return view.RenderModItemLine(line, colorMode)
}

func renderRemoveRunningSection(colorMode view.ColorMode, items []removeItem, spinner *view.Spinner) string {
	header := i18n.T("cmd.remove.header.removing", nil)
	lines := renderRemoveItems(colorMode, items, spinner)
	return strings.Join(append([]string{header}, lines...), "\n")
}

func renderRemoveResultSection(colorMode view.ColorMode, items []removeItem) string {
	lines := renderRemoveItems(colorMode, items, nil)
	return strings.Join(lines, "\n")
}

func renderRemoveDryRunSection(colorMode view.ColorMode, items []removeItem) string {
	header := i18n.T("cmd.remove.header.would_remove", nil)
	lines := renderRemoveDryRunItems(colorMode, items)
	return strings.Join(append([]string{header}, lines...), "\n")
}

func renderRemoveNoMatchesLine() string {
	return i18n.T("cmd.remove.no_matches", nil)
}

func renderRemoveSuccessSummary(colorMode view.ColorMode) string {
	return fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), i18n.T("cmd.remove.summary.success", nil))
}

func renderRemoveFailureSummary(colorMode view.ColorMode) string {
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.remove.summary.incomplete", nil))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.remove.summary.incomplete_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return strings.Join([]string{headline, hint}, "\n")
}

func renderRemoveFailureLine(colorMode view.ColorMode, err error) string {
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.remove.error.failed", &i18n.Tvars{
		Data: &i18n.TData{"reason": err.Error()},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	hint := i18n.T("cmd.remove.error.failed_hint", nil)
	if colorMode.Enabled() {
		hint = view.CtaStyle.Render(hint)
	}

	return strings.Join([]string{headline, hint}, "\n")
}

func renderRemoveItems(colorMode view.ColorMode, items []removeItem, spinner *view.Spinner) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, renderRemoveItemLine(colorMode, item, spinner))
	}
	return lines
}

func renderRemoveDryRunItems(colorMode view.ColorMode, items []removeItem) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, renderRemoveDryRunLine(colorMode, item.Mod))
	}
	return lines
}

func renderRemoveDryRunLine(colorMode view.ColorMode, mod models.Mod) string {
	icon := uncertaintyIcon(colorMode)
	label := renderRemoveLabel(colorMode, mod)
	return fmt.Sprintf("%s %s", icon, label)
}

func uncertaintyIcon(colorMode view.ColorMode) string {
	icon := "?"
	if view.SupportsUnicode() {
		icon = "\u2754"
	}
	return view.RenderIfColorEnabled(colorMode, view.QuestionStyle, icon)
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}
