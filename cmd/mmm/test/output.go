package test

import (
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type testViewInput struct {
	targetVersion string
	items         []testItem
	colorMode     view.ColorMode
	spinnerFrame  string
}

func renderTestHeader(version string) string {
	return i18n.T("cmd.test.header", &i18n.Tvars{
		Data: &i18n.TData{"version": version},
	})
}

func renderTestRunningView(input testViewInput) string {
	sections := []string{renderTestHeader(input.targetVersion)}
	if hasUnsupportedTestItems(input.items) {
		sections = append(sections, renderRunningCompatibleSection(input))
		sections = append(sections, renderRunningNotCompatibleSection(input))
	} else {
		sections = append(sections, renderCompatibilitySection(input))
	}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func renderTestFinalView(input testViewInput) string {
	sections := buildTestSections(input)
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func buildTestSections(input testViewInput) []string {
	sections := []string{renderTestHeader(input.targetVersion)}

	supported := filterTestItems(input.items, testItemStatusSupported)
	unsupported := filterTestItems(input.items, testItemStatusUnsupported)
	inconclusive := filterTestItems(input.items, testItemStatusInconclusive)

	if len(supported) > 0 {
		sections = append(sections, renderCompatibleSection(input, supported))
	}
	if len(unsupported) > 0 {
		sections = append(sections, renderNotCompatibleSection(input, unsupported))
	}
	if len(inconclusive) > 0 {
		sections = append(sections, renderInconclusiveSection(input, inconclusive))
	}

	switch {
	case len(inconclusive) > 0:
		hint := i18n.T("cmd.test.summary.inconclusive_hint", &i18n.Tvars{
			Data: &i18n.TData{"version": input.targetVersion},
		})
		if input.colorMode.Enabled() {
			hint = view.CtaStyle.Render(hint)
		}
		sections = append(sections,
			renderFinalErrorLine(input.colorMode, i18n.T("cmd.test.summary.inconclusive", &i18n.Tvars{
				Data: &i18n.TData{"version": input.targetVersion},
			})),
			hint,
		)
	case len(unsupported) > 0:
		sections = append(sections, renderFinalErrorLine(input.colorMode, i18n.T("cmd.test.summary.unsupported", &i18n.Tvars{
			Data: &i18n.TData{"version": input.targetVersion},
		})))
	default:
		sections = append(sections, renderSuccessLine(input.colorMode, i18n.T("cmd.test.success", &i18n.Tvars{
			Data: &i18n.TData{"version": input.targetVersion},
		})))
	}

	return pruneEmptySections(sections)
}

func renderCompatibilitySection(input testViewInput) string {
	lines := make([]string, 0, len(input.items)+1)
	lines = append(lines, i18n.T("cmd.test.section.compatibility", nil))
	for _, item := range input.items {
		lines = append(lines, renderTestRunningItemLine(input, item))
	}
	return strings.Join(lines, "\n")
}

func renderRunningCompatibleSection(input testViewInput) string {
	items := filterTestItemsExcluding(input.items, testItemStatusUnsupported)
	return renderTestSection(items, i18n.T("cmd.test.section.compatible", nil), func(item testItem) string {
		return renderTestRunningItemLine(input, item)
	})
}

func renderRunningNotCompatibleSection(input testViewInput) string {
	items := filterTestItems(input.items, testItemStatusUnsupported)
	return renderTestSection(items, i18n.T("cmd.test.section.not_compatible", nil), func(item testItem) string {
		return renderTestRunningItemLine(input, item)
	})
}

func renderCompatibleSection(input testViewInput, items []testItem) string {
	return renderTestSection(items, i18n.T("cmd.test.section.compatible", nil), func(item testItem) string {
		return renderTestItemLine(input.colorMode, item)
	})
}

func renderNotCompatibleSection(input testViewInput, items []testItem) string {
	return renderTestSection(items, i18n.T("cmd.test.section.not_compatible", nil), func(item testItem) string {
		return renderTestItemLine(input.colorMode, item)
	})
}

func renderInconclusiveSection(input testViewInput, items []testItem) string {
	return renderTestSection(items, i18n.T("cmd.test.section.inconclusive", nil), func(item testItem) string {
		return renderTestItemLineWithReason(input.colorMode, item)
	})
}

func renderTestSection(
	items []testItem,
	header string,
	itemLine func(testItem) string,
) string {
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, header)
	for _, item := range items {
		lines = append(lines, itemLine(item))
	}
	return strings.Join(lines, "\n")
}

func renderTestRunningItemLine(input testViewInput, item testItem) string {
	label := renderTestModLabel(input.colorMode, item.Mod)
	switch item.Status {
	case testItemStatusSupported:
		return view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Status: view.ModItemStatusSuccess,
		}, input.colorMode)
	case testItemStatusUnsupported:
		return view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Status: view.ModItemStatusError,
		}, input.colorMode)
	case testItemStatusInconclusive:
		return fmt.Sprintf("%s %s", inconclusiveIcon(input.colorMode), label)
	default:
		return view.RenderModItemLine(view.ModItemLine{
			Label:        label,
			Status:       view.ModItemStatusSpinning,
			SpinnerFrame: input.spinnerFrame,
		}, input.colorMode)
	}
}

