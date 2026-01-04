package init

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
)

func normalizeGameVersion(ctx context.Context, options initOptions, deps initDeps, mode gameVersionNormalizationMode) (initOptions, error) {
	if options.GameVersion == "" {
		return options, nil
	}

	if strings.EqualFold(options.GameVersion, "latest") {
		if mode == gameVersionInteractive {
			options.GameVersion = ""
			options.Provided.GameVersion = false
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

func normalizeGameVersionInteractive(options initOptions) initOptions {
	if options.GameVersion == "" {
		return options
	}
	if strings.EqualFold(options.GameVersion, "latest") {
		options.GameVersion = ""
		options.Provided.GameVersion = false
	}
	return options
}

func validateModsFolder(fs afero.Fs, meta config.Metadata, modsFolder string) error {
	modsFolder = strings.TrimSpace(modsFolder)
	if modsFolder == "" {
		return errors.New(i18n.T("cmd.init.error.mods-folder.empty", nil))
	}

	modsFolderConfig := models.ModsJSON{ModsFolder: modsFolder}
	modsFolderPath := meta.ModsFolderPath(modsFolderConfig)
	modsFolderExists, err := afero.Exists(fs, modsFolderPath)
	if err != nil {
		return err
	}
	if !modsFolderExists {
		return fmt.Errorf("%s", i18n.T("cmd.init.error.mods-folder.missing", &i18n.Tvars{
			Data: &i18n.TData{"path": modsFolderPath},
		}))
	}

	isDir, err := afero.IsDir(fs, modsFolderPath)
	if err != nil {
		return err
	}
	if !isDir {
		return fmt.Errorf("%s", i18n.T("cmd.init.error.mods-folder.not-directory", &i18n.Tvars{
			Data: &i18n.TData{"path": modsFolderPath},
		}))
	}

	return nil
}

func validateModsFolderInteractive(fs afero.Fs, meta config.Metadata, modsFolder string) error {
	modsFolder = strings.TrimSpace(modsFolder)
	if modsFolder == "" {
		return errors.New(i18n.T("cmd.init.error.mods-folder.empty", nil))
	}

	modsFolderConfig := models.ModsJSON{ModsFolder: modsFolder}
	modsFolderPath := meta.ModsFolderPath(modsFolderConfig)
	modsFolderExists, err := afero.Exists(fs, modsFolderPath)
	if err != nil {
		return err
	}
	if !modsFolderExists {
		return errors.New(i18n.T("cmd.init.prompt.mods-folder.missing", nil))
	}

	isDir, err := afero.IsDir(fs, modsFolderPath)
	if err != nil {
		return err
	}
	if !isDir {
		return errors.New(i18n.T("cmd.init.prompt.mods-folder.not-directory", nil))
	}

	return nil
}

func initWithDeps(ctx context.Context, options initOptions, deps initDeps) error {
	if err := requireLoader(options); err != nil {
		return err
	}

	meta := config.NewMetadata(options.ConfigPath)
	if err := deps.fs.MkdirAll(meta.Dir(), 0755); err != nil {
		return err
	}

	cfg := models.ModsJSON{
		Loader:                     options.Loader,
		GameVersion:                options.GameVersion,
		DefaultAllowedReleaseTypes: options.ReleaseTypes,
		ModsFolder:                 options.ModsFolder,
		Mods:                       []models.Mod{},
	}

	if err := config.WriteConfig(ctx, deps.fs, meta, cfg); err != nil {
		return err
	}
	if err := config.WriteLock(ctx, deps.fs, meta, []models.ModInstall{}); err != nil {
		return err
	}

	return nil
}

func requireLoader(options initOptions) error {
	if options.Loader == "" {
		return errors.New(i18n.T("cmd.init.error.loader.required", nil))
	}
	return nil
}
