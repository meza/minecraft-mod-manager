package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
)

var marshalIndent = json.MarshalIndent

func ReadConfig(ctx context.Context, fs afero.Fs, meta Metadata) (models.ModsJSON, error) {
	_, span := perf.StartSpan(ctx, "io.config.read", perf.WithAttributes(attribute.String("config_path", meta.ConfigPath)))
	defer span.End()

	exists, err := afero.Exists(fs, meta.ConfigPath)
	if err != nil {
		return models.ModsJSON{}, fmt.Errorf("failed to check configuration file: %w", err)
	}
	if !exists {
		return models.ModsJSON{}, &ConfigFileNotFoundException{Path: meta.ConfigPath}
	}

	data, err := afero.ReadFile(fs, meta.ConfigPath)
	if err != nil {
		return models.ModsJSON{}, fmt.Errorf("failed to read configuration file: %w", err)
	}

	var config models.ModsJSON
	if err := json.Unmarshal(data, &config); err != nil {
		return models.ModsJSON{}, &FileInvalidError{Err: err}
	}

	return config, nil
}

func WriteConfig(ctx context.Context, fs afero.Fs, meta Metadata, config models.ModsJSON) error {
	_, span := perf.StartSpan(ctx, "io.config.write", perf.WithAttributes(attribute.String("config_path", meta.ConfigPath)))
	defer span.End()

	data, err := marshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %w", err)
	}
	return writeFileAtomic(fs, meta.ConfigPath, data, 0644)
}

func InitConfig(ctx context.Context, fs afero.Fs, meta Metadata, minecraftClient httpclient.Doer) (models.ModsJSON, error) {
	_, span := perf.StartSpan(ctx, "io.config.init", perf.WithAttributes(attribute.String("config_path", meta.ConfigPath)))
	defer span.End()

	latest, err := minecraft.GetLatestVersion(ctx, minecraftClient)
	if err != nil {
		return models.ModsJSON{}, err
	}

	config := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                latest,
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release, models.Beta},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	if err := WriteConfig(ctx, fs, meta, config); err != nil {
		return models.ModsJSON{}, err
	}
	return config, nil
}
