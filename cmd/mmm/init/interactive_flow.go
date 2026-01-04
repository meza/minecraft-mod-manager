package init

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
)

type state int

const (
	stateConfigExists state = iota
	stateConfigPath
	stateLoader
	stateGameVersion
	stateReleaseTypes
	stateModsFolder
	stateConfirmWrite
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
	meta                 config.Metadata
	fs                   afero.Fs
	showConfigExists     bool
	showConfigPath       bool
	showConfirmWrite     bool
	configExistsQuestion confirmPromptModel
	configPathQuestion   ConfigPathModel
	confirmWriteQuestion confirmPromptModel
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
	sections := buildViewSections(model)
	orderedSections := orderedViewSections(model, sections)
	if model.state == done {
		return renderViewSectionsWithTrailingNewline(orderedSections)
	}
	return renderViewSections(orderedSections)
}

func orderedViewSections(model CommandModel, sections viewSections) []string {
	ordered := make([]string, 0, 6)
	if sections.configExists != "" {
		ordered = append(ordered, sections.configExists)
	}
	if model.state == stateConfigExists {
		return ordered
	}

	if sections.configPath != "" {
		ordered = append(ordered, sections.configPath)
	}
	if model.state == stateConfigPath {
		return ordered
	}

	if sections.loader != "" {
		ordered = append(ordered, sections.loader)
	}
	if model.state == stateLoader {
		return ordered
	}

	for _, section := range sections.forState(model.state) {
		if section != "" {
			ordered = append(ordered, section)
		}
	}

	return ordered
}

func renderViewSections(sections []string) string {
	stringBuilder, ok := renderViewSectionsBuilder(sections)
	if !ok {
		return ""
	}
	return stringBuilder.String()
}

func renderViewSectionsWithTrailingNewline(sections []string) string {
	stringBuilder, ok := renderViewSectionsBuilder(sections)
	if !ok {
		return ""
	}
	if stringBuilder.Len() > 0 {
		if err := writeString(stringBuilder, "\n"); err != nil {
			return ""
		}
	}
	return stringBuilder.String()
}

func renderViewSectionsBuilder(sections []string) (*strings.Builder, bool) {
	stringBuilder := &strings.Builder{}

	for _, section := range sections {
		if section == "" {
			continue
		}
		if stringBuilder.Len() > 0 {
			if err := writeString(stringBuilder, "\n"); err != nil {
				return stringBuilder, false
			}
		}
		if err := writeString(stringBuilder, section); err != nil {
			return stringBuilder, false
		}
	}

	return stringBuilder, true
}

type viewSections struct {
	configExists string
	configPath   string
	loader       string
	gameVersion  string
	releaseTypes string
	modsFolder   string
	confirmWrite string
}

func buildViewSections(model CommandModel) viewSections {
	sections := viewSections{}
	if model.showConfigExists {
		sections.configExists = model.configExistsQuestion.View()
	}
	if model.showConfigPath {
		sections.configPath = model.configPathQuestion.View()
	}
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
	if model.showConfirmWrite {
		sections.confirmWrite = model.confirmWriteQuestion.View()
	}
	return sections
}

func releaseTypesSection(model CommandModel) string {
	if model.state == stateReleaseTypes || !model.result.Provided.ReleaseTypes {
		return model.releaseTypesQuestion.View()
	}

	question := view.QuestionStyle.Render("? ") + view.TitleStyle.Render(i18n.T("cmd.init.prompt.release-types.question", nil))
	answer := view.SelectedItemStyle.Render(formatReleaseTypes(model.result.ReleaseTypes))
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

	question := view.QuestionStyle.Render("? ") + view.TitleStyle.Render(i18n.T("cmd.init.prompt.mods-folder.question", nil))
	return question + " " + view.SelectedItemStyle.Render(value)
}

