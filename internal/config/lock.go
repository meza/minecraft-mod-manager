package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
)

func EnsureLock(ctx context.Context, fs afero.Fs, meta Metadata) ([]models.ModInstall, error) {
	_, span := perf.StartSpan(ctx, "io.config.lock.ensure", perf.WithAttributes(attribute.String("lock_path", meta.LockPath())))
	defer span.End()

	lockPath := meta.LockPath()
	exists, err := afero.Exists(fs, lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check lock file: %w", err)
	}
	if !exists {
		empty := make([]models.ModInstall, 0)
		if err := WriteLock(ctx, fs, meta, empty); err != nil {
			return nil, err
		}
		return empty, nil
	}

	return ReadLock(ctx, fs, meta)
}

func ReadLock(ctx context.Context, fs afero.Fs, meta Metadata) ([]models.ModInstall, error) {
	_, span := perf.StartSpan(ctx, "io.config.lock.read", perf.WithAttributes(attribute.String("lock_path", meta.LockPath())))
	defer span.End()

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

func WriteLock(ctx context.Context, fs afero.Fs, meta Metadata, lock []models.ModInstall) error {
	_, span := perf.StartSpan(ctx, "io.config.lock.write", perf.WithAttributes(attribute.String("lock_path", meta.LockPath())))
	defer span.End()

	data, err := marshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize lock file: %w", err)
	}
	return writeFileAtomic(fs, meta.LockPath(), data)
}
