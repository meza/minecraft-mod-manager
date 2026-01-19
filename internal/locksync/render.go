package locksync

import (
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func renderLockSyncHeader(colorMode view.ColorMode) string {
	header := i18n.T("cmd.lock_sync.header", nil)
	header = view.RenderIfColorEnabled(colorMode, view.TitleStyle, header)
	subheader := i18n.T("cmd.lock_sync.header.detail", nil)
	return strings.Join([]string{header, subheader}, "\n")
}

func renderLockSyncList(extras []extraLockEntry, colorMode view.ColorMode) string {
	var builder strings.Builder
	header := renderLockSyncHeader(colorMode)
	if err := view.WriteString(&builder, header); err != nil {
		return ""
	}
	if err := view.WriteString(&builder, "\n\n"); err != nil {
		return ""
	}

	for index, entry := range extras {
		if index > 0 {
			if err := view.WriteString(&builder, "\n"); err != nil {
				return ""
			}
		}
		label := view.RenderModLabel(colorMode, entry.DisplayName, entry.Install.ID, entry.Install.Type.String())
		line := view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Suffix: lockSyncEntrySuffix(entry),
			Status: lockSyncEntryStatus(entry),
		}, colorMode)
		if err := view.WriteString(&builder, line); err != nil {
			return ""
		}
	}

	return builder.String()
}

func lockSyncEntrySuffix(entry extraLockEntry) string {
	switch entry.FileStatus {
	case fileStatusMissing:
		return i18n.T("cmd.lock_sync.entry.missing_suffix", &i18n.Tvars{
			Data: &i18n.TData{"file": modfilename.Display(entry.Install.FileName)},
		})
	case fileStatusInvalid:
		return i18n.T("cmd.lock_sync.entry.invalid_suffix", &i18n.Tvars{
			Data: &i18n.TData{"file": modfilename.Display(entry.Install.FileName)},
		})
	case fileStatusPresent:
		return i18n.T("cmd.lock_sync.entry.present_suffix", &i18n.Tvars{
			Data: &i18n.TData{"file": modfilename.Display(entry.Install.FileName)},
		})
	default:
		return ""
	}
}

func lockSyncEntryStatus(entry extraLockEntry) view.ModItemStatus {
	switch entry.FileStatus {
	case fileStatusInvalid:
		return view.ModItemStatusError
	default:
		return view.ModItemStatusQuestion
	}
}

func lockSyncActionLine(policy Policy) string {
	label, ok := lockSyncAnswerLabel(policy)
	if !ok {
		return i18n.T("cmd.lock_sync.action.applied", &i18n.Tvars{
			Data: &i18n.TData{"action": i18n.T("cmd.lock_sync.answer.skip", nil)},
		})
	}
	return i18n.T("cmd.lock_sync.action.applied", &i18n.Tvars{
		Data: &i18n.TData{"action": label},
	})
}

func lockSyncResolutionLines(commandName string, colorMode view.ColorMode, policy Policy) []string {
	result := lockSyncResultLine(colorMode, policy)
	if strings.TrimSpace(result) == "" {
		return nil
	}
	lines := []string{result}
	if strings.TrimSpace(commandName) != "" {
		lines = append(lines, i18n.T("cmd.lock_sync.continue", &i18n.Tvars{
			Data: &i18n.TData{"command": commandName},
		}))
	}
	lines = append(lines, lockSyncDividerLine())
	return lines
}

func lockSyncResultLine(colorMode view.ColorMode, policy Policy) string {
	switch policy {
	case PolicyAdd:
		return fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), i18n.T("cmd.lock_sync.result.add", nil))
	case PolicyDelete:
		return fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), i18n.T("cmd.lock_sync.result.delete", nil))
	case PolicyIgnore:
		return fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), i18n.T("cmd.lock_sync.result.ignore", nil))
	case PolicySkip:
		return fmt.Sprintf("%s %s", view.PendingIcon(colorMode), i18n.T("cmd.common.no_changes", nil))
	default:
		return ""
	}
}

func lockSyncDividerLine() string {
	return strings.Repeat("-", 61)
}
