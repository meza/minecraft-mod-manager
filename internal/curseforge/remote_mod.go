package curseforge

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/gameversion"
	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

// FetchRemoteMod selects the newest compatible CurseForge file and returns it as a domain RemoteMod.
func FetchRemoteMod(ctx context.Context, projectID string, opts models.FetchOptions, client httpclient.Doer) (models.RemoteMod, error) {
	curseforgeClient := NewClient(client)

	modLoaderType, err := modLoaderTypeFromLoader(opts.Loader)
	if err != nil {
		return models.RemoteMod{}, err
	}

	projectIDNumber, err := strconv.Atoi(projectID)
	if err != nil {
		return models.RemoteMod{}, &models.ModNotFoundError{Platform: models.CURSEFORGE, ProjectID: projectID}
	}

	project, err := GetProject(ctx, projectID, curseforgeClient)
	if err != nil {
		return models.RemoteMod{}, mapProjectNotFoundError(projectID, err)
	}

	currentVersion := opts.GameVersion

	for {
		files, filesErr := GetFilesForProjectWithFilters(ctx, projectIDNumber, FileListFilter{
			GameVersion:   currentVersion,
			ModLoaderType: modLoaderType,
		}, curseforgeClient)
		if filesErr != nil {
			return models.RemoteMod{}, mapProjectNotFoundError(projectID, filesErr)
		}

		candidates := filterCurseforgeFiles(files, opts, currentVersion)
		if len(candidates) == 0 {
			next, canGoDown := gameversion.NextPatchDown(currentVersion)
			if opts.AllowFallback && canGoDown {
				currentVersion = next
				continue
			}
			return models.RemoteMod{}, &models.NoCompatibleFileError{Platform: models.CURSEFORGE, ProjectID: projectID}
		}

		sort.SliceStable(candidates, func(leftIndex int, rightIndex int) bool {
			return candidates[leftIndex].FileDate.After(candidates[rightIndex].FileDate)
		})

		selected := candidates[0]
		return buildRemoteMod(project, selected, projectID)
	}
}

func mapProjectNotFoundError(projectID string, err error) error {
	var notFound *globalerrors.ProjectNotFoundError
	if errors.As(err, &notFound) {
		return &models.ModNotFoundError{Platform: models.CURSEFORGE, ProjectID: projectID}
	}
	return err
}

func buildRemoteMod(project *Project, selected File, projectID string) (models.RemoteMod, error) {
	if selected.DownloadURL == "" {
		return models.RemoteMod{}, &models.NoCompatibleFileError{Platform: models.CURSEFORGE, ProjectID: projectID}
	}

	hash, hashErr := getCurseforgeHash(selected.Hashes)
	if hashErr != nil {
		return models.RemoteMod{}, &models.NoCompatibleFileError{Platform: models.CURSEFORGE, ProjectID: projectID}
	}

	return models.RemoteMod{
		Name:        project.Name,
		FileName:    selected.FileName,
		ReleaseDate: formatReleaseDate(selected.FileDate),
		Hash:        hash,
		DownloadURL: selected.DownloadURL,
	}, nil
}

func formatReleaseDate(date time.Time) string {
	return date.Format(time.RFC3339)
}

func filterCurseforgeFiles(files []File, opts models.FetchOptions, targetVersion string) []File {
	filtered := make([]File, 0, len(files))
	for _, file := range files {
		if opts.FixedVersion != "" && !strings.EqualFold(file.FileName, opts.FixedVersion) {
			continue
		}
		if !fileHasVersion(file, targetVersion) {
			continue
		}
		releaseType, ok := curseforgeReleaseType(file.ReleaseType)
		if !ok || !containsReleaseType(opts.AllowedReleaseTypes, releaseType) {
			continue
		}
		if !isAcceptableStatus(file.FileStatus) || !file.IsAvailable {
			continue
		}
		filtered = append(filtered, file)
	}
	return filtered
}

func containsReleaseType(allowed []models.ReleaseType, candidate models.ReleaseType) bool {
	for _, releaseType := range allowed {
		if releaseType == candidate {
			return true
		}
	}
	return false
}

func fileHasVersion(file File, version string) bool {
	for _, gameVersion := range file.SortableGameVersions {
		if strings.EqualFold(gameVersion.GameVersionName, version) {
			return true
		}
	}
	return false
}

func curseforgeReleaseType(fileType FileReleaseType) (models.ReleaseType, bool) {
	switch fileType {
	case Release:
		return models.Release, true
	case Beta:
		return models.Beta, true
	case Alpha:
		return models.Alpha, true
	default:
		return "", false
	}
}

func isAcceptableStatus(status FileStatus) bool {
	return status == Approved || status == Released
}

func modLoaderTypeFromLoader(loader models.Loader) (ModLoaderType, error) {
	switch loader {
	case models.FABRIC:
		return Fabric, nil
	case models.QUILT:
		return Quilt, nil
	case models.FORGE:
		return Forge, nil
	case models.CAULDRON:
		return Cauldron, nil
	case models.LITELOADER:
		return LiteLoader, nil
	case models.NEOFORGE:
		return NeoForge, nil
	default:
		return 0, fmt.Errorf("unsupported loader for curseforge: %s", loader)
	}
}

func getCurseforgeHash(hashes []FileHash) (string, error) {
	for _, hash := range hashes {
		if hash.Algorithm == SHA1 {
			if hash.Hash == "" {
				return "", errors.New("empty hash")
			}
			return hash.Hash, nil
		}
	}
	return "", errors.New("hash not found")
}
