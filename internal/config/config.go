package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/fileutils"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"os"
	"path/filepath"
)

type ConfigFileInvalidError struct {
	Err error
}

type ConfigFileNotFoundException struct {
	Path string
	Err  error
}

func (e *ConfigFileInvalidError) Error() string {
	return fmt.Sprintf("Configuration file is invalid: %s", e.Err)
}

func (e *ConfigFileNotFoundException) Error() string {
	return fmt.Sprintf("Configuration file not found: %s", e.Path)
}

func getLockfileName(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), filepath.Base(configPath)+"-lock.json")
}

func writeConfigFile(config models.ModsJson, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func writeLockFile(config []models.ModInstall, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func readLockFile(configPath string) ([]models.ModInstall, error) {
	lockFilePath := getLockfileName(configPath)
	if !fileutils.FileExists(lockFilePath) {
		emptyModLock := []models.ModInstall{}
		if err := writeLockFile(emptyModLock, lockFilePath); err != nil {
			return nil, err
		}
		return emptyModLock, nil
	}
	data, err := os.ReadFile(lockFilePath)
	if err != nil {
		return nil, err
	}
	var config []models.ModInstall
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func readConfigFile(configPath string) (models.ModsJson, error) {
	if !fileutils.FileExists(configPath) {
		return models.ModsJson{}, &ConfigFileNotFoundException{Path: configPath}
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return models.ModsJson{}, err
	}
	var config models.ModsJson
	if err := json.Unmarshal(data, &config); err != nil {
		return models.ModsJson{}, err
	}
	return config, nil
}

func initializeConfigFile(configPath string) (models.ModsJson, error) {
	// Placeholder for actual initialization logic
	emptyModJson := models.ModsJson{
		Loader:                     models.FORGE,
		GameVersion:                "1.16.5",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release, models.Beta},
		ModsFolder:                 "mods",
		Mods:                       []models.ModInstall{},
	}
	if err := writeConfigFile(emptyModJson, configPath); err != nil {
		return models.ModsJson{}, err
	}
	return emptyModJson, nil
}

func EnsureConfiguration(configPath string, quiet bool) (models.ModsJson, error) {
	config, err := readConfigFile(configPath)
	if err != nil {
		var cf *ConfigFileNotFoundException
		if errors.As(err, &cf) && !quiet {
			// Placeholder for user interaction logic
			shouldCreateConfig := true // Assume user wants to create config
			if shouldCreateConfig {
				return initializeConfigFile(configPath)
			}
		}
		return models.ModsJson{}, err
	}
	// Placeholder for validation logic
	return config, nil
}

func getModsFolder(configPath string, config models.ModsJson) string {
	if filepath.IsAbs(config.ModsFolder) {
		return config.ModsFolder
	}
	return filepath.Join(filepath.Dir(configPath), config.ModsFolder)
}
