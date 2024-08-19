package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/fileutils"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
	"path/filepath"
)

func getLockfileName(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), filepath.Base(configPath)+"-lock.json")
}

func writeConfigFile(config models.ModsJson, configPath string, fs afero.Fs) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return afero.WriteFile(fs, configPath, data, 0644)
}

func writeLockFile(config []models.ModInstall, configPath string, fs afero.Fs) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return afero.WriteFile(fs, configPath, data, 0644)
}

func EnsureLockFile(configPath string, filesystem ...afero.Fs) ([]models.ModInstall, error) {
	fs := initFilesystem(filesystem...)
	lockFilePath := getLockfileName(configPath)
	if !fileutils.FileExists(lockFilePath) {
		emptyModLock := []models.ModInstall{}
		if err := writeLockFile(emptyModLock, lockFilePath, fs); err != nil {
			return nil, err
		}
		return emptyModLock, nil
	}
	data, err := afero.ReadFile(fs, lockFilePath)
	if err != nil {
		return nil, err
	}
	var config []models.ModInstall
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func readConfigFile(configPath string, fs afero.Fs) (models.ModsJson, error) {
	if !fileutils.FileExists(configPath, fs) {
		return models.ModsJson{}, &ConfigFileNotFoundException{Path: configPath}
	}

	data, err := afero.ReadFile(fs, configPath)
	if err != nil {
		return models.ModsJson{}, fmt.Errorf("failed to read configuration file: %w", err)
	}
	var config models.ModsJson
	if err := json.Unmarshal(data, &config); err != nil {
		return models.ModsJson{}, err
	}
	return config, nil
}

func initializeConfigFile(configPath string, fs afero.Fs) (models.ModsJson, error) {
	// Placeholder for actual initialization logic
	emptyModJson := models.ModsJson{
		Loader:                     models.FORGE,
		GameVersion:                "1.16.5",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release, models.Beta},
		ModsFolder:                 "mods",
		Mods:                       []models.ModInstall{},
	}
	if err := writeConfigFile(emptyModJson, configPath, fs); err != nil {
		return models.ModsJson{}, err
	}
	return emptyModJson, nil
}

func initFilesystem(filesystem ...afero.Fs) afero.Fs {
	if len(filesystem) > 0 {
		return filesystem[0]
	}

	return afero.NewOsFs()
}

func EnsureConfiguration(configPath string, quiet bool, filesystem ...afero.Fs) (models.ModsJson, error) {
	fs := initFilesystem(filesystem...)
	config, err := readConfigFile(configPath, fs)
	if err != nil {
		var cf *ConfigFileNotFoundException
		if errors.As(err, &cf) && !quiet {
			// Placeholder for user interaction logic
			shouldCreateConfig := true // Assume user wants to create config
			if shouldCreateConfig {
				return initializeConfigFile(configPath, fs)
			}
		}
		return models.ModsJson{}, err
	}
	// Placeholder for validation logic
	return config, nil
}

func GetModsFolder(configPath string, config models.ModsJson) string {
	if filepath.IsAbs(config.ModsFolder) {
		return config.ModsFolder
	}
	return filepath.Join(filepath.Dir(filepath.FromSlash(configPath)), config.ModsFolder)
}
