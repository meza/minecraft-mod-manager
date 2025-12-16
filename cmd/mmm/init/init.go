package init

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

func Command() *cobra.Command {
	var loader loaderFlag

	cmd := &cobra.Command{
		Use:   "init",
		Short: i18n.T("cmd.init.short"),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			ctx, span := perf.StartSpan(cmd.Context(), "app.command.init")
			defer func() {
				span.SetAttributes(attribute.Bool("success", err == nil))
				span.End()
			}()

			gameVersion, err := cmd.Flags().GetString("game-version")
			if err != nil {
				return err
			}
			modsFolder, err := cmd.Flags().GetString("mods-folder")
			if err != nil {
				return err
			}
			releaseTypesRaw, err := cmd.Flags().GetStringSlice("release-types")
			if err != nil {
				return err
			}
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			quiet, err := cmd.Flags().GetBool("quiet")
			if err != nil {
				return err
			}
			debug, err := cmd.Flags().GetBool("debug")
			if err != nil {
				return err
			}

			log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quiet, debug)
			deps := initDeps{
				fs:              afero.NewOsFs(),
				minecraftClient: http.DefaultClient,
				prompter: terminalPrompter{
					in:  cmd.InOrStdin(),
					out: cmd.OutOrStdout(),
				},
				logger: log,
			}

			releaseTypes, err := parseReleaseTypes(releaseTypesRaw)
			if err != nil {
				return err
			}

			options := initOptions{
				ConfigPath:   configPath,
				Quiet:        quiet,
				Debug:        debug,
				Loader:       loader.value,
				GameVersion:  gameVersion,
				ReleaseTypes: releaseTypes,
				ModsFolder:   modsFolder,
				Provided: providedFlags{
					Loader:       cmd.Flags().Changed("loader"),
					GameVersion:  cmd.Flags().Changed("game-version"),
					ReleaseTypes: cmd.Flags().Changed("release-types"),
					ModsFolder:   cmd.Flags().Changed("mods-folder"),
				},
			}

			meta := config.NewMetadata(configPath)

			useTUI := tui.ShouldUseTUI(options.Quiet, cmd.InOrStdin(), cmd.OutOrStdout())

			options, err = normalizeGameVersion(ctx, options, deps, useTUI)
			if err != nil {
				return err
			}

			if useTUI {
				updated, err := runInteractiveInit(ctx, cmd, options, deps, meta)
				if err != nil {
					return err
				}
				options = updated
			}

			_, err = initWithDeps(ctx, options, deps)
			return err
		},
	}

	cmd.Flags().VarP(&loader, "loader", "l", i18n.T("cmd.init.usage.loader", i18n.Tvars{
		Data: &i18n.TData{"loaders": getAllLoaders()},
	}))
	cmd.Flags().StringSliceP("release-types", "r", []string{"release"}, i18n.T("cmd.init.usage.release-types", i18n.Tvars{
		Data: &i18n.TData{"releaseTypes": getAllReleaseTypes()},
	}))
	cmd.Flags().StringP("game-version", "g", "latest", i18n.T("cmd.init.usage.game-version"))
	cmd.Flags().StringP("mods-folder", "m", "mods", i18n.T("cmd.init.usage.mods-folder", i18n.Tvars{
		Data: &i18n.TData{"cwd": getCurrentWorkingDirectory()},
	}))

	_ = cmd.RegisterFlagCompletionFunc("loader", completeLoaders)
	_ = cmd.RegisterFlagCompletionFunc("release-types", completeReleaseTypes)

	return cmd
}

type initOptions struct {
	ConfigPath   string
	Quiet        bool
	Debug        bool
	Loader       models.Loader
	GameVersion  string
	ReleaseTypes []models.ReleaseType
	ModsFolder   string
	Provided     providedFlags
}

type providedFlags struct {
	Loader       bool
	GameVersion  bool
	ReleaseTypes bool
	ModsFolder   bool
}

type initDeps struct {
	fs              afero.Fs
	minecraftClient httpClient.Doer
	prompter        prompter
	logger          *logger.Logger
}

type prompter interface {
	ConfirmOverwrite(configPath string) (bool, error)
	RequestNewConfigPath(configPath string) (string, error)
}

