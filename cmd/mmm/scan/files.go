package scan

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/mmmignore"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
)

func listJarFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON) ([]string, error) {
	all, err := afero.ReadDir(fs, meta.ModsFolderPath(cfg))
	if err != nil {
		return nil, err
	}

	candidates := make([]string, 0, len(all))
	for _, entry := range all {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
			continue
		}
		candidates = append(candidates, filepath.Join(meta.ModsFolderPath(cfg), entry.Name()))
	}

	patterns, err := mmmignore.ListPatterns(fs, meta.Dir())
	if err != nil {
		return nil, err
	}

	filtered := make([]string, 0, len(candidates))
	for _, path := range candidates {
		if mmmignore.IsIgnored(meta.ModsFolderPath(cfg), path, patterns) {
			continue
		}
		filtered = append(filtered, path)
	}
	return filtered, nil
}

func fileIsManaged(filePath string, installations []models.ModInstall) bool {
	filename := filepath.Base(filePath)
	for _, install := range installations {
		if install.FileName == filename {
			return true
		}
	}
	return false
}

func sha1Candidates(ctx context.Context, fs afero.Fs, files []string) ([]scanCandidate, error) {
	out := make([]scanCandidate, len(files))

	group, groupCtx := errgroup.WithContext(ctx)
	limit := runtime.GOMAXPROCS(0)
	group.SetLimit(limit)

	for i := range files {
		i := i
		group.Go(func() error {
			sha, err := sha1ForFile(groupCtx, fs, files[i])
			if err != nil {
				return err
			}
			out[i] = scanCandidate{
				Path:     files[i],
				FileName: filepath.Base(files[i]),
				Sha1:     sha,
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

var newSha1Hasher = sha1.New

func sha1ForFile(ctx context.Context, fs afero.Fs, path string) (hash string, returnErr error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	hasher := newSha1Hasher()
	buf := make([]byte, 32*1024)
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		n, err := file.Read(buf)
		if n > 0 {
			if _, writeErr := hasher.Write(buf[:n]); writeErr != nil {
				return "", writeErr
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
