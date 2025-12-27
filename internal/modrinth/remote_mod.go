package modrinth

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

// FetchRemoteMod selects the newest compatible Modrinth file and returns it as a domain RemoteMod.
func FetchRemoteMod(ctx context.Context, projectID string, opts models.FetchOptions, client httpclient.Doer) (models.RemoteMod, error) {
	modrinthClient := NewClient(client)

	project, err := GetProject(ctx, projectID, modrinthClient)
	if err != nil {
		return models.RemoteMod{}, mapProjectNotFoundError(projectID, err)
	}

	currentVersion := opts.GameVersion

	for {
		versions, versionErr := GetVersionsForProject(ctx, &VersionLookup{
			ProjectID:    projectID,
			Loaders:      []models.Loader{opts.Loader},
			GameVersions: []string{currentVersion},
		}, modrinthClient)
		if versionErr != nil {
			return models.RemoteMod{}, mapProjectNotFoundError(projectID, versionErr)
		}

		candidates := filterVersions(versions, opts, currentVersion)
		if len(candidates) == 0 {
			nextVersion, canFallback, err := fallbackVersion(ctx, currentVersion, opts, client)
			if err != nil {
				return models.RemoteMod{}, err
			}
			if canFallback {
				currentVersion = nextVersion
				continue
			}
			return models.RemoteMod{}, &models.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: projectID}
		}

		sort.SliceStable(candidates, func(leftIndex int, rightIndex int) bool {
			return candidates[leftIndex].DatePublished.After(candidates[rightIndex].DatePublished)
		})

		return buildRemoteMod(project, candidates[0], projectID)
	}
}

func mapProjectNotFoundError(projectID string, err error) error {
	var notFound *globalerrors.ProjectNotFoundError
	if errors.As(err, &notFound) {
		return &models.ModNotFoundError{Platform: models.MODRINTH, ProjectID: projectID}
	}
	return err
}

func filterVersions(versions Versions, opts models.FetchOptions, targetVersion string) Versions {
	var filtered Versions

	if opts.FixedVersion != "" {
		for _, version := range versions {
			if version.VersionNumber == opts.FixedVersion {
				filtered = append(filtered, version)
			}
		}
		return filtered
	}

	for _, version := range versions {
		if !containsReleaseType(opts.AllowedReleaseTypes, version.Type) {
			continue
		}
		if !containsGameVersion(version.GameVersions, targetVersion) {
			continue
		}
		filtered = append(filtered, version)
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

func containsGameVersion(gameVersions []string, target string) bool {
	for _, version := range gameVersions {
		if version == target {
			return true
		}
	}
	return false
}

func selectPrimaryFile(files []VersionFile) (VersionFile, bool) {
	if len(files) == 0 {
		return VersionFile{}, false
	}

	for _, file := range files {
		if file.Primary {
			return file, true
		}
	}

	return files[0], true
}

func buildRemoteMod(project *Project, selectedVersion Version, projectID string) (models.RemoteMod, error) {
	selectedFile, ok := selectPrimaryFile(selectedVersion.Files)
	if !ok || selectedFile.Hashes.SHA1 == "" || selectedFile.URL == "" {
		return models.RemoteMod{}, &models.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: projectID}
	}

	return models.RemoteMod{
		Name:        project.Title,
		FileName:    selectedFile.FileName,
		ReleaseDate: formatReleaseDate(selectedVersion.DatePublished),
		Hash:        selectedFile.Hashes.SHA1,
		DownloadURL: selectedFile.URL,
	}, nil
}

func fallbackVersion(ctx context.Context, currentVersion string, opts models.FetchOptions, client httpclient.Doer) (string, bool, error) {
	if !opts.AllowFallback {
		return "", false, nil
	}
	return minecraft.NextPatchDown(ctx, currentVersion, client)
}

func formatReleaseDate(date time.Time) string {
	return date.Format(time.RFC3339)
}
