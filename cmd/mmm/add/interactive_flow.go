package add

import (
	"errors"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type recoveryReason int

const (
	recoveryReasonNotFound recoveryReason = iota
	recoveryReasonNoCompatible
	recoveryReasonDownloadFailed
)

type recoveryState int

const (
	recoveryStateConfirm recoveryState = iota
	recoveryStateSelectPlatform
	recoveryStateEnterProjectID
	recoveryStateDone
	recoveryStateAborted
)

type recoveryFlowInput struct {
	reason      recoveryReason
	platform    models.Platform
	projectID   string
	loader      string
	gameVersion string
	retryCount  int
	colorMode   view.ColorMode
	in          io.Reader
	out         io.Writer
	runTea      func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
}

type recoveryFlowResult struct {
	platform  models.Platform
	projectID string
}

type recoveryFlowModel struct {
	state            recoveryState
	prompt           confirmPromptModel
	platformList     list.Model
	projectIDPrompt  textInputPromptModel
	reason           recoveryReason
	selectedPlatform models.Platform
	selectedProject  string
	aborted          bool
	declined         bool
	headlineLines    []string
	colorMode        view.ColorMode
}

type platformListItem struct {
	label    string
	platform models.Platform
}

func (item platformListItem) FilterValue() string { return item.label }

type platformListDelegate struct{}

func (delegate platformListDelegate) Height() int                             { return 1 }
func (delegate platformListDelegate) Spacing() int                            { return 0 }
func (delegate platformListDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (delegate platformListDelegate) Render(writer io.Writer, listModel list.Model, itemIndex int, listItem list.Item) {
	item, ok := listItem.(platformListItem)
	if !ok {
		return
	}

	pointer := listPointer()
	value := item.label
	if itemIndex == listModel.Index() {
		if err := view.WriteString(writer, view.SelectedItemStyle.Render(pointer+" "+value)); err != nil {
			return
		}
		return
	}

	if err := view.WriteString(writer, view.ItemStyle.Render(value)); err != nil {
		return
	}
}

func runRecoveryFlow(input recoveryFlowInput) (recoveryFlowResult, error) {
	if input.runTea == nil {
		return recoveryFlowResult{}, errors.New("missing bubble tea runner")
	}
	model := newRecoveryFlowModel(input)
	result, err := input.runTea(model, view.ProgramOptions(input.in, input.out)...)
	if err != nil {
		return recoveryFlowResult{}, err
	}
	return recoveryFlowResultFromModel(result)
}

func newRecoveryFlowModel(input recoveryFlowInput) recoveryFlowModel {
	model := recoveryFlowModel{
		state:            recoveryStateConfirm,
		reason:           input.reason,
		selectedPlatform: input.platform,
		selectedProject:  input.projectID,
		colorMode:        input.colorMode,
	}
	model.headlineLines = buildRecoveryHeadlineLines(input)
	model.prompt = newConfirmPromptModel(i18n.T("cmd.add.prompt.modify_search", nil), func(confirmed bool) tea.Msg {
		return confirmSelectedMessage{confirmed: confirmed}
	})
	model.platformList = newPlatformListModel()
	model.projectIDPrompt = newTextInputPromptModel(i18n.T("cmd.add.prompt.project_id", nil), input.projectID)
	return model
}

func buildRecoveryHeadlineLines(input recoveryFlowInput) []string {
	colorMode := input.colorMode
	var lines []string
	switch input.reason {
	case recoveryReasonNotFound:
		summary := renderFinalErrorLine(colorMode, projectNotFoundSummary(input.platform, input.projectID))
		lines = []string{summary}
	case recoveryReasonNoCompatible:
		summary := renderFinalErrorLine(colorMode, noCompatibleSummary(input.loader, input.gameVersion))
		lines = []string{summary}
	case recoveryReasonDownloadFailed:
		summary := renderFinalErrorLine(colorMode, downloadFailedSummary(input.platform, input.projectID))
		lines = []string{summary, "", downloadFailedDetails(input.platform, input.retryCount)}
	default:
		lines = nil
	}
	return lines
}

func (model recoveryFlowModel) Init() tea.Cmd {
	return nil
}

func (model recoveryFlowModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		skipAbort := model.state == recoveryStateSelectPlatform && model.platformList.SettingFilter()
		switch keyMsg.String() {
		case "ctrl+c":
			model.aborted = true
			model.state = recoveryStateAborted
			return model, tea.Quit
		case "esc":
			if !skipAbort {
				model.aborted = true
				model.state = recoveryStateAborted
				return model, tea.Quit
			}
		}
	}

	if typed, ok := msg.(confirmSelectedMessage); ok {
		if !typed.confirmed {
			model.declined = true
			model.state = recoveryStateDone
			return model, tea.Quit
		}
		model.state = recoveryStateSelectPlatform
		return model, nil
	}

	switch model.state {
	case recoveryStateConfirm:
		updated, cmd := model.prompt.Update(msg)
		model.prompt = updated
		return model, cmd
	case recoveryStateSelectPlatform:
		return model.updatePlatformList(msg)
	case recoveryStateEnterProjectID:
		return model.updateProjectID(msg)
	default:
		return model, nil
	}
}

func (model recoveryFlowModel) View() string {
	switch model.state {
	case recoveryStateConfirm:
		return renderRecoveryConfirmView(model.headlineLines, model.prompt.View())
	case recoveryStateSelectPlatform:
		return model.platformList.View()
	case recoveryStateEnterProjectID:
		platformLine := renderSelectedPlatformLine(model.selectedPlatform)
		return platformLine + "\n" + model.projectIDPrompt.View()
	default:
		return ""
	}
}

func renderRecoveryConfirmView(headlines []string, promptView string) string {
	if len(headlines) == 0 {
		return promptView
	}
	return strings.Join(headlines, "\n") + "\n" + promptView
}

func renderSelectedPlatformLine(platformValue models.Platform) string {
	question := view.QuestionStyle.Render("? ") + view.TitleStyle.Render(i18n.T("cmd.add.prompt.platform_selected", nil))
	answer := view.SelectedItemStyle.Render(string(platformValue))
	return question + " " + answer
}

func (model recoveryFlowModel) updatePlatformList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		item, ok := model.platformList.SelectedItem().(platformListItem)
		if !ok {
			return model, nil
		}
		model.selectedPlatform = item.platform
		model.state = recoveryStateEnterProjectID
		return model, nil
	}

	var cmd tea.Cmd
	model.platformList, cmd = model.platformList.Update(msg)
	return model, cmd
}

