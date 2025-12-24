package add

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"go.opentelemetry.io/otel/attribute"
)

type addTUIState int

const (
	addTUIStateUnknownPlatformSelect addTUIState = iota
	addTUIStateModNotFoundConfirm
	addTUIStateModNotFoundSelectPlatform
	addTUIStateModNotFoundEnterProjectID
	addTUIStateNoFileConfirm
	addTUIStateNoFileEnterProjectID
	addTUIStateFatalError
	addTUIStateDone
	addTUIStateAborted
)

type addTUIFetchCmd func(platform models.Platform, projectID string) tea.Cmd

type addTUIFetchResultMsg struct {
	platform  models.Platform
	projectID string
	remote    platform.RemoteMod
	err       error
}

type addTUIHistory struct {
	state             addTUIState
	candidatePlatform models.Platform
	candidateProject  string
}

type addTUIModel struct {
	state addTUIState

	width int

	ctx context.Context

	sessionSpan *perf.Span

	cfg models.ModsJSON

	failurePlatform models.Platform
	failureProject  string

	candidatePlatform models.Platform
	candidateProject  string

	history []addTUIHistory

	list list.Model

	confirmMessage string
	confirmDefault bool

	input textinput.Model

	fetchCmd addTUIFetchCmd

	remoteMod        platform.RemoteMod
	resolvedPlatform models.Platform
	resolvedProject  string
	err              error

	fetchSpan *perf.Span
	waitSpan  *perf.Span
}

type addTUIListItem struct {
	value string
}

func (item addTUIListItem) FilterValue() string { return "" }

type addTUIListDelegate struct{}

func (delegate addTUIListDelegate) Height() int                             { return 1 }
func (delegate addTUIListDelegate) Spacing() int                            { return 0 }
func (delegate addTUIListDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (delegate addTUIListDelegate) Render(w io.Writer, listModel list.Model, itemIndex int, listItem list.Item) {
	item, ok := listItem.(addTUIListItem)
	if !ok {
		return
	}

	itemLine := item.value
	if itemIndex == listModel.Index() {
		if _, err := fmt.Fprint(w, tui.SelectedItemStyle.Render("❯ "+itemLine)); err != nil {
			return
		}
		return
	}

	if _, err := fmt.Fprint(w, tui.ItemStyle.Render(itemLine)); err != nil {
		return
	}
}

func newAddTUIModel(ctx context.Context, sessionSpan *perf.Span, initialState addTUIState, platformValue models.Platform, projectID string, cfg models.ModsJSON, fetchCmd addTUIFetchCmd) addTUIModel {
	model := addTUIModel{
		ctx:               ctx,
		sessionSpan:       sessionSpan,
		state:             initialState,
		width:             80,
		cfg:               cfg,
		failurePlatform:   platformValue,
		failureProject:    projectID,
		candidatePlatform: platformValue,
		candidateProject:  projectID,
		fetchCmd:          fetchCmd,
	}

	model.enterState(initialState)
	return model
}

func (model addTUIModel) Init() tea.Cmd {
	switch model.state {
	case addTUIStateDone, addTUIStateAborted:
		return tea.Quit
	default:
		return nil
	}
}

func (model addTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			model.width = msg.Width
			if model.list.Items() != nil {
				model.list.SetWidth(msg.Width)
			}
		}
		return model, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			model.endWait("abort")
			if model.sessionSpan != nil {
				model.sessionSpan.AddEvent("tui.add.action.abort", perf.WithEventAttributes(attribute.String("state", model.stateName())))
			}
			model.state = addTUIStateAborted
			return model, tea.Quit
		case "esc":
			model.endWait("back")
			if model.sessionSpan != nil {
				model.sessionSpan.AddEvent("tui.add.action.back", perf.WithEventAttributes(attribute.String("state", model.stateName())))
			}
			if model.goBack() {
				return model, nil
			}
			model.state = addTUIStateAborted
			return model, tea.Quit
		}
	case addTUIFetchResultMsg:
		return model.handleFetchResult(msg)
	}

	switch model.state {
	case addTUIStateUnknownPlatformSelect, addTUIStateModNotFoundSelectPlatform:
		return model.updateList(msg)
	case addTUIStateModNotFoundEnterProjectID, addTUIStateNoFileEnterProjectID:
		return model.updateInput(msg)
	case addTUIStateModNotFoundConfirm, addTUIStateNoFileConfirm:
		return model.updateConfirm(msg)
	default:
		return model, nil
	}
}

