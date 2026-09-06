package stories_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/internal/view/stories"
)

func TestProgressRendersAStyledModRow(t *testing.T) {
	setColorProfile(t, termenv.ANSI256)
	setUnicodeSupport(t, true)
	story := stories.NewProgress(view.ColorEnabled, 50)

	rendered := story.View()

	assert.Contains(t, ansi.Strip(rendered), "Sodium (AANobbMI) [modrinth]")
	assert.Contains(t, ansi.Strip(rendered), "50% (50 MB / 100 MB)")
	assert.Contains(t, rendered, view.QuestionStyle.Render("⬇️"))
	assert.Contains(t, rendered, view.ParenStyle.Render("AANobbMI"))
	assert.Contains(t, rendered, view.HelpStyle.Render("Use left and right arrows to adjust progress."))
}

func TestProgressDisablesStylesWhenColorIsDisabled(t *testing.T) {
	setColorProfile(t, termenv.ANSI256)
	setUnicodeSupport(t, true)
	story := stories.NewProgress(view.ColorDisabled, 50)

	rendered := story.View()
	assert.NotContains(t, rendered, view.QuestionStyle.Render("⬇️"))
	assert.NotContains(t, rendered, view.ParenStyle.Render("AANobbMI"))
	assert.NotContains(t, rendered, view.HelpStyle.Render("Use left and right arrows to adjust progress."))
}

func TestProgressAdjustsWithinBounds(t *testing.T) {
	testCases := []struct {
		name            string
		initialPercent  int
		key             tea.KeyType
		expectedPercent string
	}{
		{name: "increases", initialPercent: 50, key: tea.KeyRight, expectedPercent: "60%"},
		{name: "decreases", initialPercent: 50, key: tea.KeyLeft, expectedPercent: "40%"},
		{name: "stops at zero", initialPercent: 0, key: tea.KeyLeft, expectedPercent: "0%"},
		{name: "stops at one hundred", initialPercent: 100, key: tea.KeyRight, expectedPercent: "100%"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			story := stories.NewProgress(view.ColorDisabled, testCase.initialPercent)

			updated, command := story.Update(tea.KeyMsg{Type: testCase.key})

			assert.Contains(t, updated.View(), testCase.expectedPercent)
			assert.Nil(t, command)
		})
	}
}

func TestProgressIgnoresOtherMessages(t *testing.T) {
	story := stories.NewProgress(view.ColorDisabled, 50)

	assert.Nil(t, story.Init())

	updated, command := story.Update(struct{}{})
	assert.Contains(t, updated.View(), "50%")
	assert.Nil(t, command)

	updated, command = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Contains(t, updated.View(), "50%")
	assert.Nil(t, command)
}
