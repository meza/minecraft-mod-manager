package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
)

func ReadConfig(ctx context.Context, fs afero.Fs, meta Metadata) (models.ModsJson, error) {
	_, span := perf.StartSpan(ctx, "io.config.read", perf.WithAttributes(attribute.String("config_path", meta.ConfigPath)))
	defer span.End()

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

func WriteConfig(ctx context.Context, fs afero.Fs, meta Metadata, config models.ModsJson) error {
	_, span := perf.StartSpan(ctx, "io.config.write", perf.WithAttributes(attribute.String("config_path", meta.ConfigPath)))
	defer span.End()

	data, _ := json.MarshalIndent(config, "", "  ")
	return writeFileAtomic(fs, meta.ConfigPath, data, 0644)
}

func InitConfig(ctx context.Context, fs afero.Fs, meta Metadata, minecraftClient httpClient.Doer) (models.ModsJson, error) {
	_, span := perf.StartSpan(ctx, "io.config.init", perf.WithAttributes(attribute.String("config_path", meta.ConfigPath)))
	defer span.End()

	latest, err := minecraft.GetLatestVersion(ctx, minecraftClient)
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

	if err := WriteConfig(ctx, fs, meta, config); err != nil {
		return models.ModsJson{}, err
	}
	return config, nil
}