func (model addTUIModel) View() string {
	switch model.state {
	case addTUIStateDone, addTUIStateAborted:
		return ""
	case addTUIStateUnknownPlatformSelect, addTUIStateModNotFoundSelectPlatform:
		return model.list.View()
	case addTUIStateModNotFoundConfirm, addTUIStateNoFileConfirm:
		return renderConfirm(model.confirmMessage, model.confirmDefault)
	case addTUIStateModNotFoundEnterProjectID, addTUIStateNoFileEnterProjectID:
		return renderInput(model.input)
	case addTUIStateFatalError:
		if model.err == nil {
			return ""
		}
		return tui.ErrorStyle.Render(model.err.Error())
	default:
		return ""
	}
}

func (model *addTUIModel) enterState(state addTUIState) {
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("tui.add.state.enter", perf.WithEventAttributes(
			attribute.String("state", model.stateNameFor(state)),
			attribute.String("failure_platform", string(model.failurePlatform)),
			attribute.String("failure_project", model.failureProject),
		))
	}

	model.startWait(state)

	switch state {
	case addTUIStateUnknownPlatformSelect:
		message := i18n.T("cmd.add.tui.unknown_platform", i18n.Tvars{
			Data: &i18n.TData{"platform": string(model.failurePlatform)},
		})
		model.list = newPlatformListModel(message, "", true, model.width)
	case addTUIStateModNotFoundConfirm:
		model.confirmMessage = i18n.T("cmd.add.tui.mod_not_found", i18n.Tvars{
			Data: &i18n.TData{
				"id":       model.failureProject,
				"platform": model.failurePlatform,
			},
		})
		model.confirmDefault = true
	case addTUIStateModNotFoundSelectPlatform:
		message := i18n.T("cmd.add.tui.choose_platform")
		model.list = newPlatformListModel(message, string(model.failurePlatform), false, model.width)
	case addTUIStateModNotFoundEnterProjectID:
		message := i18n.T("cmd.add.tui.enter_project_id")
		model.input = newProjectIDInputModel(message, model.failureProject)
	case addTUIStateNoFileConfirm:
		message := i18n.T("cmd.add.tui.no_file_found", i18n.Tvars{
			Data: &i18n.TData{
				"name":        model.failureProject,
				"platform":    model.failurePlatform,
				"gameVersion": model.cfg.GameVersion,
				"loader":      model.cfg.Loader,
				"other":       alternatePlatform(model.failurePlatform),
			},
		})
		model.confirmMessage = message
		model.confirmDefault = true
	case addTUIStateNoFileEnterProjectID:
		message := i18n.T("cmd.add.tui.enter_project_id_on", i18n.Tvars{
			Data: &i18n.TData{
				"platform": alternatePlatform(model.failurePlatform),
			},
		})
		model.input = newProjectIDInputModel(message, "")
	case addTUIStateFatalError:
		return
	}
}

func (model addTUIModel) stateName() string {
	return model.stateNameFor(model.state)
}

func (model addTUIModel) stateNameFor(state addTUIState) string {
	switch state {
	case addTUIStateUnknownPlatformSelect:
		return "unknown_platform_select"
	case addTUIStateModNotFoundConfirm:
		return "mod_not_found_confirm"
	case addTUIStateModNotFoundSelectPlatform:
		return "mod_not_found_select_platform"
	case addTUIStateModNotFoundEnterProjectID:
		return "mod_not_found_enter_project_id"
	case addTUIStateNoFileConfirm:
		return "no_file_confirm"
	case addTUIStateNoFileEnterProjectID:
		return "no_file_enter_project_id"
	case addTUIStateFatalError:
		return "fatal_error"
	case addTUIStateDone:
		return "done"
	case addTUIStateAborted:
		return "aborted"
	default:
		return "unknown"
	}
}

func (model *addTUIModel) pushState(state addTUIState) {
	model.history = append(model.history, addTUIHistory{
		state:             state,
		candidatePlatform: model.candidatePlatform,
		candidateProject:  model.candidateProject,
	})
}

func (model *addTUIModel) goBack() bool {
	if len(model.history) == 0 {
		return false
	}
	entry := model.history[len(model.history)-1]
	model.history = model.history[:len(model.history)-1]
	model.state = entry.state
	model.candidatePlatform = entry.candidatePlatform
	model.candidateProject = entry.candidateProject
	model.enterState(model.state)
	return true
}

