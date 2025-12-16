package init

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"go.opentelemetry.io/otel/attribute"
)

type state int

const (
	stateLoader state = iota
	stateGameVersion
	stateReleaseTypes
	stateModsFolder
	done
)

type CommandModel struct {
	state                state
	entered              bool
	ctx                  context.Context
	sessionSpan          *perf.Span
	waitSpan             *perf.Span
	loaderQuestion       LoaderModel
	gameVersionQuestion  GameVersionModel
	releaseTypesQuestion ReleaseTypesModel
	modsFolderQuestion   ModsFolderModel
	result               initOptions
	initialProvided      providedFlags
	err                  error
}

func (m CommandModel) Init() tea.Cmd {
	if m.state == done {
		return tea.Quit
	}
	return nil
}

func (m CommandModel) View() string {
	stringBuilder := strings.Builder{}

	loaderView := ""
	if !m.initialProvided.Loader {
		loaderView = m.loaderQuestion.View()
	}
	gameVersionView := ""
	if !m.initialProvided.GameVersion {
		gameVersionView = m.gameVersionQuestion.View()
	}
	releaseTypesView := ""
	if !m.initialProvided.ReleaseTypes {
		releaseTypesView = m.releaseTypesQuestion.View()
	}
	modsFolderView := ""
	if !m.initialProvided.ModsFolder {
		modsFolderView = m.modsFolderQuestion.View()
	}

	appendSection := func(section string) {
		if section == "" {
			return
		}
		if stringBuilder.Len() > 0 {
			stringBuilder.WriteString("\n")
		}
		stringBuilder.WriteString(section)
	}

	appendSection(loaderView)

	switch m.state {
	case stateLoader:
		return stringBuilder.String()
	case stateGameVersion:
		appendSection(gameVersionView)
	case stateReleaseTypes:
		appendSection(gameVersionView)
		appendSection(releaseTypesView)
	case stateModsFolder:
		appendSection(gameVersionView)
		appendSection(releaseTypesView)
		appendSection(modsFolderView)
	case done:
		appendSection(gameVersionView)
		appendSection(releaseTypesView)
		appendSection(modsFolderView)
	}

	return stringBuilder.String()

}

