package scan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type scanRunningViewInput struct {
	items        []scanItem
	colorMode    view.ColorMode
	spinnerFrame string
}

type scanResultsViewInput struct {
	matches   []scanMatch
	unknown   []string
	unsure    []scanUnsure
	colorMode view.ColorMode
}

type scanAddedViewInput struct {
	added     []scanMatch
	colorMode view.ColorMode
}

func renderScanRunningView(input scanRunningViewInput) string {
	items := cloneScanItems(input.items)
	sortScanItemsByFile(items)

	running := filterScanItems(items, scanItemStatusScanning, scanItemStatusPending)
	recognized := filterScanItems(items, scanItemStatusRecognized)
	unknown := filterScanItems(items, scanItemStatusUnknown)
	unsure := filterScanItems(items, scanItemStatusUnsure)

	header := i18n.T("cmd.scan.header.running", nil)
	sections := make([]string, 0, 4)
	sections = append(sections, renderScanRunningSection(header, input, running))

	if len(recognized) > 0 {
		sections = append(sections, renderScanRecognizedSection(input.colorMode, matchesFromItems(recognized)))
	}
	if len(unknown) > 0 {
		sections = append(sections, renderScanUnknownSection(input.colorMode, fileNamesFromItems(unknown)))
	}
	if len(unsure) > 0 {
		sections = append(sections, renderScanUnsureSection(input.colorMode, unsureFromItems(unsure)))
	}

	return view.RenderViewSections(pruneEmptySections(sections), view.SectionSeparatorParagraph)
}

func renderScanResultsView(input scanResultsViewInput) string {
	sections := buildScanResultSections(input)
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func buildScanResultSections(input scanResultsViewInput) []string {
	sections := []string{i18n.T("cmd.scan.header.results", nil)}

	if len(input.matches) > 0 {
		sections = append(sections, renderScanRecognizedSection(input.colorMode, input.matches))
	}
	if len(input.unknown) > 0 {
		sections = append(sections, renderScanUnknownSection(input.colorMode, input.unknown))
	}
	if len(input.unsure) > 0 {
		sections = append(sections, renderScanUnsureSection(input.colorMode, input.unsure))
	}

	return pruneEmptySections(sections)
}

func renderScanAddedSection(input scanAddedViewInput) string {
	lines := make([]string, 0, len(input.added)+1)
	lines = append(lines, i18n.T("cmd.scan.section.added", nil))

	items := cloneScanMatches(input.added)
	sortScanMatchesByName(items)
	for _, match := range items {
		lines = append(lines, renderScanAddedLine(input.colorMode, match))
	}

	return strings.Join(lines, "\n")
}

func renderScanQuietView(input scanResultsViewInput) string {
	sections := make([]string, 0, 2)
	if len(input.unknown) > 0 {
		sections = append(sections, renderScanUnknownSection(input.colorMode, input.unknown))
	}
	if len(input.unsure) > 0 {
		sections = append(sections, renderScanUnsureSection(input.colorMode, input.unsure))
	}
	return view.RenderViewSections(pruneEmptySections(sections), view.SectionSeparatorParagraph)
}

func renderScanFailureLine(colorMode view.ColorMode, err error) string {
	line := fmt.Sprintf("%s %s", view.FinalErrorIcon(colorMode), i18n.T("cmd.scan.error.failed", &i18n.Tvars{
		Data: &i18n.TData{"reason": err.Error()},
	}))
	if colorMode.Enabled() {
		return view.ErrorStyle.Render(line)
	}
	return line
}

func renderScanAdoptionCancelledLine(colorMode view.ColorMode) string {
	line := fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), i18n.T("cmd.scan.adoption.cancelled", nil))
	return line
}

func renderScanRunningSection(header string, input scanRunningViewInput, items []scanItem) string {
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, header)
	for _, item := range items {
		lines = append(lines, renderScanRunningItemLine(input, item))
	}
	return strings.Join(lines, "\n")
}

