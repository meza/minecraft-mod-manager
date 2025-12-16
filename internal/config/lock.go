package config

import (
	"encoding/json"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/spf13/afero"
)

func EnsureLock(fs afero.Fs, meta Metadata) ([]models.ModInstall, error) {
	region := perf.StartRegion("io.config.lock.ensure")
	defer region.End()

	lockPath := meta.LockPath()
	exists, _ := afero.Exists(fs, lockPath)
	if !exists {
		empty := make([]models.ModInstall, 0)
		if err := WriteLock(fs, meta, empty); err != nil {
			return nil, err
		}
		return empty, nil
	}

	return ReadLock(fs, meta)
}

func ReadLock(fs afero.Fs, meta Metadata) ([]models.ModInstall, error) {
	region := perf.StartRegion("io.config.lock.read")
	defer region.End()

	data, err := afero.ReadFile(fs, meta.LockPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lock []models.ModInstall
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}

	return lock, nil
}

func WriteLock(fs afero.Fs, meta Metadata, lock []models.ModInstall) error {
	region := perf.StartRegion("io.config.lock.write")
	defer region.End()

	data, _ := json.MarshalIndent(lock, "", "  ")
	return afero.WriteFile(fs, meta.LockPath(), data, 0644)
}