func (m CommandModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case LoaderSelectedMessage:
		m.endWait("select_loader")
		if m.sessionSpan != nil {
			m.sessionSpan.AddEvent("tui.init.action.select_loader", perf.WithEventAttributes(attribute.String("loader", msg.Loader.String())))
		}
		m.result.Loader = msg.Loader
		m.result.Provided.Loader = true
		m.setState(nextMissingState(m.result))
	case GameVersionSelectedMessage:
		m.endWait("select_game_version")
		if m.sessionSpan != nil {
			m.sessionSpan.AddEvent("tui.init.action.select_game_version", perf.WithEventAttributes(attribute.String("game_version", msg.GameVersion)))
		}
		m.result.GameVersion = msg.GameVersion
		m.result.Provided.GameVersion = true
		m.setState(nextMissingState(m.result))
	case ReleaseTypesSelectedMessage:
		m.endWait("select_release_types")
		if m.sessionSpan != nil {
			m.sessionSpan.AddEvent("tui.init.action.select_release_types", perf.WithEventAttributes(attribute.Int("count", len(msg.ReleaseTypes))))
		}
		m.result.ReleaseTypes = msg.ReleaseTypes
		m.result.Provided.ReleaseTypes = true
		m.setState(nextMissingState(m.result))
	case ModsFolderSelectedMessage:
		m.endWait("select_mods_folder")
		if m.sessionSpan != nil {
			m.sessionSpan.AddEvent("tui.init.action.select_mods_folder", perf.WithEventAttributes(attribute.String("mods_folder", msg.ModsFolder)))
		}
		m.result.ModsFolder = msg.ModsFolder
		m.result.Provided.ModsFolder = true
		m.setState(nextMissingState(m.result))
		if m.state == done {
			if m.sessionSpan != nil {
				m.sessionSpan.AddEvent("tui.init.outcome.completed")
			}
			return m, tea.Quit
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.endWait("abort")
			if m.sessionSpan != nil {
				m.sessionSpan.AddEvent("tui.init.action.abort", perf.WithEventAttributes(attribute.String("state", m.stateName())))
			}
			m.err = fmt.Errorf("init cancelled")
			cmds = append(cmds, tea.Quit)
		}
	}

	switch m.state {
	case stateLoader:
		m.loaderQuestion, cmd = m.loaderQuestion.Update(msg)
	case stateGameVersion:
		m.gameVersionQuestion, cmd = m.gameVersionQuestion.Update(msg)
	case stateReleaseTypes:
		m.releaseTypesQuestion, cmd = m.releaseTypesQuestion.Update(msg)
	case stateModsFolder:
		m.modsFolderQuestion, cmd = m.modsFolderQuestion.Update(msg)
	default:
		return m, tea.Quit
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func NewModel(ctx context.Context, sessionSpan *perf.Span, options initOptions, deps initDeps, meta config.Metadata) *CommandModel {
	defaultReleaseTypes := options.ReleaseTypes
	if !options.Provided.ReleaseTypes {
		defaultReleaseTypes = []models.ReleaseType{models.Release}
	}

	model := &CommandModel{
		ctx:                  ctx,
		sessionSpan:          sessionSpan,
		loaderQuestion:       NewLoaderModel(options.Loader.String()),
		gameVersionQuestion:  NewGameVersionModel(ctx, deps.minecraftClient, options.GameVersion),
		releaseTypesQuestion: NewReleaseTypesModel(defaultReleaseTypes),
		modsFolderQuestion:   NewModsFolderModel(options.ModsFolder, meta, deps.fs, options.Provided.ModsFolder),
		result:               options,
		initialProvided:      options.Provided,
	}

	if options.Provided.Loader && options.Loader != "" {
		model.loaderQuestion.Value = options.Loader
	}
	if options.Provided.GameVersion && options.GameVersion != "" {
		model.gameVersionQuestion.Value = options.GameVersion
		model.gameVersionQuestion.input.SetValue(options.GameVersion)
	}
	if options.Provided.ReleaseTypes && len(options.ReleaseTypes) > 0 {
		model.releaseTypesQuestion.Value = options.ReleaseTypes
	}
	if options.Provided.ModsFolder && options.ModsFolder != "" {
		model.modsFolderQuestion.Value = options.ModsFolder
		model.modsFolderQuestion.input.SetValue(options.ModsFolder)
	}

	model.setState(nextMissingState(model.result))

	return model

}

func (m *CommandModel) setState(next state) {
	if m.state == next && m.entered {
		return
	}
	m.state = next
	m.entered = true
	if m.sessionSpan != nil {
		m.sessionSpan.AddEvent("tui.init.state.enter", perf.WithEventAttributes(attribute.String("state", m.stateName())))
	}

	m.startWait()
}

func (m CommandModel) stateName() string {
	switch m.state {
	case stateLoader:
		return "loader"
	case stateGameVersion:
		return "game_version"
	case stateReleaseTypes:
		return "release_types"
	case stateModsFolder:
		return "mods_folder"
	case done:
		return "done"
	default:
		return "unknown"
	}
}

func (m *CommandModel) startWait() {
	if m.state == done {
		m.waitSpan = nil
		return
	}

	m.endWait("state_change")
	stateName := m.stateName()
	_, m.waitSpan = perf.StartSpan(m.ctx, "tui.init.wait."+stateName, perf.WithAttributes(attribute.String("state", stateName)))
}

func (m *CommandModel) endWait(action string) {
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

func nextMissingState(result initOptions) state {
	if !result.Provided.Loader {
		return stateLoader
	}
	if !result.Provided.GameVersion {
		return stateGameVersion
	}
	if !result.Provided.ReleaseTypes {
		return stateReleaseTypes
	}
	if !result.Provided.ModsFolder {
		return stateModsFolder
	}
	return done
}
