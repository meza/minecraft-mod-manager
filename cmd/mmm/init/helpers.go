package init

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
)

func normalizeGameVersion(ctx context.Context, options initOptions, deps initDeps, mode gameVersionNormalizationMode) (initOptions, error) {
	if options.GameVersion == "" {
		return options, nil
	}

	if strings.EqualFold(options.GameVersion, "latest") {
		if mode == gameVersionInteractive && !options.Provided.GameVersion {
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

func validateModsFolder(fs afero.Fs, meta config.Metadata, modsFolder string) error {
	modsFolder = strings.TrimSpace(modsFolder)
	if modsFolder == "" {
		return errors.New(i18n.T("cmd.init.error.mods-folder.empty", i18n.Tvars{}))
	}

	modsFolderConfig := models.ModsJSON{ModsFolder: modsFolder}
	modsFolderPath := meta.ModsFolderPath(modsFolderConfig)
	modsFolderExists, err := afero.Exists(fs, modsFolderPath)
	if err != nil {
		return err
	}
	if !modsFolderExists {
		return fmt.Errorf("%s", i18n.T("cmd.init.error.mods-folder.missing", i18n.Tvars{
			Data: &i18n.TData{"path": modsFolderPath},
		}))
	}

	isDir, err := afero.IsDir(fs, modsFolderPath)
	if err != nil {
		return err
	}
	if !isDir {
		return fmt.Errorf("%s", i18n.T("cmd.init.error.mods-folder.not-directory", i18n.Tvars{
			Data: &i18n.TData{"path": modsFolderPath},
		}))
	}

	return nil
}

func initWithDeps(ctx context.Context, options initOptions, deps initDeps) (config.Metadata, error) {
	if err := requireLoader(options); err != nil {
		return config.Metadata{}, err
	}

	options, err := resolveLatestGameVersion(ctx, options, deps)
	if err != nil {
		return config.Metadata{}, err
	}

	meta := config.NewMetadata(options.ConfigPath)
	meta, err = resolveConfigPath(meta, options, deps)
	if err != nil {
		return config.Metadata{}, err
	}

	if err := validateModsFolder(deps.fs, meta, options.ModsFolder); err != nil {
		return config.Metadata{}, err
	}

	if err := validateGameVersion(ctx, options.GameVersion, deps); err != nil {
		return config.Metadata{}, err
	}

	if err := deps.fs.MkdirAll(meta.Dir(), 0755); err != nil {
		return config.Metadata{}, err
	}

	cfg := models.ModsJSON{
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
		deps.logger.Log(i18n.T("cmd.init.success", i18n.Tvars{
			Data: &i18n.TData{"configPath": meta.ConfigPath},
		}), logger.LogQuiet)
	}

	return meta, nil
}

func validateGameVersion(ctx context.Context, gameVersion string, deps initDeps) error {
	valid, validationErr := minecraft.IsValidVersion(ctx, gameVersion, deps.minecraftClient)
	if validationErr != nil {
		return fmt.Errorf("%s", i18n.T("cmd.init.error.game-version.unavailable", i18n.Tvars{}))
	}
	if !valid {
		return fmt.Errorf("%s", i18n.T("cmd.init.error.game-version.invalid", i18n.Tvars{
			Data: &i18n.TData{"gameVersion": gameVersion},
		}))
	}
	return nil
}

func requireLoader(options initOptions) error {
	if options.Loader == "" {
		return errors.New(i18n.T("cmd.init.error.loader.required", i18n.Tvars{}))
	}
	return nil
}

func resolveLatestGameVersion(ctx context.Context, options initOptions, deps initDeps) (initOptions, error) {
	if options.GameVersion != "" && !strings.EqualFold(options.GameVersion, "latest") {
		return options, nil
	}

	latest, err := minecraft.GetLatestVersion(ctx, deps.minecraftClient)
	if err != nil {
		return options, errors.New(i18n.T("cmd.init.error.game-version.latest-unavailable", i18n.Tvars{}))
	}
	options.GameVersion = latest
	return options, nil
}

func resolveConfigPath(meta config.Metadata, options initOptions, deps initDeps) (config.Metadata, error) {
	exists, err := afero.Exists(deps.fs, meta.ConfigPath)
	if err != nil {
		return config.Metadata{}, fmt.Errorf("%s: %w", i18n.T("cmd.init.error.config-file.check", i18n.Tvars{}), err)
	}
	if !exists {
		return meta, nil
	}
	if options.Quiet {
		return config.Metadata{}, fmt.Errorf("%s", i18n.T("cmd.init.error.config-file.exists", i18n.Tvars{
			Data: &i18n.TData{"configPath": meta.ConfigPath},
		}))
	}

	overwrite, err := deps.prompter.ConfirmOverwrite(meta.ConfigPath)
	if err != nil {
		return config.Metadata{}, err
	}
	if overwrite {
		return meta, nil
	}

	newPath, err := deps.prompter.RequestNewConfigPath(meta.ConfigPath)
	if err != nil {
		return config.Metadata{}, err
	}
	return config.NewMetadata(newPath), nil
}
