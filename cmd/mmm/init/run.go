package init

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

func runInitCommand(ctx context.Context, cmd *cobra.Command, options initOptions, deps initDeps, meta config.Metadata) error {
	finalOptions, didUseInteractiveFlow, err := runInit(ctx, cmd, options, deps, meta)
	if errors.Is(err, ErrInitCanceled) {
		err = nil
	}

	if deps.telemetry != nil {
		deps.telemetry(buildTelemetryPayload(finalOptions, didUseInteractiveFlow, err))
	}

	return err
}

func runInit(ctx context.Context, cmd *cobra.Command, options initOptions, deps initDeps, meta config.Metadata) (initOptions, bool, error) {
	executionMode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: options.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})
	options = markModsFolderProvided(executionMode, options)
	useInteractiveFlow := executionMode == interaction.ExecutionModeInteractive
	didUseInteractiveFlow := false

	if !useInteractiveFlow {
		updatedOptions, err := runInitWithoutPrompt(ctx, cmd, options, deps, meta)
		return updatedOptions, didUseInteractiveFlow, err
	}

	configExists, err := configFileExists(deps.fs, meta.ConfigPath)
	if err != nil {
		return options, didUseInteractiveFlow, err
	}

	configState := configMissing
	if configExists {
		configState = configPresent
	}
	if shouldRunWithoutPrompt(configState, options) {
		options, err = runInitWithoutPrompt(ctx, cmd, options, deps, meta)
		return options, didUseInteractiveFlow, err
	}

	options, didUseInteractiveFlow, err = runInitInteractive(ctx, cmd, options, deps, meta, configExists)
	return options, didUseInteractiveFlow, err
}

func markModsFolderProvided(mode interaction.ExecutionMode, options initOptions) initOptions {
	if mode != interaction.ExecutionModeInteractive {
		return options
	}
	if options.Provided.ModsFolder {
		return options
	}
	if !options.Provided.Loader || !options.Provided.GameVersion || !options.Provided.ReleaseTypes {
		return options
	}
	if strings.TrimSpace(options.ModsFolder) == "" {
		return options
	}
	options.Provided.ModsFolder = true
	return options
}

type configExistence int

const (
	configMissing configExistence = iota
	configPresent
)

func shouldRunWithoutPrompt(configState configExistence, options initOptions) bool {
	if configState == configMissing || options.Force {
		return hasAllRequiredInputs(options)
	}
	return false
}

func hasAllRequiredInputs(options initOptions) bool {
	if !options.Provided.Loader || options.Loader == "" {
		return false
	}
	if !options.Provided.GameVersion || options.GameVersion == "" {
		return false
	}
	if !options.Provided.ReleaseTypes || len(options.ReleaseTypes) == 0 {
		return false
	}
	if !options.Provided.ModsFolder || strings.TrimSpace(options.ModsFolder) == "" {
		return false
	}
	return true
}

func runInitWithoutPrompt(ctx context.Context, cmd *cobra.Command, options initOptions, deps initDeps, meta config.Metadata) (initOptions, error) {
	updated, err := validateUnattendedInputs(ctx, cmd, options, deps, meta)
	if err != nil {
		outputErr := writeUnattendedOutput(cmd, deps, err)
		if outputErr != nil {
			return options, outputErr
		}
		if isUnattendedOutputError(err) {
			return options, clierrors.MarkHandled(err)
		}
		return options, err
	}
	options = updated
	if err := initWithDeps(ctx, options, deps); err != nil {
		return options, err
	}
	if err := writeInitSuccess(cmd, deps, options, meta); err != nil {
		return options, err
	}
	return options, nil
}

func runInitInteractive(ctx context.Context, cmd *cobra.Command, options initOptions, deps initDeps, meta config.Metadata, configExists bool) (initOptions, bool, error) {
	options = normalizeGameVersionInteractive(options)

	updated, launched, runErr := runInteractiveInitWithLaunchFlag(ctx, cmd, options, deps, meta, configExists)
	if runErr != nil {
		return options, launched, normalizeInitCancelError(runErr)
	}
	options = updated

	if err := initWithDeps(ctx, options, deps); err != nil {
		return options, launched, err
	}
	updatedMeta := config.NewMetadata(options.ConfigPath)
	if err := writeInitSuccess(cmd, deps, options, updatedMeta); err != nil {
		return options, launched, err
	}
	return options, launched, nil
}

func runInteractiveInitWithLaunchFlag(ctx context.Context, cmd *cobra.Command, options initOptions, deps initDeps, meta config.Metadata, configExists bool) (initOptions, bool, error) {
	sessionCtx, sessionSpan := perf.StartSpan(ctx, "interactive.init.session",
		perf.WithAttributes(
			attribute.Bool("provided_loader", options.Provided.Loader),
			attribute.Bool("provided_game_version", options.Provided.GameVersion),
			attribute.Bool("provided_release_types", options.Provided.ReleaseTypes),
			attribute.Bool("provided_mods_folder", options.Provided.ModsFolder),
		),
	)
	defer sessionSpan.End()

	model := NewModel(sessionCtx, sessionSpan, options, deps, meta, configExists)

	result, err := deps.runTea(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return options, true, err
	}

	finalResult, err := finalizeInteractiveResult(result)
	if err != nil {
		return options, true, err
	}

	return finalResult, true, nil
}