func (sections viewSections) forState(current state) []string {
	switch current {
	case stateGameVersion:
		return []string{sections.gameVersion}
	case stateReleaseTypes:
		return []string{sections.gameVersion, sections.releaseTypes}
	case stateModsFolder:
		return []string{sections.gameVersion, sections.releaseTypes, sections.modsFolder}
	case stateConfirmWrite:
		return []string{sections.gameVersion, sections.releaseTypes, sections.modsFolder, sections.confirmWrite}
	case done:
		return []string{sections.gameVersion, sections.releaseTypes, sections.modsFolder, sections.confirmWrite}
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

func NewModel(ctx context.Context, sessionSpan *perf.Span, options initOptions, deps initDeps, meta config.Metadata, configExists bool) *CommandModel {
	modsFolderPrefill := options.Provided.ModsFolder
	options.Provided = normalizeProvidedFlags(ctx, options, deps, meta)
	defaultReleaseTypes := releaseTypesForOptions(options)
	model := buildCommandModel(ctx, sessionSpan, options, deps, meta, configExists, modsFolderPrefill, defaultReleaseTypes)
	applyProvidedSelections(model, options)
	model.setState(nextMissingState(model))
	return model
}

func normalizeProvidedFlags(ctx context.Context, options initOptions, deps initDeps, meta config.Metadata) providedFlags {
	provided := options.Provided
	if provided.GameVersion && options.GameVersion != "" {
		if err := validateMinecraftVersion(ctx, options.GameVersion, deps.minecraftClient); err != nil {
			provided.GameVersion = false
		}
	}
	if provided.ModsFolder && strings.TrimSpace(options.ModsFolder) != "" {
		if err := validateModsFolderInteractive(deps.fs, meta, options.ModsFolder); err != nil {
			provided.ModsFolder = false
		}
	}
	modsFolderCandidate := strings.TrimSpace(options.ModsFolder)
	if !provided.ModsFolder && provided.Loader && provided.GameVersion && provided.ReleaseTypes && modsFolderCandidate != "" {
		provided.ModsFolder = validateModsFolderInteractive(deps.fs, meta, options.ModsFolder) == nil
	}
	return provided
}

func releaseTypesForOptions(options initOptions) []models.ReleaseType {
	if options.Provided.ReleaseTypes {
		return options.ReleaseTypes
	}
	return []models.ReleaseType{models.Release}
}

func buildCommandModel(ctx context.Context, sessionSpan *perf.Span, options initOptions, deps initDeps, meta config.Metadata, configExists bool, modsFolderPrefill bool, defaultReleaseTypes []models.ReleaseType) *CommandModel {
	return &CommandModel{
		ctx:                  ctx,
		sessionSpan:          sessionSpan,
		meta:                 meta,
		fs:                   deps.fs,
		showConfigExists:     configExists && !options.Force,
		showConfirmWrite:     true,
		configExistsQuestion: newConfigOverwritePromptModel(i18n.T("cmd.init.prompt.config-overwrite.question", &i18n.Tvars{Data: &i18n.TData{"configPath": meta.ConfigPath}})),
		configPathQuestion:   NewConfigPathModel(deps.fs),
		confirmWriteQuestion: newConfirmWritePromptModel(i18n.T("cmd.init.prompt.confirm-write.question", nil)),
		loaderQuestion:       NewLoaderModel(options.Loader.String()),
		gameVersionQuestion:  NewGameVersionModel(ctx, deps.minecraftClient, options.GameVersion),
		releaseTypesQuestion: NewReleaseTypesModel(defaultReleaseTypes),
		modsFolderQuestion: NewModsFolderModel(modsFolderModelInput{
			modsFolder: options.ModsFolder,
			meta:       meta,
			fs:         deps.fs,
			prefill:    modsFolderPrefill,
		}),
		result:          options,
		initialProvided: options.Provided,
	}
}

func applyProvidedSelections(model *CommandModel, options initOptions) {
	if options.Provided.Loader && options.Loader != "" {
		model.loaderQuestion.Value = options.Loader
	}
	if options.Provided.ReleaseTypes && len(options.ReleaseTypes) > 0 {
		model.releaseTypesQuestion.Value = options.ReleaseTypes
	}
}

func (model CommandModel) handleMessage(msg tea.Msg) (CommandModel, tea.Cmd) {
	switch typed := msg.(type) {
	case ConfigOverwriteSelectedMessage:
		return model.handleConfigOverwriteSelected(typed), nil
	case ConfigPathSelectedMessage:
		return model.handleConfigPathSelected(typed), nil
	case LoaderSelectedMessage:
		return model.handleLoaderSelected(typed), nil
	case GameVersionSelectedMessage:
		return model.handleGameVersionSelected(typed), nil
	case ReleaseTypesSelectedMessage:
		return model.handleReleaseTypesSelected(typed), nil
	case ModsFolderSelectedMessage:
		return model.handleModsFolderSelected(typed)
	case ConfirmWriteSelectedMessage:
		return model.handleConfirmWriteSelected(typed)
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" {
			return model.handleAbort(), tea.Quit
		}
	}
	return model, nil
}

func (model CommandModel) updateCurrentState(msg tea.Msg) (CommandModel, tea.Cmd) {
	switch model.state {
	case stateConfigExists:
		updated, cmd := model.configExistsQuestion.Update(msg)
		model.configExistsQuestion = updated
		return model, cmd
	case stateConfigPath:
		updated, cmd := model.configPathQuestion.Update(msg)
		model.configPathQuestion = updated
		return model, cmd
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
	case stateConfirmWrite:
		updated, cmd := model.confirmWriteQuestion.Update(msg)
		model.confirmWriteQuestion = updated
		return model, cmd
	default:
		return model, tea.Quit
	}
}

func (model CommandModel) handleConfigOverwriteSelected(msg ConfigOverwriteSelectedMessage) CommandModel {
	model.endWait("select_config_overwrite")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("interactive.init.action.select_config_overwrite", perf.WithEventAttributes(attribute.Bool("overwrite", msg.Overwrite)))
	}

	if msg.Overwrite {
		model.showConfigPath = false
		model.setState(nextMissingState(&model))
		return model
	}

	model.showConfigPath = true
	model.setState(nextMissingState(&model))
	return model
}

