package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
)

func Accept() key.Binding {
	return key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp(i18n.T("key.enter", nil), i18n.T("key.help.accept", nil)),
	)
}

func ApplyFilter() key.Binding {
	return key.NewBinding(
		key.WithKeys("enter", "tab", "shift_tab", "ctrl+k", "up", "ctrl+j", "down"),
		key.WithHelp(i18n.T("key.enter", nil), i18n.T("key.help.apply_filter", nil)),
	)
}

func Cancel() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp(i18n.T("key.esc", nil), i18n.T("key.help.cancel", nil)),
	)
}

func ClearFilter() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp(i18n.T("key.esc", nil), i18n.T("key.help.clear_filter", nil)),
	)
}

func Complete() key.Binding {
	return key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp(i18n.T("key.tab", nil), i18n.T("key.help.complete", nil)),
	)
}

func CursorDown() key.Binding {
	return key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", i18n.T("key.help.down", nil)),
	)
}

func CursorUp() key.Binding {
	return key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", i18n.T("key.help.up", nil)),
	)
}

func Filter() key.Binding {
	return key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", i18n.T("key.help.filter", nil)),
	)
}

func ForceQuit() key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+c"),
	)
}

func GoToEnd() key.Binding {
	return key.NewBinding(
		key.WithKeys("end", "G"),
		key.WithHelp(fmt.Sprintf("%s/%s", "G", i18n.T("key.end", nil)), i18n.T("key.help.go_to_end", nil)),
	)
}

func GoToStart() key.Binding {
	return key.NewBinding(
		key.WithKeys("home", "g"),
		key.WithHelp(fmt.Sprintf("%s/%s", "g", i18n.T("key.home", nil)), i18n.T("key.help.go_to_start", nil)),
	)
}

func HelpMore() key.Binding {
	return key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", i18n.T("key.help.more", nil)),
	)
}

func HelpMoreClose() key.Binding {
	return key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", i18n.T("key.help.close_help", nil)),
	)
}

func NextPage() key.Binding {
	return key.NewBinding(
		key.WithKeys("right", "l", "pgdown", "f", "d"),
		key.WithHelp(fmt.Sprintf("%s/%s/%s", "→", "l", i18n.T("key.pgdown", nil)), i18n.T("key.help.page_next", nil)),
	)
}

func PreviousPage() key.Binding {
	return key.NewBinding(
		key.WithKeys("left", "h", "pgup", "b", "u"),
		key.WithHelp(fmt.Sprintf("%s/%s/%s", "←", "h", i18n.T("key.pgup", nil)), i18n.T("key.help.page_previous", nil)),
	)
}

func Quit() key.Binding {
	return key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q", i18n.T("key.help.quit", nil)),
	)
}

func QuitWithEsc() key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+c/esc"),
		key.WithHelp(fmt.Sprintf("%s/%s", i18n.T("key.ctrl_c", nil), i18n.T("key.esc", nil)), i18n.T("key.help.quit", nil)),
	)
}

func TranslatedInputKeyBindings() []key.Binding {
	return []key.Binding{Complete(), Accept(), QuitWithEsc()}
}

func Toggle() key.Binding {
	return key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp(i18n.T("key.space", nil), i18n.T("key.help.toggle", nil)),
	)
}