func finalizeInteractiveResult(result tea.Model) (initOptions, error) {
	var finalModel CommandModel
	switch typed := result.(type) {
	case *CommandModel:
		finalModel = *typed
	case CommandModel:
		finalModel = typed
	default:
		return initOptions{}, errors.New(i18n.T("cmd.init.error.interactive.failed", nil))
	}

	if finalModel.err != nil {
		return initOptions{}, finalModel.err
	}

	if finalModel.state != done {
		return initOptions{}, ErrInitCanceled
	}

	return finalModel.result, nil
}

func normalizeInitCancelError(err error) error {
	if errors.Is(err, ErrInitCanceled) {
		return ErrInitCanceled
	}
	return err
}

func validateUnattendedInputs(ctx context.Context, cmd *cobra.Command, options initOptions, deps initDeps, meta config.Metadata) (initOptions, error) {
	if missingRequiredUnattended(options) {
		return options, reportUnattendedMissingRequired(cmd, deps)
	}

	configExists, err := configFileExists(deps.fs, meta.ConfigPath)
	if err != nil {
		return options, err
	}
	if configExists && !options.Force {
		return options, reportUnattendedConfigExists(cmd, deps, meta)
	}

	updated, err := normalizeGameVersion(ctx, options, deps, gameVersionUnattended)
	if err != nil {
		return options, reportUnattendedLatestUnavailable(cmd, deps)
	}
	options = updated

	valid, validationErr := minecraft.IsValidVersion(ctx, options.GameVersion, deps.minecraftClient)
	if validationErr != nil {
		return options, reportUnattendedGameVersionUnavailable(cmd, deps)
	}
	if !valid {
		return options, reportUnattendedInvalidGameVersion(cmd, deps, options.GameVersion)
	}

	if err := validateModsFolderUnattended(deps.fs, meta, options.ModsFolder); err != nil {
		return options, reportUnattendedMissingModsFolder(cmd, deps, err)
	}

	return options, nil
}

func missingRequiredUnattended(options initOptions) bool {
	if options.Loader == "" {
		return true
	}
	if options.GameVersion == "" {
		return true
	}
	return false
}

func validateModsFolderUnattended(fs afero.Fs, meta config.Metadata, modsFolder string) error {
	modsFolder = strings.TrimSpace(modsFolder)
	if modsFolder == "" {
		return &modsFolderValidationError{kind: modsFolderEmpty}
	}

	modsFolderConfig := models.ModsJSON{ModsFolder: modsFolder}
	modsFolderPath := meta.ModsFolderPath(modsFolderConfig)
	modsFolderExists, err := afero.Exists(fs, modsFolderPath)
	if err != nil {
		return err
	}
	if !modsFolderExists {
		return &modsFolderValidationError{path: modsFolderPath, kind: modsFolderMissing}
	}

	isDir, err := afero.IsDir(fs, modsFolderPath)
	if err != nil {
		return err
	}
	if !isDir {
		return &modsFolderValidationError{path: modsFolderPath, kind: modsFolderNotDirectory}
	}

	return nil
}

type modsFolderValidationKind int

const (
	modsFolderEmpty modsFolderValidationKind = iota
	modsFolderMissing
	modsFolderNotDirectory
)

type modsFolderValidationError struct {
	path string
	kind modsFolderValidationKind
}

func (err *modsFolderValidationError) Error() string {
	return err.path
}

func (err *modsFolderValidationError) Kind() modsFolderValidationKind {
	return err.kind
}

func configFileExists(fs afero.Fs, configPath string) (bool, error) {
	exists, err := afero.Exists(fs, configPath)
	if err != nil {
		return false, fmt.Errorf("%s: %w", i18n.T("cmd.init.error.config-file.check", nil), err)
	}
	return exists, nil
}

func reportUnattendedMissingRequired(cmd *cobra.Command, deps initDeps) error {
	return reportUnattendedError(cmd, deps,
		"cmd.init.error.unattended.missing_required",
		nil,
		"cmd.init.error.unattended.missing_required_hint",
	)
}

func reportUnattendedConfigExists(cmd *cobra.Command, deps initDeps, meta config.Metadata) error {
	return reportUnattendedError(cmd, deps,
		"cmd.init.error.unattended.config_exists",
		&i18n.Tvars{Data: &i18n.TData{"configPath": meta.ConfigPath}},
		"cmd.init.error.unattended.config_exists_hint",
	)
}

func reportUnattendedLatestUnavailable(cmd *cobra.Command, deps initDeps) error {
	return reportUnattendedError(cmd, deps,
		"cmd.init.error.unattended.latest_unavailable",
		nil,
		"cmd.init.error.unattended.latest_unavailable_hint",
	)
}

func reportUnattendedGameVersionUnavailable(cmd *cobra.Command, deps initDeps) error {
	return reportUnattendedError(cmd, deps,
		"cmd.init.error.game-version.unavailable",
		nil,
		"",
	)
}

