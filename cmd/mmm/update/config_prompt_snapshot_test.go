package update

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/muesli/termenv"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestUpdateConfigPromptSnapshots(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	headline := messageWithIcon(view.FinalErrorIcon(view.ColorDisabled), i18n.T("cmd.config.error.missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": "/cfg/modlist.json"},
	}))
	question := i18n.T("cmd.config.confirm_init_short", nil)

	t.Run("initial", func(t *testing.T) {
		model := newConfigInitModel(headline, question)
		snaps.MatchSnapshot(t, model.View())
	})

	t.Run("invalid_choice", func(t *testing.T) {
		model := newConfigInitModel(headline, question)
		model.prompt.input.SetValue("maybe")
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		snaps.MatchSnapshot(t, updated.(configInitModel).View())
	})

	t.Run("default_no", func(t *testing.T) {
		model := newConfigInitModel(headline, question)
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		snaps.MatchSnapshot(t, updated.(configInitModel).View())
	})

	t.Run("confirmed_yes", func(t *testing.T) {
		model := newConfigInitModel(headline, question)
		model.prompt.input.SetValue(model.prompt.yesOption.short)
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		snaps.MatchSnapshot(t, updated.(configInitModel).View())
	})
}
