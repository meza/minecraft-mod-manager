package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
)

type TranslatedInputKeyMap struct{}

func (k TranslatedInputKeyMap) ShortHelp() []key.Binding {
	return TranslatedInputKeyBindings()
}

func (k TranslatedInputKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

func TranslatedListKeyMap() list.KeyMap {
	return list.KeyMap{
		// Browsing.
		CursorUp:    CursorUp(),
		CursorDown:  CursorDown(),
		PrevPage:    PreviousPage(),
		NextPage:    NextPage(),
		GoToStart:   GoToStart(),
		GoToEnd:     GoToEnd(),
		Filter:      Filter(),
		ClearFilter: ClearFilter(),
		// Filtering.
		CancelWhileFiltering: Cancel(),
		AcceptWhileFiltering: ApplyFilter(),
		// Toggle help.
		ShowFullHelp:  HelpMore(),
		CloseFullHelp: HelpMoreClose(),
		// Quitting.
		Quit:      Quit(),
		ForceQuit: ForceQuit(),
	}
}
