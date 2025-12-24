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

func (model CommandModel) Init() tea.Cmd {
	if model.state == done {
		return tea.Quit
	}
	return nil
}

func (model CommandModel) View() string {
	stringBuilder := strings.Builder{}

	loaderView := ""
	if !model.initialProvided.Loader {
		loaderView = model.loaderQuestion.View()
	}
	gameVersionView := ""
	if !model.initialProvided.GameVersion {
		gameVersionView = model.gameVersionQuestion.View()
	}
	releaseTypesView := ""
	if !model.initialProvided.ReleaseTypes {
		releaseTypesView = model.releaseTypesQuestion.View()
	}
	modsFolderView := ""
	if !model.initialProvided.ModsFolder {
		modsFolderView = model.modsFolderQuestion.View()
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

	switch model.state {
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

func (model CommandModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case LoaderSelectedMessage:
		model.endWait("select_loader")
		if model.sessionSpan != nil {
			model.sessionSpan.AddEvent("tui.init.action.select_loader", perf.WithEventAttributes(attribute.String("loader", msg.Loader.String())))
		}
		model.result.Loader = msg.Loader
		model.result.Provided.Loader = true
		model.setState(nextMissingState(model.result))
	case GameVersionSelectedMessage:
		model.endWait("select_game_version")
		if model.sessionSpan != nil {
			model.sessionSpan.AddEvent("tui.init.action.select_game_version", perf.WithEventAttributes(attribute.String("game_version", msg.GameVersion)))
		}
		model.result.GameVersion = msg.GameVersion
		model.result.Provided.GameVersion = true
		model.setState(nextMissingState(model.result))
	case ReleaseTypesSelectedMessage:
		model.endWait("select_release_types")
		if model.sessionSpan != nil {
			model.sessionSpan.AddEvent("tui.init.action.select_release_types", perf.WithEventAttributes(attribute.Int("count", len(msg.ReleaseTypes))))
		}
		model.result.ReleaseTypes = msg.ReleaseTypes
		model.result.Provided.ReleaseTypes = true
		model.setState(nextMissingState(model.result))
	case ModsFolderSelectedMessage:
		model.endWait("select_mods_folder")
		if model.sessionSpan != nil {
			model.sessionSpan.AddEvent("tui.init.action.select_mods_folder", perf.WithEventAttributes(attribute.String("mods_folder", msg.ModsFolder)))
		}
		model.result.ModsFolder = msg.ModsFolder
		model.result.Provided.ModsFolder = true
		model.setState(nextMissingState(model.result))
		if model.state == done {
			if model.sessionSpan != nil {
				model.sessionSpan.AddEvent("tui.init.outcome.completed")
			}
			return model, tea.Quit
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			model.endWait("abort")
			if model.sessionSpan != nil {
				model.sessionSpan.AddEvent("tui.init.action.abort", perf.WithEventAttributes(attribute.String("state", model.stateName())))
			}
			model.err = fmt.Errorf("init canceled")
			cmds = append(cmds, tea.Quit)
		}
	}

	switch model.state {
	case stateLoader:
		model.loaderQuestion, cmd = model.loaderQuestion.Update(msg)
	case stateGameVersion:
		model.gameVersionQuestion, cmd = model.gameVersionQuestion.Update(msg)
	case stateReleaseTypes:
		model.releaseTypesQuestion, cmd = model.releaseTypesQuestion.Update(msg)
	case stateModsFolder:
		model.modsFolderQuestion, cmd = model.modsFolderQuestion.Update(msg)
	default:
		return model, tea.Quit
	}
	cmds = append(cmds, cmd)

	return model, tea.Batch(cmds...)
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

func (model *CommandModel) setState(next state) {
	if model.state == next && model.entered {
		return
	}
	model.state = next
	model.entered = true
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("tui.init.state.enter", perf.WithEventAttributes(attribute.String("state", model.stateName())))
	}

	model.startWait()
}

func (model CommandModel) stateName() string {
	switch model.state {
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

func (model *CommandModel) startWait() {
	if model.state == done {
		model.waitSpan = nil
		return
	}

	model.endWait("state_change")
	stateName := model.stateName()
	_, model.waitSpan = perf.StartSpan(model.ctx, "tui.init.wait."+stateName, perf.WithAttributes(attribute.String("state", stateName)))
}

func (model *CommandModel) endWait(action string) {
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
