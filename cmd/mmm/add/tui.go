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

	cfg models.ModsJson

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

func (i addTUIListItem) FilterValue() string { return "" }

type addTUIListDelegate struct{}

func (d addTUIListDelegate) Height() int                             { return 1 }
func (d addTUIListDelegate) Spacing() int                            { return 0 }
func (d addTUIListDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d addTUIListDelegate) Render(w io.Writer, m list.Model, itemIndex int, listItem list.Item) {
	item, ok := listItem.(addTUIListItem)
	if !ok {
		return
	}

	itemLine := item.value
	if itemIndex == m.Index() {
		fmt.Fprint(w, tui.SelectedItemStyle.Render("❯ "+itemLine))
		return
	}

	fmt.Fprint(w, tui.ItemStyle.Render(itemLine))
}

func newAddTUIModel(ctx context.Context, sessionSpan *perf.Span, initialState addTUIState, platformValue models.Platform, projectID string, cfg models.ModsJson, fetchCmd addTUIFetchCmd) addTUIModel {
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

func (m addTUIModel) Init() tea.Cmd {
	switch m.state {
	case addTUIStateDone, addTUIStateAborted:
		return tea.Quit
	default:
		return nil
	}
}

func (m addTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
			if m.list.Items() != nil {
				m.list.SetWidth(msg.Width)
			}
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.endWait("abort")
			if m.sessionSpan != nil {
				m.sessionSpan.AddEvent("tui.add.action.abort", perf.WithEventAttributes(attribute.String("state", m.stateName())))
			}
			m.state = addTUIStateAborted
			return m, tea.Quit
		case "esc":
			m.endWait("back")
			if m.sessionSpan != nil {
				m.sessionSpan.AddEvent("tui.add.action.back", perf.WithEventAttributes(attribute.String("state", m.stateName())))
			}
			if m.goBack() {
				return m, nil
			}
			m.state = addTUIStateAborted
			return m, tea.Quit
		}
	case addTUIFetchResultMsg:
		return m.handleFetchResult(msg)
	}

	switch m.state {
	case addTUIStateUnknownPlatformSelect, addTUIStateModNotFoundSelectPlatform:
		return m.updateList(msg)
	case addTUIStateModNotFoundEnterProjectID, addTUIStateNoFileEnterProjectID:
		return m.updateInput(msg)
	case addTUIStateModNotFoundConfirm, addTUIStateNoFileConfirm:
		return m.updateConfirm(msg)
	default:
		return m, nil
	}
}

func (m addTUIModel) View() string {
	switch m.state {
	case addTUIStateDone, addTUIStateAborted:
		return ""
	case addTUIStateUnknownPlatformSelect, addTUIStateModNotFoundSelectPlatform:
		return m.list.View()
	case addTUIStateModNotFoundConfirm, addTUIStateNoFileConfirm:
		return renderConfirm(m.confirmMessage, m.confirmDefault)
	case addTUIStateModNotFoundEnterProjectID, addTUIStateNoFileEnterProjectID:
		return renderInput(m.input)
	case addTUIStateFatalError:
		if m.err == nil {
			return ""
		}
		return tui.ErrorStyle.Render(m.err.Error())
	default:
		return ""
	}
}

func (m *addTUIModel) enterState(state addTUIState) {
	if m.sessionSpan != nil {
		m.sessionSpan.AddEvent("tui.add.state.enter", perf.WithEventAttributes(
			attribute.String("state", m.stateNameFor(state)),
			attribute.String("failure_platform", string(m.failurePlatform)),
			attribute.String("failure_project", m.failureProject),
		))
	}

	m.startWait(state)

	switch state {
	case addTUIStateUnknownPlatformSelect:
		message := i18n.T("cmd.add.tui.unknown_platform", i18n.Tvars{
			Data: &i18n.TData{"platform": string(m.failurePlatform)},
		})
		m.list = newPlatformListModel(message, "", true, m.width)
	case addTUIStateModNotFoundConfirm:
		m.confirmMessage = i18n.T("cmd.add.tui.mod_not_found", i18n.Tvars{
			Data: &i18n.TData{
				"id":       m.failureProject,
				"platform": m.failurePlatform,
			},
		})
		m.confirmDefault = true
	case addTUIStateModNotFoundSelectPlatform:
		message := i18n.T("cmd.add.tui.choose_platform")
		m.list = newPlatformListModel(message, string(m.failurePlatform), false, m.width)
	case addTUIStateModNotFoundEnterProjectID:
		message := i18n.T("cmd.add.tui.enter_project_id")
		m.input = newProjectIDInputModel(message, m.failureProject)
	case addTUIStateNoFileConfirm:
		message := i18n.T("cmd.add.tui.no_file_found", i18n.Tvars{
			Data: &i18n.TData{
				"name":        m.failureProject,
				"platform":    m.failurePlatform,
				"gameVersion": m.cfg.GameVersion,
				"loader":      m.cfg.Loader,
				"other":       alternatePlatform(m.failurePlatform),
			},
		})
		m.confirmMessage = message
		m.confirmDefault = true
	case addTUIStateNoFileEnterProjectID:
		message := i18n.T("cmd.add.tui.enter_project_id_on", i18n.Tvars{
			Data: &i18n.TData{
				"platform": alternatePlatform(m.failurePlatform),
			},
		})
		m.input = newProjectIDInputModel(message, "")
	case addTUIStateFatalError:
		return
	}
}