func (model CommandModel) handleConfigPathSelected(msg ConfigPathSelectedMessage) CommandModel {
	model.endWait("select_config_path")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("interactive.init.action.select_config_path", perf.WithEventAttributes(attribute.String("config_path", msg.ConfigPath)))
	}

	model.result.ConfigPath = msg.ConfigPath
	model.meta = config.NewMetadata(msg.ConfigPath)
	model.revalidateModsFolder()
	model.refreshModsFolderModel()
	model.setState(nextMissingState(&model))
	return model
}

func (model CommandModel) handleLoaderSelected(msg LoaderSelectedMessage) CommandModel {
	model.endWait("select_loader")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("interactive.init.action.select_loader", perf.WithEventAttributes(attribute.String("loader", msg.Loader.String())))
	}
	model.result.Loader = msg.Loader
	model.result.Provided.Loader = true
	model.setState(nextMissingState(&model))
	return model
}

func (model CommandModel) handleGameVersionSelected(msg GameVersionSelectedMessage) CommandModel {
	model.endWait("select_game_version")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("interactive.init.action.select_game_version", perf.WithEventAttributes(attribute.String("game_version", msg.GameVersion)))
	}
	model.result.GameVersion = msg.GameVersion
	model.result.Provided.GameVersion = true
	model.setState(nextMissingState(&model))
	return model
}

func (model CommandModel) handleReleaseTypesSelected(msg ReleaseTypesSelectedMessage) CommandModel {
	model.endWait("select_release_types")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("interactive.init.action.select_release_types", perf.WithEventAttributes(attribute.Int("count", len(msg.ReleaseTypes))))
	}
	model.result.ReleaseTypes = msg.ReleaseTypes
	model.result.Provided.ReleaseTypes = true
	model.setState(nextMissingState(&model))
	return model
}