type terminalPrompter struct {
	in  io.Reader
	out io.Writer
}

func (p terminalPrompter) ConfirmOverwrite(configPath string) (bool, error) {
	_, _ = fmt.Fprintf(p.out, "Configuration file already exists at %s. Overwrite? (y/N): ", configPath)
	answer, err := readLine(p.in)
	if err != nil {
		return false, err
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func (p terminalPrompter) RequestNewConfigPath(configPath string) (string, error) {
	_, _ = fmt.Fprintf(p.out, "Enter a new config file path (current: %s): ", configPath)
	answer, err := readLine(p.in)
	if err != nil {
		return "", err
	}

	answer = strings.TrimSpace(answer)
	if answer == "" {
		return "", fmt.Errorf("config path cannot be empty")
	}
	return answer, nil
}

func readLine(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return scanner.Text(), nil
}

func runInteractiveInit(ctx context.Context, cmd *cobra.Command, options initOptions, deps initDeps, meta config.Metadata) (initOptions, error) {
	sessionCtx, sessionSpan := perf.StartSpan(ctx, "tui.init.session",
		perf.WithAttributes(
			attribute.Bool("provided_loader", options.Provided.Loader),
			attribute.Bool("provided_game_version", options.Provided.GameVersion),
			attribute.Bool("provided_release_types", options.Provided.ReleaseTypes),
			attribute.Bool("provided_mods_folder", options.Provided.ModsFolder),
		),
	)
	defer sessionSpan.End()

	model := NewModel(sessionCtx, sessionSpan, options, deps, meta)

	if model.state == done {
		return model.result, nil
	}

	program := tea.NewProgram(model, tui.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	result, err := program.Run()
	if err != nil {
		return options, err
	}

	var finalModel CommandModel
	switch typed := result.(type) {
	case *CommandModel:
		finalModel = *typed
	case CommandModel:
		finalModel = typed
	default:
		return options, fmt.Errorf("interactive init failed")
	}

	if finalModel.err != nil {
		return options, finalModel.err
	}

	if finalModel.state != done {
		return options, fmt.Errorf("init cancelled")
	}

	return finalModel.result, nil
}

func normalizeGameVersion(ctx context.Context, options initOptions, deps initDeps, interactive bool) (initOptions, error) {
	if options.GameVersion == "" {
		return options, nil
	}

	if strings.EqualFold(options.GameVersion, "latest") {
		if interactive && !options.Provided.GameVersion {
			options.GameVersion = ""
			return options, nil
		}

		latest, err := minecraft.GetLatestVersion(ctx, deps.minecraftClient)
		if err != nil {
			return options, err
		}
		options.GameVersion = latest
	}

	return options, nil
}

func validateModsFolder(fs afero.Fs, meta config.Metadata, modsFolder string) (string, error) {
	modsFolder = strings.TrimSpace(modsFolder)
	if modsFolder == "" {
		return "", fmt.Errorf("mods folder cannot be empty")
	}

	modsFolderConfig := models.ModsJson{ModsFolder: modsFolder}
	modsFolderPath := meta.ModsFolderPath(modsFolderConfig)
	modsFolderExists, err := afero.Exists(fs, modsFolderPath)
	if err != nil {
		return "", err
	}
	if !modsFolderExists {
		return "", fmt.Errorf("mods folder does not exist: %s", modsFolderPath)
	}

	isDir, err := afero.IsDir(fs, modsFolderPath)
	if err != nil {
		return "", err
	}
	if !isDir {
		return "", fmt.Errorf("mods folder is not a directory: %s", modsFolderPath)
	}

	return modsFolderPath, nil
}

func initWithDeps(ctx context.Context, options initOptions, deps initDeps) (config.Metadata, error) {
	if options.Loader == "" {
		return config.Metadata{}, fmt.Errorf("init requires flag: -l/--loader")
	}

	if options.GameVersion == "" || strings.EqualFold(options.GameVersion, "latest") {
		latest, err := minecraft.GetLatestVersion(ctx, deps.minecraftClient)
		if err != nil {
			return config.Metadata{}, fmt.Errorf("could not determine latest minecraft version; provide -g/--game-version")
		}
		options.GameVersion = latest
	}

	meta := config.NewMetadata(options.ConfigPath)

	exists, _ := afero.Exists(deps.fs, meta.ConfigPath)
	if exists {
		if options.Quiet {
			return config.Metadata{}, fmt.Errorf("configuration file already exists: %s", meta.ConfigPath)
		}

		overwrite, err := deps.prompter.ConfirmOverwrite(meta.ConfigPath)
		if err != nil {
			return config.Metadata{}, err
		}
		if !overwrite {
			newPath, err := deps.prompter.RequestNewConfigPath(meta.ConfigPath)
			if err != nil {
				return config.Metadata{}, err
			}
			meta = config.NewMetadata(newPath)
		}
	}

	if _, err := validateModsFolder(deps.fs, meta, options.ModsFolder); err != nil {
		return config.Metadata{}, err
	}

	if !minecraft.IsValidVersion(ctx, options.GameVersion, deps.minecraftClient) {
		return config.Metadata{}, fmt.Errorf("invalid minecraft version: %s", options.GameVersion)
	}

	if err := deps.fs.MkdirAll(meta.Dir(), 0755); err != nil {
		return config.Metadata{}, err
	}

	cfg := models.ModsJson{
		Loader:                     options.Loader,
		GameVersion:                options.GameVersion,
		DefaultAllowedReleaseTypes: options.ReleaseTypes,
		ModsFolder:                 options.ModsFolder,
		Mods:                       []models.Mod{},
	}

	if err := config.WriteConfig(ctx, deps.fs, meta, cfg); err != nil {
		return config.Metadata{}, err
	}
	if err := config.WriteLock(ctx, deps.fs, meta, []models.ModInstall{}); err != nil {
		return config.Metadata{}, err
	}

	if deps.logger != nil {
		deps.logger.Log("Initialized configuration at "+meta.ConfigPath, false)
	}

	return meta, nil
}

func getCurrentWorkingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func getAllReleaseTypes() string {
	releaseTypes := models.AllReleaseTypes()
	var releaseTypeList string

	for i, releaseType := range releaseTypes {
		releaseTypeList += fmt.Sprintf("%s", releaseType)
		if i < len(releaseTypes)-1 {
			releaseTypeList += ", "
		}
	}

	return releaseTypeList
}

func getAllLoaders() string {
	loaders := models.AllLoaders()
	var loaderList string

	for i, loader := range loaders {
		loaderList += fmt.Sprintf("%s", string(loader))
		if i < len(loaders)-1 {
			loaderList += ", "
		}
	}

	return loaderList
}

type loaderFlag struct {
	value models.Loader
}

func (f *loaderFlag) String() string {
	return f.value.String()
}

func (f *loaderFlag) Set(value string) error {
	candidate := models.Loader(strings.TrimSpace(strings.ToLower(value)))
	for _, loader := range models.AllLoaders() {
		if loader == candidate {
			f.value = candidate
			return nil
		}
	}
	return fmt.Errorf("invalid loader: %s", value)
}

func (f *loaderFlag) Type() string {
	return "loader"
}

func isValidReleaseType(releaseType models.ReleaseType) bool {
	for _, r := range models.AllReleaseTypes() {
		if r == releaseType {
			return true
		}
	}
	return false
}

func parseReleaseTypes(raw []string) ([]models.ReleaseType, error) {
	releaseTypes := make([]models.ReleaseType, 0, len(raw))

	for _, part := range raw {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}

		candidate := models.ReleaseType(part)
		if !isValidReleaseType(candidate) {
			return nil, fmt.Errorf("invalid release type: %s", part)
		}
		releaseTypes = append(releaseTypes, candidate)
	}

	if len(releaseTypes) == 0 {
		return nil, fmt.Errorf("release types cannot be empty")
	}

	return releaseTypes, nil
}

func completeLoaders(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	loaders := models.AllLoaders()
	out := make([]string, 0, len(loaders))
	for _, loader := range loaders {
		out = append(out, loader.String())
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func completeReleaseTypes(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	releaseTypes := models.AllReleaseTypes()
	out := make([]string, 0, len(releaseTypes))
	for _, releaseType := range releaseTypes {
		out = append(out, string(releaseType))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