func (m addTUIModel) stateName() string {
	return m.stateNameFor(m.state)
}

func (m addTUIModel) stateNameFor(state addTUIState) string {
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

func (m *addTUIModel) pushState(state addTUIState) {
	m.history = append(m.history, addTUIHistory{
		state:             state,
		candidatePlatform: m.candidatePlatform,
		candidateProject:  m.candidateProject,
	})
}

func (m *addTUIModel) goBack() bool {
	if len(m.history) == 0 {
		return false
	}
	entry := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	m.state = entry.state
	m.candidatePlatform = entry.candidatePlatform
	m.candidateProject = entry.candidateProject
	m.enterState(m.state)
	return true
}

func (m addTUIModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			item, ok := m.list.SelectedItem().(addTUIListItem)
			if !ok {
				return m, nil
			}

			switch m.state {
			case addTUIStateUnknownPlatformSelect:
				if item.value == "cancel" {
					m.endWait("cancel")
					if m.sessionSpan != nil {
						m.sessionSpan.AddEvent("tui.add.action.cancel", perf.WithEventAttributes(attribute.String("state", m.stateName())))
					}
					m.state = addTUIStateAborted
					return m, tea.Quit
				}
				m.endWait("select_platform")
				if m.sessionSpan != nil {
					m.sessionSpan.AddEvent("tui.add.action.select_platform", perf.WithEventAttributes(
						attribute.String("state", m.stateName()),
						attribute.String("platform", item.value),
					))
				}
				m.candidatePlatform = models.Platform(item.value)
				m.candidateProject = m.failureProject
				m.beginFetch("select_platform", m.candidatePlatform, m.candidateProject)
				return m, m.fetchCmd(m.candidatePlatform, m.candidateProject)
			case addTUIStateModNotFoundSelectPlatform:
				m.endWait("select_platform")
				if m.sessionSpan != nil {
					m.sessionSpan.AddEvent("tui.add.action.select_platform", perf.WithEventAttributes(
						attribute.String("state", m.stateName()),
						attribute.String("platform", item.value),
					))
				}
				m.candidatePlatform = models.Platform(item.value)
				m.pushState(addTUIStateModNotFoundSelectPlatform)
				m.state = addTUIStateModNotFoundEnterProjectID
				m.enterState(m.state)
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m addTUIModel) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				value = strings.TrimSpace(m.input.Placeholder)
			}
			if value == "" {
				return m, nil
			}
			m.endWait("submit_project_id")
			if m.sessionSpan != nil {
				m.sessionSpan.AddEvent("tui.add.action.submit_project_id", perf.WithEventAttributes(
					attribute.String("state", m.stateName()),
					attribute.String("project_id", value),
				))
			}
			m.candidateProject = value
			if m.state == addTUIStateNoFileEnterProjectID {
				m.candidatePlatform = alternatePlatform(m.failurePlatform)
			}
			m.beginFetch("submit_project_id", m.candidatePlatform, m.candidateProject)
			return m, m.fetchCmd(m.candidatePlatform, m.candidateProject)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m addTUIModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "y", "Y":
			m.endWait("confirm_yes")
			if m.sessionSpan != nil {
				m.sessionSpan.AddEvent("tui.add.action.confirm_yes", perf.WithEventAttributes(attribute.String("state", m.stateName())))
			}
			switch m.state {
			case addTUIStateModNotFoundConfirm:
				m.pushState(addTUIStateModNotFoundConfirm)
				m.state = addTUIStateModNotFoundSelectPlatform
				m.enterState(m.state)
				return m, nil
			case addTUIStateNoFileConfirm:
				m.pushState(addTUIStateNoFileConfirm)
				m.state = addTUIStateNoFileEnterProjectID
				m.enterState(m.state)
				return m, nil
			}
		case "n", "N":
			m.endWait("confirm_no")
			if m.sessionSpan != nil {
				m.sessionSpan.AddEvent("tui.add.action.confirm_no", perf.WithEventAttributes(attribute.String("state", m.stateName())))
			}
			m.state = addTUIStateAborted
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m addTUIModel) handleFetchResult(msg addTUIFetchResultMsg) (tea.Model, tea.Cmd) {
	m.endFetch(msg)

	if msg.err == nil {
		m.remoteMod = msg.remote
		m.resolvedPlatform = msg.platform
		m.resolvedProject = msg.projectID
		if m.sessionSpan != nil {
			m.sessionSpan.AddEvent("tui.add.outcome.resolved", perf.WithEventAttributes(
				attribute.String("platform", string(msg.platform)),
				attribute.String("project_id", msg.projectID),
			))
		}
		m.state = addTUIStateDone
		return m, tea.Quit
	}

	m.err = msg.err
	m.failurePlatform = msg.platform
	m.failureProject = msg.projectID
	m.candidatePlatform = msg.platform
	m.candidateProject = msg.projectID
	m.history = nil

	switch msg.err.(type) {
	case *platform.UnknownPlatformError:
		m.state = addTUIStateUnknownPlatformSelect
		m.enterState(m.state)
		return m, nil
	case *platform.ModNotFoundError:
		m.state = addTUIStateModNotFoundConfirm
		m.enterState(m.state)
		return m, nil
	case *platform.NoCompatibleFileError:
		m.state = addTUIStateNoFileConfirm
		m.enterState(m.state)
		return m, nil
	default:
		m.state = addTUIStateFatalError
		return m, tea.Quit
	}
}

