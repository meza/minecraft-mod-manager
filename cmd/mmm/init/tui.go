package init

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/tui"
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

var writeString = func(builder *strings.Builder, value string) error {
	_, err := builder.WriteString(value)
	return err
}

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

	sections := buildViewSections(model)

	var appendErr error
	appendSection := func(section string) {
		if appendErr != nil || section == "" {
			return
		}
		if stringBuilder.Len() > 0 {
			if err := writeString(&stringBuilder, "\n"); err != nil {
				appendErr = err
				return
			}
		}
		if err := writeString(&stringBuilder, section); err != nil {
			appendErr = err
		}
	}

	appendSection(sections.loader)
	if appendErr != nil {
		return ""
	}

	if model.state == stateLoader {
		return stringBuilder.String()
	}

	for _, section := range sections.forState(model.state) {
		appendSection(section)
	}

	if appendErr != nil {
		return ""
	}

	if model.state == done && stringBuilder.Len() > 0 {
		if err := writeString(&stringBuilder, "\n"); err != nil {
			return ""
		}
	}

	return stringBuilder.String()
}

type viewSections struct {
	loader       string
	gameVersion  string
	releaseTypes string
	modsFolder   string
}

func buildViewSections(model CommandModel) viewSections {
	sections := viewSections{}
	if !model.initialProvided.Loader {
		sections.loader = model.loaderQuestion.View()
	}
	if !model.initialProvided.GameVersion {
		sections.gameVersion = model.gameVersionQuestion.View()
	}
	if !model.initialProvided.ReleaseTypes {
		sections.releaseTypes = releaseTypesSection(model)
	}
	if !model.initialProvided.ModsFolder {
		sections.modsFolder = modsFolderSection(model)
	}
	return sections
}

func releaseTypesSection(model CommandModel) string {
	if model.state == stateReleaseTypes || !model.result.Provided.ReleaseTypes {
		return model.releaseTypesQuestion.View()
	}

	question := tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(i18n.T("cmd.init.tui.release-types.question", nil))
	answer := tui.SelectedItemStyle.Render(formatReleaseTypes(model.result.ReleaseTypes))
	return question + " " + answer
}

func modsFolderSection(model CommandModel) string {
	if model.state == stateModsFolder {
		return model.modsFolderQuestion.View()
	}

	value := strings.TrimSpace(model.result.ModsFolder)
	if value == "" {
		value = strings.TrimSpace(model.modsFolderQuestion.Value)
	}
	if value == "" {
		return model.modsFolderQuestion.View()
	}

	question := tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(i18n.T("cmd.init.tui.mods-folder.question", nil))
	return question + " " + tui.SelectedItemStyle.Render(value)
}

func (sections viewSections) forState(current state) []string {
	switch current {
	case stateGameVersion:
		return []string{sections.gameVersion}
	case stateReleaseTypes:
		return []string{sections.gameVersion, sections.releaseTypes}
	case stateModsFolder, done:
		return []string{sections.gameVersion, sections.releaseTypes, sections.modsFolder}
	default:
		return nil
	}
}

func (model CommandModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	updatedModel, cmd := model.handleMessage(msg)
	model = updatedModel
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	updatedModel, cmd = model.updateCurrentState(msg)
	model = updatedModel
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

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
		modsFolderQuestion: NewModsFolderModel(modsFolderModelInput{
			modsFolder: options.ModsFolder,
			meta:       meta,
			fs:         deps.fs,
			prefill:    options.Provided.ModsFolder,
		}),
		result:          options,
		initialProvided: options.Provided,
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

func (model CommandModel) handleMessage(msg tea.Msg) (CommandModel, tea.Cmd) {
	switch typed := msg.(type) {
	case LoaderSelectedMessage:
		return model.handleLoaderSelected(typed), nil
	case GameVersionSelectedMessage:
		return model.handleGameVersionSelected(typed), nil
	case ReleaseTypesSelectedMessage:
		return model.handleReleaseTypesSelected(typed), nil
	case ModsFolderSelectedMessage:
		return model.handleModsFolderSelected(typed)
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" {
			return model.handleAbort(), tea.Quit
		}
	}
	return model, nil
}

func (model CommandModel) updateCurrentState(msg tea.Msg) (CommandModel, tea.Cmd) {
	switch model.state {
	case stateLoader:
		updated, cmd := model.loaderQuestion.Update(msg)
		model.loaderQuestion = updated
		return model, cmd
	case stateGameVersion:
		updated, cmd := model.gameVersionQuestion.Update(msg)
		model.gameVersionQuestion = updated
		return model, cmd
	case stateReleaseTypes:
		updated, cmd := model.releaseTypesQuestion.Update(msg)
		model.releaseTypesQuestion = updated
		return model, cmd
	case stateModsFolder:
		updated, cmd := model.modsFolderQuestion.Update(msg)
		model.modsFolderQuestion = updated
		return model, cmd
	default:
		return model, tea.Quit
	}
}

func (model CommandModel) handleLoaderSelected(msg LoaderSelectedMessage) CommandModel {
	model.endWait("select_loader")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("tui.init.action.select_loader", perf.WithEventAttributes(attribute.String("loader", msg.Loader.String())))
	}
	model.result.Loader = msg.Loader
	model.result.Provided.Loader = true
	model.setState(nextMissingState(model.result))
	return model
}

func (model CommandModel) handleGameVersionSelected(msg GameVersionSelectedMessage) CommandModel {
	model.endWait("select_game_version")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("tui.init.action.select_game_version", perf.WithEventAttributes(attribute.String("game_version", msg.GameVersion)))
	}
	model.result.GameVersion = msg.GameVersion
	model.result.Provided.GameVersion = true
	model.setState(nextMissingState(model.result))
	return model
}

func (model CommandModel) handleReleaseTypesSelected(msg ReleaseTypesSelectedMessage) CommandModel {
	model.endWait("select_release_types")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("tui.init.action.select_release_types", perf.WithEventAttributes(attribute.Int("count", len(msg.ReleaseTypes))))
	}
	model.result.ReleaseTypes = msg.ReleaseTypes
	model.result.Provided.ReleaseTypes = true
	model.setState(nextMissingState(model.result))
	return model
}

func (model CommandModel) handleModsFolderSelected(msg ModsFolderSelectedMessage) (CommandModel, tea.Cmd) {
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
		return model, nil
	}
	return model, nil
}

func (model CommandModel) handleAbort() CommandModel {
	model.endWait("abort")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("tui.init.action.abort", perf.WithEventAttributes(attribute.String("state", model.stateName())))
	}
	model.err = ErrInitCanceled
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