func (model recoveryFlowModel) updateProjectID(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		value := strings.TrimSpace(model.projectIDPrompt.input.Value())
		if value == "" {
			return model, nil
		}
		model.projectIDPrompt.Value = value
		model.selectedProject = value
		model.state = recoveryStateDone
		return model, tea.Quit
	}

	updated, cmd := model.projectIDPrompt.Update(msg)
	model.projectIDPrompt = updated
	return model, cmd
}

func recoveryFlowResultFromModel(result tea.Model) (recoveryFlowResult, error) {
	switch typed := result.(type) {
	case recoveryFlowModel:
		return typed.result()
	case *recoveryFlowModel:
		return typed.result()
	default:
		return recoveryFlowResult{}, errors.New("unexpected recovery flow model")
	}
}

func (model recoveryFlowModel) result() (recoveryFlowResult, error) {
	if model.aborted {
		return recoveryFlowResult{}, errors.New("recovery aborted")
	}
	if model.declined {
		return recoveryFlowResult{}, errors.New("recovery declined")
	}
	if model.selectedProject == "" {
		return recoveryFlowResult{}, errors.New("recovery did not select a project")
	}
	return recoveryFlowResult{platform: model.selectedPlatform, projectID: model.selectedProject}, nil
}

func newPlatformListModel() list.Model {
	items := []list.Item{
		platformListItem{label: string(models.CURSEFORGE), platform: models.CURSEFORGE},
		platformListItem{label: string(models.MODRINTH), platform: models.MODRINTH},
	}
	model := list.New(items, platformListDelegate{}, 80, len(items)+3)
	model.Title = view.QuestionStyle.Render("? ") + view.TitleStyle.Render(i18n.T("cmd.add.prompt.platform_list", nil))
	model.SetFilteringEnabled(true)
	model.SetShowStatusBar(false)
	model.SetShowPagination(false)
	model.SetShowHelp(true)
	model.SetShowTitle(true)
	model.Styles.Title = view.TitleStyle
	model.Styles.TitleBar = view.TitleStyle
	model.Styles.PaginationStyle = view.PaginationStyle
	model.Styles.HelpStyle = view.HelpStyle
	model.KeyMap = view.TranslatedListKeyMap()
	return model
}

func listPointer() string {
	if view.SupportsUnicode() {
		return "\u276F"
	}
	return ">"
}
