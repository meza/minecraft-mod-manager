package config

import (
	"encoding/json"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/spf13/afero"
)

func ReadConfig(fs afero.Fs, meta Metadata) (models.ModsJson, error) {
	region := perf.StartRegion("io.config.read")
	defer region.End()

	exists, _ := afero.Exists(fs, meta.ConfigPath)
	if !exists {
		return models.ModsJson{}, &ConfigFileNotFoundException{Path: meta.ConfigPath}
	}

	data, err := afero.ReadFile(fs, meta.ConfigPath)
	if err != nil {
		return models.ModsJson{}, fmt.Errorf("failed to read configuration file: %w", err)
	}

	var config models.ModsJson
	if err := json.Unmarshal(data, &config); err != nil {
		return models.ModsJson{}, &ConfigFileInvalidError{Err: err}
	}

	return config, nil
}

func WriteConfig(fs afero.Fs, meta Metadata, config models.ModsJson) error {
	region := perf.StartRegion("io.config.write")
	defer region.End()

	data, _ := json.MarshalIndent(config, "", "  ")
	return afero.WriteFile(fs, meta.ConfigPath, data, 0644)
}

func InitConfig(fs afero.Fs, meta Metadata, minecraftClient httpClient.Doer) (models.ModsJson, error) {
	region := perf.StartRegion("io.config.init")
	defer region.End()

	latest, err := minecraft.GetLatestVersion(minecraftClient)
	if err != nil {
		return models.ModsJson{}, err
	}

	config := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                latest,
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release, models.Beta},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	if err := WriteConfig(fs, meta, config); err != nil {
		return models.ModsJson{}, err
	}
	return config, nil
}
