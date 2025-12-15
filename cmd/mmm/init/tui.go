package init

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
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
		m.result.Loader = msg.Loader
		m.result.Provided.Loader = true
		m.state = nextMissingState(m.result)
	case GameVersionSelectedMessage:
		m.result.GameVersion = msg.GameVersion
		m.result.Provided.GameVersion = true
		m.state = nextMissingState(m.result)
	case ReleaseTypesSelectedMessage:
		m.result.ReleaseTypes = msg.ReleaseTypes
		m.result.Provided.ReleaseTypes = true
		m.state = nextMissingState(m.result)
	case ModsFolderSelectedMessage:
		m.result.ModsFolder = msg.ModsFolder
		m.result.Provided.ModsFolder = true
		m.state = nextMissingState(m.result)
		if m.state == done {
			return m, tea.Quit
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
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

func NewModel(options initOptions, deps initDeps, meta config.Metadata) *CommandModel {
	defaultReleaseTypes := options.ReleaseTypes
	if !options.Provided.ReleaseTypes {
		defaultReleaseTypes = []models.ReleaseType{models.Release}
	}

	model := &CommandModel{
		loaderQuestion:       NewLoaderModel(options.Loader.String()),
		gameVersionQuestion:  NewGameVersionModel(deps.minecraftClient, options.GameVersion),
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

	model.state = nextMissingState(model.result)

	return model

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