func (model addTUIModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			item, ok := model.list.SelectedItem().(addTUIListItem)
			if !ok {
				return model, nil
			}

			switch model.state {
			case addTUIStateUnknownPlatformSelect:
				if item.value == "cancel" {
					model.endWait("cancel")
					if model.sessionSpan != nil {
						model.sessionSpan.AddEvent("tui.add.action.cancel", perf.WithEventAttributes(attribute.String("state", model.stateName())))
					}
					model.state = addTUIStateAborted
					return model, tea.Quit
				}
				model.endWait("select_platform")
				if model.sessionSpan != nil {
					model.sessionSpan.AddEvent("tui.add.action.select_platform", perf.WithEventAttributes(
						attribute.String("state", model.stateName()),
						attribute.String("platform", item.value),
					))
				}
				model.candidatePlatform = models.Platform(item.value)
				model.candidateProject = model.failureProject
				model.beginFetch("select_platform", model.candidatePlatform, model.candidateProject)
				return model, model.fetchCmd(model.candidatePlatform, model.candidateProject)
			case addTUIStateModNotFoundSelectPlatform:
				model.endWait("select_platform")
				if model.sessionSpan != nil {
					model.sessionSpan.AddEvent("tui.add.action.select_platform", perf.WithEventAttributes(
						attribute.String("state", model.stateName()),
						attribute.String("platform", item.value),
					))
				}
				model.candidatePlatform = models.Platform(item.value)
				model.pushState(addTUIStateModNotFoundSelectPlatform)
				model.state = addTUIStateModNotFoundEnterProjectID
				model.enterState(model.state)
				return model, nil
			}
		}
	}

	var cmd tea.Cmd
	model.list, cmd = model.list.Update(msg)
	return model, cmd
}

func (model addTUIModel) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			value := strings.TrimSpace(model.input.Value())
			if value == "" {
				value = strings.TrimSpace(model.input.Placeholder)
			}
			if value == "" {
				return model, nil
			}
			model.endWait("submit_project_id")
			if model.sessionSpan != nil {
				model.sessionSpan.AddEvent("tui.add.action.submit_project_id", perf.WithEventAttributes(
					attribute.String("state", model.stateName()),
					attribute.String("project_id", value),
				))
			}
			model.candidateProject = value
			if model.state == addTUIStateNoFileEnterProjectID {
				model.candidatePlatform = alternatePlatform(model.failurePlatform)
			}
			model.beginFetch("submit_project_id", model.candidatePlatform, model.candidateProject)
			return model, model.fetchCmd(model.candidatePlatform, model.candidateProject)
		}
	}

	var cmd tea.Cmd
	model.input, cmd = model.input.Update(msg)
	return model, cmd
}

func (model addTUIModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "y", "Y":
			model.endWait("confirm_yes")
			if model.sessionSpan != nil {
				model.sessionSpan.AddEvent("tui.add.action.confirm_yes", perf.WithEventAttributes(attribute.String("state", model.stateName())))
			}
			switch model.state {
			case addTUIStateModNotFoundConfirm:
				model.pushState(addTUIStateModNotFoundConfirm)
				model.state = addTUIStateModNotFoundSelectPlatform
				model.enterState(model.state)
				return model, nil
			case addTUIStateNoFileConfirm:
				model.pushState(addTUIStateNoFileConfirm)
				model.state = addTUIStateNoFileEnterProjectID
				model.enterState(model.state)
				return model, nil
			}
		case "n", "N":
			model.endWait("confirm_no")
			if model.sessionSpan != nil {
				model.sessionSpan.AddEvent("tui.add.action.confirm_no", perf.WithEventAttributes(attribute.String("state", model.stateName())))
			}
			model.state = addTUIStateAborted
			return model, tea.Quit
		}
	}
	return model, nil
}

func (model addTUIModel) handleFetchResult(msg addTUIFetchResultMsg) (tea.Model, tea.Cmd) {
	model.endFetch(msg)

	if msg.err == nil {
		model.remoteMod = msg.remote
		model.resolvedPlatform = msg.platform
		model.resolvedProject = msg.projectID
		if model.sessionSpan != nil {
			model.sessionSpan.AddEvent("tui.add.outcome.resolved", perf.WithEventAttributes(
				attribute.String("platform", string(msg.platform)),
				attribute.String("project_id", msg.projectID),
			))
		}
		model.state = addTUIStateDone
		return model, tea.Quit
	}

	model.err = msg.err
	model.failurePlatform = msg.platform
	model.failureProject = msg.projectID
	model.candidatePlatform = msg.platform
	model.candidateProject = msg.projectID
	model.history = nil

	var unknownPlatformError *platform.UnknownPlatformError
	if errors.As(msg.err, &unknownPlatformError) {
		model.state = addTUIStateUnknownPlatformSelect
		model.enterState(model.state)
		return model, nil
	}

	var modNotFoundError *platform.ModNotFoundError
	if errors.As(msg.err, &modNotFoundError) {
		model.state = addTUIStateModNotFoundConfirm
		model.enterState(model.state)
		return model, nil
	}

	var noCompatibleFileError *platform.NoCompatibleFileError
	if errors.As(msg.err, &noCompatibleFileError) {
		model.state = addTUIStateNoFileConfirm
		model.enterState(model.state)
		return model, nil
	}

	model.state = addTUIStateFatalError
	return model, tea.Quit
}