func renderTestItemLine(colorMode view.ColorMode, item testItem) string {
	label := renderTestModLabel(colorMode, item.Mod)
	switch item.Status {
	case testItemStatusSupported:
		return view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Status: view.ModItemStatusSuccess,
		}, colorMode)
	case testItemStatusUnsupported:
		return view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Status: view.ModItemStatusError,
		}, colorMode)
	case testItemStatusInconclusive:
		return fmt.Sprintf("%s %s", inconclusiveIcon(colorMode), label)
	default:
		return view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Status: view.ModItemStatusPending,
		}, colorMode)
	}
}

func renderTestItemLineWithReason(colorMode view.ColorMode, item testItem) string {
	if item.Status == testItemStatusInconclusive && strings.TrimSpace(item.Reason) != "" {
		label := renderTestModLabel(colorMode, item.Mod)
		return fmt.Sprintf("%s %s %s", inconclusiveIcon(colorMode), label, item.Reason)
	}
	return renderTestItemLine(colorMode, item)
}

func renderTestModLabel(colorMode view.ColorMode, mod models.Mod) string {
	return view.RenderModLabel(colorMode, modDisplayName(mod), mod.ID, string(mod.Type))
}

func renderSuccessLine(colorMode view.ColorMode, message string) string {
	return fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), message)
}

func renderFinalErrorLine(colorMode view.ColorMode, message string) string {
	line := fmt.Sprintf("%s %s", view.FinalErrorIcon(colorMode), message)
	if colorMode.Enabled() {
		return view.ErrorStyle.Render(line)
	}
	return line
}

func inconclusiveIcon(colorMode view.ColorMode) string {
	icon := "?"
	if view.SupportsUnicode() {
		icon = "\u2754"
	}
	return view.RenderIfColorEnabled(colorMode, view.QuestionStyle, icon)
}

func filterTestItems(items []testItem, status testItemStatus) []testItem {
	filtered := make([]testItem, 0, len(items))
	for _, item := range items {
		if item.Status == status {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterTestItemsExcluding(items []testItem, status testItemStatus) []testItem {
	filtered := make([]testItem, 0, len(items))
	for _, item := range items {
		if item.Status != status {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func hasUnsupportedTestItems(items []testItem) bool {
	for _, item := range items {
		if item.Status == testItemStatusUnsupported {
			return true
		}
	}
	return false
}

func buildQuietTestLines(items []testItem, colorMode view.ColorMode) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		switch item.Status {
		case testItemStatusUnsupported:
			lines = append(lines, renderTestItemLine(colorMode, item))
		case testItemStatusInconclusive:
			lines = append(lines, renderTestItemLine(colorMode, item))
		default:
			continue
		}
	}
	return lines
}

func pruneEmptySections(sections []string) []string {
	filtered := make([]string, 0, len(sections))
	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}
		filtered = append(filtered, section)
	}
	return filtered
}
