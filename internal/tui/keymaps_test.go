package tui

import (
	"github.com/gkampitakis/go-snaps/snaps"
	"testing"
)

func TestTranslatedListKeyMap(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	keymap := TranslatedListKeyMap()
	snaps.MatchSnapshot(t, keymap)
}

func TestTranslatedInputKeyMap(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	keymap := TranslatedInputKeyMap{}
	snaps.MatchSnapshot(t, keymap.ShortHelp())
	snaps.MatchSnapshot(t, keymap.FullHelp())
}