func (m *addTUIModel) startWait(state addTUIState) {
	m.endWait("state_change")

	stateName := m.stateNameFor(state)
	if stateName == "done" || stateName == "aborted" {
		return
	}

	_, m.waitSpan = perf.StartSpan(m.ctx, "tui.add.wait."+stateName, perf.WithAttributes(attribute.String("state", stateName)))
}

func (m *addTUIModel) endWait(action string) {
	if m.waitSpan == nil {
		return
	}
	m.waitSpan.SetAttributes(
		attribute.String("state", m.stateName()),
		attribute.String("action", action),
	)
	m.waitSpan.End()
	m.waitSpan = nil
}

func (m *addTUIModel) beginFetch(action string, platformValue models.Platform, projectID string) {
	m.endWait(action)

	if m.fetchSpan != nil {
		m.fetchSpan.SetAttributes(
			attribute.Bool("success", false),
			attribute.String("error_type", "overlapping_fetch"),
		)
		m.fetchSpan.End()
		m.fetchSpan = nil
	}

	_, m.fetchSpan = perf.StartSpan(m.ctx, "tui.add.fetch",
		perf.WithAttributes(
			attribute.String("action", action),
			attribute.String("platform", string(platformValue)),
			attribute.String("project_id", projectID),
			attribute.String("state", m.stateName()),
		),
	)
}

func (m *addTUIModel) endFetch(msg addTUIFetchResultMsg) {
	if m.fetchSpan == nil {
		return
	}

	m.fetchSpan.SetAttributes(attribute.Bool("success", msg.err == nil))
	if msg.err != nil {
		m.fetchSpan.SetAttributes(attribute.String("error_type", fmt.Sprintf("%T", msg.err)))
	}
	m.fetchSpan.End()
	m.fetchSpan = nil
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
	m := textinput.New()
	m.Prompt = tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(message) + " "
	m.Placeholder = placeholder
	m.PlaceholderStyle = tui.PlaceholderStyle
	width := len(placeholder)
	if width < 10 {
		width = 10
	}
	m.Width = width
	m.Focus()
	return m
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

func (m addTUIModel) result() (platform.RemoteMod, models.Platform, string, error) {
	switch m.state {
	case addTUIStateDone:
		if m.remoteMod.FileName == "" {
			return platform.RemoteMod{}, "", "", errors.New("add TUI finished without a mod selection")
		}
		return m.remoteMod, m.resolvedPlatform, m.resolvedProject, nil
	case addTUIStateAborted:
		return platform.RemoteMod{}, "", "", errAborted
	default:
		if m.err != nil {
			return platform.RemoteMod{}, "", "", m.err
		}
		return platform.RemoteMod{}, "", "", errors.New("add TUI did not finish")
	}
}
