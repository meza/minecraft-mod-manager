package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
)

func Accept() key.Binding {
	return key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp(i18n.T("key.enter"), i18n.T("key.help.accept")),
	)
}

func ApplyFilter() key.Binding {
	return key.NewBinding(
		key.WithKeys("enter", "tab", "shift_tab", "ctrl+k", "up", "ctrl+j", "down"),
		key.WithHelp(i18n.T("key.enter"), i18n.T("key.help.apply_filter")),
	)
}

func Cancel() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp(i18n.T("key.esc"), i18n.T("key.help.cancel")),
	)
}

func ClearFilter() key.Binding {
	return key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp(i18n.T("key.esc"), i18n.T("key.help.clear_filter")),
	)
}

func Complete() key.Binding {
	return key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp(i18n.T("key.tab"), i18n.T("key.help.complete")),
	)
}

func CursorDown() key.Binding {
	return key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", i18n.T("key.help.down")),
	)
}

func CursorUp() key.Binding {
	return key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", i18n.T("key.help.up")),
	)
}

func Filter() key.Binding {
	return key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", i18n.T("key.help.filter")),
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
		key.WithHelp(fmt.Sprintf("%s/%s", "G", i18n.T("key.end")), i18n.T("key.help.go_to_end")),
	)
}

func GoToStart() key.Binding {
	return key.NewBinding(
		key.WithKeys("home", "g"),
		key.WithHelp(fmt.Sprintf("%s/%s", "g", i18n.T("key.home")), i18n.T("key.help.go_to_start")),
	)
}

func HelpMore() key.Binding {
	return key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", i18n.T("key.help.more")),
	)
}

func HelpMoreClose() key.Binding {
	return key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", i18n.T("key.help.close_help")),
	)
}

func NextPage() key.Binding {
	return key.NewBinding(
		key.WithKeys("right", "l", i18n.T("key.pgdown"), "f", "d"),
		key.WithHelp(fmt.Sprintf("%s/%s/%s", "→", "l", i18n.T("key.pgdown")), i18n.T("key.help.page_next")),
	)
}

func PreviousPage() key.Binding {
	return key.NewBinding(
		key.WithKeys("left", "h", i18n.T("key.pgup"), "b", "u"),
		key.WithHelp(fmt.Sprintf("%s/%s/%s", "←", "h", i18n.T("key.pgup")), i18n.T("key.help.page_previous")),
	)
}

func Quit() key.Binding {
	return key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q", i18n.T("key.help.quit")),
	)
}

func QuitWithEsc() key.Binding {
	return key.NewBinding(
		key.WithKeys("ctrl+c/esc"),
		key.WithHelp(fmt.Sprintf("%s/%s", i18n.T("key.ctrl_c"), i18n.T("key.esc")), i18n.T("key.help.quit")),
	)
}

func TranslatedInputKeyBindings() []key.Binding {
	return []key.Binding{Complete(), Accept(), QuitWithEsc()}
}