func (model CommandModel) handleModsFolderSelected(msg ModsFolderSelectedMessage) (CommandModel, tea.Cmd) {
	model.endWait("select_mods_folder")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("interactive.init.action.select_mods_folder", perf.WithEventAttributes(attribute.String("mods_folder", msg.ModsFolder)))
	}
	model.result.ModsFolder = msg.ModsFolder
	model.result.Provided.ModsFolder = true
	model.setState(nextMissingState(&model))
	if model.state == done {
		if model.sessionSpan != nil {
			model.sessionSpan.AddEvent("interactive.init.outcome.completed")
		}
		return model, nil
	}
	return model, nil
}

func (model CommandModel) handleConfirmWriteSelected(msg ConfirmWriteSelectedMessage) (CommandModel, tea.Cmd) {
	model.endWait("confirm_write")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("interactive.init.action.confirm_write", perf.WithEventAttributes(attribute.Bool("confirmed", msg.Confirmed)))
	}

	if !msg.Confirmed {
		model.err = ErrInitCanceled
		model.setState(done)
		return model, tea.Quit
	}

	model.setState(done)
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("interactive.init.outcome.completed")
	}
	return model, tea.Quit
}

func (model *CommandModel) refreshModsFolderModel() {
	model.modsFolderQuestion = NewModsFolderModel(modsFolderModelInput{
		modsFolder: model.result.ModsFolder,
		meta:       model.meta,
		fs:         model.fs,
		prefill:    model.result.Provided.ModsFolder,
	})
}

func (model *CommandModel) revalidateModsFolder() {
	if !model.result.Provided.ModsFolder {
		return
	}
	if strings.TrimSpace(model.result.ModsFolder) == "" {
		model.result.Provided.ModsFolder = false
		model.initialProvided.ModsFolder = false
		return
	}
	if err := validateModsFolderInteractive(model.fs, model.meta, model.result.ModsFolder); err != nil {
		model.result.ModsFolder = ""
		model.result.Provided.ModsFolder = false
		model.initialProvided.ModsFolder = false
	}
}

func (model CommandModel) handleAbort() CommandModel {
	model.endWait("abort")
	if model.sessionSpan != nil {
		model.sessionSpan.AddEvent("interactive.init.action.abort", perf.WithEventAttributes(attribute.String("state", model.stateName())))
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
		model.sessionSpan.AddEvent("interactive.init.state.enter", perf.WithEventAttributes(attribute.String("state", model.stateName())))
	}

	model.startWait()
}

func (model CommandModel) stateName() string {
	switch model.state {
	case stateConfigExists:
		return "config_exists"
	case stateConfigPath:
		return "config_path"
	case stateLoader:
		return "loader"
	case stateGameVersion:
		return "game_version"
	case stateReleaseTypes:
		return "release_types"
	case stateModsFolder:
		return "mods_folder"
	case stateConfirmWrite:
		return "confirm_write"
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
	_, model.waitSpan = perf.StartSpan(model.ctx, "interactive.init.wait."+stateName, perf.WithAttributes(attribute.String("state", stateName)))
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

func nextMissingState(model *CommandModel) state {
	if model.showConfigExists && model.configExistsQuestion.Value == "" {
		return stateConfigExists
	}
	if model.showConfigPath && model.configPathQuestion.Value == "" {
		return stateConfigPath
	}
	if !model.result.Provided.Loader {
		return stateLoader
	}
	if !model.result.Provided.GameVersion {
		return stateGameVersion
	}
	if !model.result.Provided.ReleaseTypes {
		return stateReleaseTypes
	}
	if !model.result.Provided.ModsFolder {
		return stateModsFolder
	}
	if model.showConfirmWrite && model.confirmWriteQuestion.Value == "" {
		return stateConfirmWrite
	}
	return done
}
