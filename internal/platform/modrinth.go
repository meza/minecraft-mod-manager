package platform

import (
	"context"
	"net/http"
	"sort"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
)

func fetchModrinth(ctx context.Context, projectID string, opts FetchOptions, client modrinthDoer) (RemoteMod, error) {
	modrinthClient := modrinth.NewClient(client)

	project, err := modrinth.GetProject(ctx, projectID, modrinthClient)
	if err != nil {
		return RemoteMod{}, mapProjectNotFound(models.MODRINTH, projectID, err)
	}

	currentVersion := opts.GameVersion

	for {
		versions, versionErr := modrinth.GetVersionsForProject(ctx, &modrinth.VersionLookup{
			ProjectId:    projectID,
			Loaders:      []models.Loader{opts.Loader},
			GameVersions: []string{currentVersion},
		}, modrinthClient)
		if versionErr != nil {
			return RemoteMod{}, mapProjectNotFound(models.MODRINTH, projectID, versionErr)
		}

		candidates := filterModrinthVersions(versions, opts, currentVersion)
		if len(candidates) == 0 {
			next, canGoDown := nextVersionDown(currentVersion)
			if opts.AllowFallback && canGoDown {
				currentVersion = next
				continue
			}
			return RemoteMod{}, &NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: projectID}
		}

		sort.SliceStable(candidates, func(i, j int) bool {
			return candidates[i].DatePublished.After(candidates[j].DatePublished)
		})

		selectedVersion := candidates[0]
		if len(selectedVersion.Files) == 0 || selectedVersion.Files[0].Hashes.Sha1 == "" || selectedVersion.Files[0].Url == "" {
			return RemoteMod{}, &NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: projectID}
		}

		file := selectedVersion.Files[0]

		return RemoteMod{
			Name:        project.Title,
			FileName:    file.FileName,
			ReleaseDate: formatTime(selectedVersion.DatePublished),
			Hash:        file.Hashes.Sha1,
			DownloadURL: file.Url,
		}, nil
	}
}

func filterModrinthVersions(versions modrinth.Versions, opts FetchOptions, targetVersion string) modrinth.Versions {
	var filtered modrinth.Versions

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

type modrinthDoer interface {
	Do(req *http.Request) (*http.Response, error)
}
