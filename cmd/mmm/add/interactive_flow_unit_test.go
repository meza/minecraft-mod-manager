package add

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestListPointerUsesUnicodeWhenAvailable(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restore)

	assert.Equal(t, "\u276F", listPointer())
}

func TestListPointerUsesASCIIWhenUnicodeUnavailable(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	assert.Equal(t, ">", listPointer())
}

func TestBuildRecoveryHeadlineLines(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	colorMode := view.ColorDisabled
	lines := buildRecoveryHeadlineLines(recoveryFlowInput{
		reason:      recoveryReasonNotFound,
		platform:    models.MODRINTH,
		projectID:   "abc",
		colorMode:   colorMode,
		loader:      "fabric",
		gameVersion: "1.20.1",
	})
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "cmd.add.error.not_found")

	lines = buildRecoveryHeadlineLines(recoveryFlowInput{
		reason:      recoveryReasonNoCompatible,
		platform:    models.MODRINTH,
		projectID:   "abc",
		colorMode:   colorMode,
		loader:      "fabric",
		gameVersion: "1.20.1",
	})
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "cmd.add.error.no_compatible")

	lines = buildRecoveryHeadlineLines(recoveryFlowInput{
		reason:      recoveryReasonDownloadFailed,
		platform:    models.CURSEFORGE,
		projectID:   "abc",
		colorMode:   colorMode,
		loader:      "fabric",
		gameVersion: "1.20.1",
		retryCount:  3,
	})
	assert.Len(t, lines, 3)
	assert.Contains(t, lines[0], "cmd.add.error.download_failed")

	lines = buildRecoveryHeadlineLines(recoveryFlowInput{
		reason:    recoveryReason(99),
		colorMode: colorMode,
	})
	assert.Nil(t, lines)
}

func TestPlatformListItemFilterValueReturnsLabel(t *testing.T) {
	item := platformListItem{label: "modrinth", platform: models.MODRINTH}
	assert.Equal(t, "modrinth", item.FilterValue())
}

func TestPlatformListDelegateUpdateReturnsNil(t *testing.T) {
	delegate := platformListDelegate{}
	cmd := delegate.Update(nil, &list.Model{})
	assert.Nil(t, cmd)
}

func TestPlatformListDelegateRenderSelectedAndUnselected(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newPlatformListModel()
	model.Select(0)

	selected := platformListItem{label: "curseforge", platform: models.CURSEFORGE}
	unselected := platformListItem{label: "modrinth", platform: models.MODRINTH}

	var selectedBuffer strings.Builder
	delegate := platformListDelegate{}
	delegate.Render(&selectedBuffer, model, 0, selected)
	assert.Contains(t, selectedBuffer.String(), "curseforge")

	var unselectedBuffer strings.Builder
	delegate.Render(&unselectedBuffer, model, 1, unselected)
	assert.Contains(t, unselectedBuffer.String(), "modrinth")
}

func TestPlatformListDelegateRenderIgnoresUnknownItem(t *testing.T) {
	model := newPlatformListModel()
	var buffer strings.Builder
	delegate := platformListDelegate{}

	delegate.Render(&buffer, model, 0, list.Item(nil))
	assert.Empty(t, buffer.String())
}

func TestPlatformListDelegateRenderHandlesWriteError(t *testing.T) {
	original := view.WriteString
	t.Cleanup(func() { view.WriteString = original })
	view.WriteString = func(io.Writer, string) error { return errors.New("write failed") }

	model := newPlatformListModel()
	model.Select(0)
	delegate := platformListDelegate{}
	var buffer strings.Builder
	delegate.Render(&buffer, model, 0, platformListItem{label: "curseforge", platform: models.CURSEFORGE})
	assert.Empty(t, buffer.String())
}

func TestPlatformListDelegateRenderHandlesWriteErrorUnselected(t *testing.T) {
	original := view.WriteString
	t.Cleanup(func() { view.WriteString = original })
	view.WriteString = func(io.Writer, string) error { return errors.New("write failed") }

	model := newPlatformListModel()
	model.Select(0)
	delegate := platformListDelegate{}
	var buffer strings.Builder
	delegate.Render(&buffer, model, 1, platformListItem{label: "modrinth", platform: models.MODRINTH})
	assert.Empty(t, buffer.String())
}

func TestRenderSelectedPlatformLineIncludesPromptAndAnswer(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	line := renderSelectedPlatformLine(models.MODRINTH)
	assert.Contains(t, line, "cmd.add.prompt.platform_selected")
	assert.Contains(t, line, "modrinth")
}

func TestRecoveryFlowInitReturnsNil(t *testing.T) {
	model := newRecoveryFlowModel(recoveryFlowInput{colorMode: view.ColorDisabled})
	assert.Nil(t, model.Init())
}