func reportUnattendedInvalidGameVersion(cmd *cobra.Command, deps initDeps, gameVersion string) error {
	return reportUnattendedError(cmd, deps,
		"cmd.init.error.unattended.game-version.invalid",
		&i18n.Tvars{Data: &i18n.TData{"gameVersion": gameVersion}},
		"cmd.init.error.unattended.game-version.invalid_hint",
	)
}

func reportUnattendedMissingModsFolder(cmd *cobra.Command, deps initDeps, err error) error {
	path := ""
	if err != nil {
		path = err.Error()
	}
	modsFolderErr := &modsFolderValidationError{}
	if errors.As(err, &modsFolderErr) && modsFolderErr.Kind() == modsFolderNotDirectory {
		return reportUnattendedError(cmd, deps,
			"cmd.init.error.unattended.mods-folder-not-directory",
			&i18n.Tvars{Data: &i18n.TData{"path": modsFolderErr.path}},
			"cmd.init.error.unattended.mods-folder-not-directory_hint",
		)
	}
	if errors.As(err, &modsFolderErr) && modsFolderErr.Kind() == modsFolderEmpty {
		return reportUnattendedError(cmd, deps,
			"cmd.init.error.unattended.mods-folder.empty",
			nil,
			"cmd.init.error.unattended.mods-folder.empty_hint",
		)
	}

	return reportUnattendedError(cmd, deps,
		"cmd.init.error.unattended.mods-folder-missing",
		&i18n.Tvars{Data: &i18n.TData{"path": path}},
		"cmd.init.error.unattended.mods-folder-missing_hint",
	)
}

func reportUnattendedError(_ *cobra.Command, _ initDeps, messageKey string, messageVars *i18n.Tvars, hintKey string) error {
	return &unattendedOutputError{
		messageKey:  messageKey,
		messageVars: messageVars,
		hintKey:     hintKey,
	}
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

func colorModeForWriter(cmd *cobra.Command) view.ColorMode {
	if cmd == nil {
		return view.ColorDisabled
	}
	if !view.SupportsColor(cmd.OutOrStdout()) {
		return view.ColorDisabled
	}
	return view.ColorEnabled
}

type unattendedOutputError struct {
	messageKey  string
	messageVars *i18n.Tvars
	hintKey     string
}

func (err *unattendedOutputError) Error() string {
	return "init unattended error"
}

func isUnattendedOutputError(err error) bool {
	var outputErr *unattendedOutputError
	return errors.As(err, &outputErr)
}

func writeUnattendedOutput(cmd *cobra.Command, deps initDeps, err error) error {
	var outputErr *unattendedOutputError
	if !errors.As(err, &outputErr) {
		return nil
	}
	colorMode := colorModeForWriter(cmd)
	icon := "!!"
	if colorMode.Enabled() && view.SupportsUnicode() {
		icon = "\u203c\ufe0f"
	}
	headline := messageWithIcon(
		icon,
		i18n.T(outputErr.messageKey, outputErr.messageVars),
	)
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}
	lines := []string{
		headline,
	}
	if outputErr.hintKey != "" {
		hint := i18n.T(outputErr.hintKey, nil)
		if colorMode.Enabled() {
			hint = view.CtaStyle.Render(hint)
		}
		lines = append(lines, hint)
	}
	return runOutputLines(cmd, deps, lines)
}

func writeInitSuccess(cmd *cobra.Command, deps initDeps, options initOptions, meta config.Metadata) error {
	if options.Quiet {
		return nil
	}
	colorMode := colorModeForWriter(cmd)
	nextSteps := i18n.T("cmd.init.success.next_steps", nil)
	if colorMode.Enabled() {
		nextSteps = view.CtaStyle.Render(nextSteps)
	}
	lines := []string{
		i18n.T("cmd.init.success", &i18n.Tvars{
			Data: &i18n.TData{
				"configPath":  meta.ConfigPath,
				"loader":      options.Loader.String(),
				"gameVersion": options.GameVersion,
			},
		}),
		nextSteps,
	}
	return runOutputLines(cmd, deps, lines)
}

func runOutputLines(cmd *cobra.Command, deps initDeps, lines []string) error {
	runTea := deps.runTea
	if runTea == nil {
		runTea = defaultRunTea
	}
	return view.RunOutputLines(runTea, view.OutputLinesModel{
		Lines:     lines,
		Output:    cmd.OutOrStdout(),
		Separator: view.SectionSeparatorLine,
	}, outputProgramOptions(cmd)...)
}

func outputProgramOptions(cmd *cobra.Command) []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithInput(nil),
		tea.WithOutput(cmd.OutOrStdout()),
		tea.WithoutRenderer(),
	}
}

type outputLinesModel = view.OutputLinesModel
type outputLineErrorMsg = view.OutputLineErrorMsg

func outputLineCmd(out io.Writer, line string) tea.Cmd {
	return view.OutputLineCmd(out, line)
}

func outputLinesModelError(result tea.Model) error {
	return view.OutputLinesModelError(result)
}