func (model *addTUIModel) startWait(state addTUIState) {
	model.endWait("state_change")

	stateName := model.stateNameFor(state)
	if stateName == "done" || stateName == "aborted" {
		return
	}

	_, model.waitSpan = perf.StartSpan(model.ctx, "tui.add.wait."+stateName, perf.WithAttributes(attribute.String("state", stateName)))
}

func (model *addTUIModel) endWait(action string) {
	if model.waitSpan == nil {
		return
	}
	model.waitSpan.SetAttributes(
		attribute.String("state", model.stateName()),
		attribute.String("action", action),
	)
	model.waitSpan.End()
	model.waitSpan = nil
}

func (model *addTUIModel) beginFetch(action string, platformValue models.Platform, projectID string) {
	model.endWait(action)

	if model.fetchSpan != nil {
		model.fetchSpan.SetAttributes(
			attribute.Bool("success", false),
			attribute.String("error_type", "overlapping_fetch"),
		)
		model.fetchSpan.End()
		model.fetchSpan = nil
	}

	_, model.fetchSpan = perf.StartSpan(model.ctx, "tui.add.fetch",
		perf.WithAttributes(
			attribute.String("action", action),
			attribute.String("platform", string(platformValue)),
			attribute.String("project_id", projectID),
			attribute.String("state", model.stateName()),
		),
	)
}

func (model *addTUIModel) endFetch(msg addTUIFetchResultMsg) {
	if model.fetchSpan == nil {
		return
	}

	model.fetchSpan.SetAttributes(attribute.Bool("success", msg.err == nil))
	if msg.err != nil {
		model.fetchSpan.SetAttributes(attribute.String("error_type", fmt.Sprintf("%T", msg.err)))
	}
	model.fetchSpan.End()
	model.fetchSpan = nil
}

func newPlatformListModel(message string, defaultValue string, includeCancel bool, width int) list.Model {
	items := []list.Item{
		addTUIListItem{value: string(models.CURSEFORGE)},
		addTUIListItem{value: string(models.MODRINTH)},
	}
	if includeCancel {
		items = append(items, addTUIListItem{value: "cancel"})
	}

	model := list.New(items, addTUIListDelegate{}, width, len(items)+3)
	model.Title = tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(message)
	model.SetShowStatusBar(false)
	model.SetFilteringEnabled(false)
	model.SetShowTitle(true)
	model.SetShowHelp(false)
	model.SetShowPagination(false)
	model.Styles.Title = tui.TitleStyle
	model.Styles.TitleBar = tui.TitleStyle
	model.Styles.PaginationStyle = tui.PaginationStyle
	model.Styles.HelpStyle = tui.HelpStyle
	model.KeyMap = tui.TranslatedListKeyMap()

	for idx, item := range items {
		if candidate, ok := item.(addTUIListItem); ok && candidate.value == defaultValue {
			model.Select(idx)
			break
		}
	}

	return model
}

func newProjectIDInputModel(message string, placeholder string) textinput.Model {
	inputModel := textinput.New()
	inputModel.Prompt = tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(message) + " "
	inputModel.Placeholder = placeholder
	inputModel.PlaceholderStyle = tui.PlaceholderStyle
	width := len(placeholder)
	if width < 10 {
		width = 10
	}
	inputModel.Width = width
	inputModel.Focus()
	return inputModel
}

func renderConfirm(message string, defaultYes bool) string {
	suffix := " (y/N)"
	if defaultYes {
		suffix = " (Y/n)"
	}
	return tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(message) + suffix
}

func renderInput(input textinput.Model) string {
	return input.View()
}

func (model addTUIModel) result() (platform.RemoteMod, models.Platform, string, error) {
	switch model.state {
	case addTUIStateDone:
		if model.remoteMod.FileName == "" {
			return platform.RemoteMod{}, "", "", errors.New("add TUI finished without a mod selection")
		}
		return model.remoteMod, model.resolvedPlatform, model.resolvedProject, nil
	case addTUIStateAborted:
		return platform.RemoteMod{}, "", "", errAborted
	default:
		if model.err != nil {
			return platform.RemoteMod{}, "", "", model.err
		}
		return platform.RemoteMod{}, "", "", errors.New("add TUI did not finish")
	}
}