func TestRecoveryFlowUpdateCtrlCAborts(t *testing.T) {
	model := recoveryFlowModel{state: recoveryStateConfirm}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	typed := updated.(recoveryFlowModel)
	assert.True(t, typed.aborted)
	assert.Equal(t, recoveryStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestRecoveryFlowUpdateEscAbortsWhenNotFiltering(t *testing.T) {
	model := newRecoveryFlowModel(recoveryFlowInput{colorMode: view.ColorDisabled})
	model.state = recoveryStateSelectPlatform

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	typed := updated.(recoveryFlowModel)
	assert.True(t, typed.aborted)
	assert.Equal(t, recoveryStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestRecoveryFlowUpdateEscSkipsWhenFiltering(t *testing.T) {
	model := newRecoveryFlowModel(recoveryFlowInput{colorMode: view.ColorDisabled})
	model.state = recoveryStateSelectPlatform
	model.platformList.SetFilterState(list.Filtering)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	typed := updated.(recoveryFlowModel)
	assert.False(t, typed.aborted)
	assert.Equal(t, recoveryStateSelectPlatform, typed.state)
	assert.Nil(t, cmd)
}

func TestRecoveryFlowUpdateConfirmTransitions(t *testing.T) {
	model := recoveryFlowModel{state: recoveryStateConfirm}

	updated, _ := model.Update(confirmSelectedMessage{confirmed: true})
	typed := updated.(recoveryFlowModel)
	assert.Equal(t, recoveryStateSelectPlatform, typed.state)

	model = recoveryFlowModel{state: recoveryStateConfirm}
	updated, _ = model.Update(confirmSelectedMessage{confirmed: false})
	typed = updated.(recoveryFlowModel)
	assert.True(t, typed.declined)
	assert.Equal(t, recoveryStateDone, typed.state)
}

func TestRecoveryFlowUpdatePassesThroughPrompt(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newRecoveryFlowModel(recoveryFlowInput{colorMode: view.ColorDisabled})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	typed := updated.(recoveryFlowModel)
	assert.Contains(t, typed.prompt.input.Value(), "y")
}
func TestRenderRecoveryConfirmViewNoHeadlines(t *testing.T) {
	view := renderRecoveryConfirmView(nil, "prompt")
	assert.Equal(t, "prompt", view)
}

func TestRecoveryFlowViewDoneReturnsEmpty(t *testing.T) {
	model := recoveryFlowModel{state: recoveryStateDone}
	assert.Equal(t, "", model.View())
}

func TestUpdatePlatformListSelectsItem(t *testing.T) {
	model := newRecoveryFlowModel(recoveryFlowInput{colorMode: view.ColorDisabled})
	model.state = recoveryStateSelectPlatform
	model.platformList.Select(0)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(recoveryFlowModel)
	assert.Equal(t, recoveryStateEnterProjectID, typed.state)
	assert.Equal(t, models.CURSEFORGE, typed.selectedPlatform)
	assert.Nil(t, cmd)
}

type badPlatformItem struct{}

func (badPlatformItem) FilterValue() string { return "bad" }

func TestUpdatePlatformListSkipsInvalidItem(t *testing.T) {
	model := newRecoveryFlowModel(recoveryFlowInput{colorMode: view.ColorDisabled})
	model.state = recoveryStateSelectPlatform
	model.platformList.SetItems([]list.Item{badPlatformItem{}})
	model.platformList.Select(0)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(recoveryFlowModel)
	assert.Equal(t, recoveryStateSelectPlatform, typed.state)
	assert.Nil(t, cmd)
}

func TestRecoveryFlowUpdateNoopState(t *testing.T) {
	model := recoveryFlowModel{state: recoveryStateDone}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	typed := updated.(recoveryFlowModel)
	assert.Equal(t, recoveryStateDone, typed.state)
	assert.Nil(t, cmd)
}

func TestUpdateProjectIDUpdatesOnNonEnter(t *testing.T) {
	model := newRecoveryFlowModel(recoveryFlowInput{projectID: "abc", colorMode: view.ColorDisabled})
	model.state = recoveryStateEnterProjectID
	model.projectIDPrompt.input.SetValue("")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	typed := updated.(recoveryFlowModel)
	assert.Contains(t, typed.projectIDPrompt.input.Value(), "x")
}

func TestRecoveryFlowUpdateProjectIDIgnoresEmptyValue(t *testing.T) {
	model := newRecoveryFlowModel(recoveryFlowInput{projectID: "abc", colorMode: view.ColorDisabled})
	model.state = recoveryStateEnterProjectID
	model.projectIDPrompt.input.SetValue(strings.Repeat(" ", 3))

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(recoveryFlowModel)
	assert.Equal(t, recoveryStateEnterProjectID, typed.state)
	assert.Nil(t, cmd)
}

func TestRecoveryFlowUpdateProjectIDSetsValue(t *testing.T) {
	model := newRecoveryFlowModel(recoveryFlowInput{projectID: "abc", colorMode: view.ColorDisabled})
	model.state = recoveryStateEnterProjectID
	model.projectIDPrompt.input.SetValue("xyz")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(recoveryFlowModel)
	assert.Equal(t, recoveryStateDone, typed.state)
	assert.Equal(t, "xyz", typed.selectedProject)
	assert.NotNil(t, cmd)
}

func TestRecoveryFlowResultFromModelPointer(t *testing.T) {
	result, err := recoveryFlowResultFromModel(&recoveryFlowModel{
		selectedPlatform: models.MODRINTH,
		selectedProject:  "abc",
	})
	assert.NoError(t, err)
	assert.Equal(t, models.MODRINTH, result.platform)
	assert.Equal(t, "abc", result.projectID)
}

func TestNewRecoveryFlowModelWithUnknownReason(t *testing.T) {
	model := newRecoveryFlowModel(recoveryFlowInput{reason: recoveryReason(99), colorMode: view.ColorDisabled})
	assert.Nil(t, model.headlineLines)
}

func TestNewRecoveryFlowModelMessageBuilder(t *testing.T) {
	model := newRecoveryFlowModel(recoveryFlowInput{colorMode: view.ColorDisabled})
	cmd := model.prompt.confirmSelected(true)
	msg := cmd()
	typed, ok := msg.(confirmSelectedMessage)
	assert.True(t, ok)
	assert.True(t, typed.confirmed)
}