func renderScanRecognizedSection(colorMode view.ColorMode, matches []scanMatch) string {
	lines := make([]string, 0, len(matches)+1)
	lines = append(lines, i18n.T("cmd.scan.section.recognized", nil))

	items := cloneScanMatches(matches)
	sortScanMatchesByName(items)
	for _, match := range items {
		lines = append(lines, renderScanRecognizedLine(colorMode, match))
	}

	return strings.Join(lines, "\n")
}

func renderScanUnknownSection(colorMode view.ColorMode, files []string) string {
	lines := make([]string, 0, len(files)+1)
	header := fmt.Sprintf("%s (%s)", i18n.T("cmd.scan.section.unknown", nil), i18n.T("cmd.scan.section.unknown_note", nil))
	lines = append(lines, header)

	items := cloneStrings(files)
	sortStringsCaseInsensitive(items)
	for _, name := range items {
		lines = append(lines, renderScanUnknownLine(colorMode, name))
	}

	return strings.Join(lines, "\n")
}

func renderScanUnsureSection(colorMode view.ColorMode, unsure []scanUnsure) string {
	lines := make([]string, 0, len(unsure)+1)
	header := fmt.Sprintf("%s (%s)", i18n.T("cmd.scan.section.unsure", nil), i18n.T("cmd.scan.section.unsure_note", nil))
	lines = append(lines, header)

	items := cloneScanUnsure(unsure)
	sortScanUnsureByPath(items)
	for _, entry := range items {
		lines = append(lines, renderScanUnsureLine(colorMode, entry.Path))
	}

	return strings.Join(lines, "\n")
}

func renderScanRunningItemLine(input scanRunningViewInput, item scanItem) string {
	label := item.FileName
	switch item.Status {
	case scanItemStatusScanning:
		return view.RenderModItemLine(view.ModItemLine{
			Label:        label,
			Status:       view.ModItemStatusSpinning,
			SpinnerFrame: input.spinnerFrame,
		}, input.colorMode)
	case scanItemStatusPending:
		return view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Status: view.ModItemStatusPending,
		}, input.colorMode)
	default:
		return view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Status: view.ModItemStatusPending,
		}, input.colorMode)
	}
}

func renderScanRecognizedLine(colorMode view.ColorMode, match scanMatch) string {
	label := fmt.Sprintf("%s -> %s", match.FileName, renderScanMatchLabel(match))
	return view.RenderModItemLine(view.ModItemLine{
		Label:  label,
		Status: view.ModItemStatusSuccess,
	}, colorMode)
}

func renderScanAddedLine(colorMode view.ColorMode, match scanMatch) string {
	return view.RenderModItemLine(view.ModItemLine{
		Label:  renderScanMatchLabel(match),
		Status: view.ModItemStatusSuccess,
	}, colorMode)
}

func renderScanUnknownLine(colorMode view.ColorMode, fileName string) string {
	return view.RenderModItemLine(view.ModItemLine{
		Label:  fileName,
		Status: view.ModItemStatusError,
	}, colorMode)
}

func renderScanUnsureLine(colorMode view.ColorMode, fileName string) string {
	return fmt.Sprintf("%s %s", scanUnsureIcon(colorMode), fileName)
}

func scanUnsureIcon(colorMode view.ColorMode) string {
	icon := "?"
	if view.SupportsUnicode() {
		icon = "\u2754"
	}
	return view.RenderIfColorEnabled(colorMode, view.QuestionStyle, icon)
}

func renderScanMatchLabel(match scanMatch) string {
	return fmt.Sprintf("%s (%s) [%s]", match.Name, match.ProjectID, strings.ToLower(string(match.Platform)))
}

func sortScanItemsByFile(items []scanItem) {
	sort.SliceStable(items, func(index int, other int) bool {
		left := strings.ToLower(items[index].FileName)
		right := strings.ToLower(items[other].FileName)
		if left == right {
			return items[index].FileName < items[other].FileName
		}
		return left < right
	})
}

func sortScanCandidatesByFile(candidates []scanCandidate) {
	sort.SliceStable(candidates, func(index int, other int) bool {
		left := strings.ToLower(candidates[index].FileName)
		right := strings.ToLower(candidates[other].FileName)
		if left == right {
			return candidates[index].FileName < candidates[other].FileName
		}
		return left < right
	})
}

func sortScanMatchesByName(items []scanMatch) {
	sort.SliceStable(items, func(index int, other int) bool {
		left := strings.ToLower(items[index].Name)
		right := strings.ToLower(items[other].Name)
		if left != right {
			return left < right
		}
		if items[index].Platform != items[other].Platform {
			return items[index].Platform < items[other].Platform
		}
		if items[index].ProjectID != items[other].ProjectID {
			return items[index].ProjectID < items[other].ProjectID
		}
		return items[index].FileName < items[other].FileName
	})
}

func sortScanMatchesByFile(items []scanMatch) {
	sort.SliceStable(items, func(index int, other int) bool {
		left := strings.ToLower(items[index].FileName)
		right := strings.ToLower(items[other].FileName)
		if left == right {
			return items[index].FileName < items[other].FileName
		}
		return left < right
	})
}

func sortScanUnsureByPath(items []scanUnsure) {
	sort.SliceStable(items, func(index int, other int) bool {
		left := strings.ToLower(items[index].Path)
		right := strings.ToLower(items[other].Path)
		if left == right {
			return items[index].Path < items[other].Path
		}
		return left < right
	})
}

func sortStringsCaseInsensitive(items []string) {
	sort.SliceStable(items, func(index int, other int) bool {
		left := strings.ToLower(items[index])
		right := strings.ToLower(items[other])
		if left == right {
			return items[index] < items[other]
		}
		return left < right
	})
}

func filterScanItems(items []scanItem, statuses ...scanItemStatus) []scanItem {
	if len(statuses) == 0 {
		return nil
	}
	filtered := make([]scanItem, 0, len(items))
	for _, item := range items {
		if scanItemHasStatus(item.Status, statuses) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func scanItemHasStatus(status scanItemStatus, statuses []scanItemStatus) bool {
	for _, candidate := range statuses {
		if status == candidate {
			return true
		}
	}
	return false
}

func matchesFromItems(items []scanItem) []scanMatch {
	matches := make([]scanMatch, 0, len(items))
	for _, item := range items {
		if item.Status == scanItemStatusRecognized {
			matches = append(matches, item.Match)
		}
	}
	return matches
}

func fileNamesFromItems(items []scanItem) []string {
	files := make([]string, 0, len(items))
	for _, item := range items {
		files = append(files, item.FileName)
	}
	return files
}

func unsureFromItems(items []scanItem) []scanUnsure {
	unsure := make([]scanUnsure, 0, len(items))
	for _, item := range items {
		unsure = append(unsure, scanUnsure{Path: item.FileName})
	}
	return unsure
}

func cloneScanMatches(items []scanMatch) []scanMatch {
	cloned := make([]scanMatch, len(items))
	copy(cloned, items)
	return cloned
}

func cloneScanUnsure(items []scanUnsure) []scanUnsure {
	cloned := make([]scanUnsure, len(items))
	copy(cloned, items)
	return cloned
}

func cloneStrings(items []string) []string {
	cloned := make([]string, len(items))
	copy(cloned, items)
	return cloned
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

func renderScanTranscriptLine(colorMode view.ColorMode, item scanItem) string {
	switch item.Status {
	case scanItemStatusRecognized:
		return renderScanRecognizedLine(colorMode, item.Match)
	case scanItemStatusUnknown:
		return renderScanUnknownLine(colorMode, item.FileName)
	case scanItemStatusUnsure:
		return renderScanUnsureLine(colorMode, item.FileName)
	default:
		return ""
	}
}
