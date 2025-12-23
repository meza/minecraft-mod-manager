// Package platform coordinates platform-specific integrations.
package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

type curseforgeDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type curseforgeFilesResponse struct {
	Data []curseforge.File `json:"data"`
}

var newRequestWithContext = http.NewRequestWithContext

func fetchCurseforge(ctx context.Context, projectID string, opts FetchOptions, client curseforgeDoer) (RemoteMod, error) {
	curseforgeClient := curseforge.NewClient(client)

	project, err := curseforge.GetProject(ctx, projectID, curseforgeClient)
	if err != nil {
		return RemoteMod{}, mapProjectNotFound(models.CURSEFORGE, projectID, err)
	}

	modLoader, err := curseforgeLoaderFromLoader(opts.Loader)
	if err != nil {
		return RemoteMod{}, err
	}

	currentVersion := opts.GameVersion

	for {
		files, filesErr := fetchCurseforgeFiles(ctx, projectID, currentVersion, modLoader, curseforgeClient)
		if filesErr != nil {
			return RemoteMod{}, mapProjectNotFound(models.CURSEFORGE, projectID, filesErr)
		}

		candidates := filterCurseforgeFiles(files, opts, currentVersion)
		if len(candidates) == 0 {
			next, canGoDown := nextVersionDown(currentVersion)
			if opts.AllowFallback && canGoDown {
				currentVersion = next
				continue
			}
			return RemoteMod{}, &NoCompatibleFileError{Platform: models.CURSEFORGE, ProjectID: projectID}
		}

		sort.SliceStable(candidates, func(i, j int) bool {
			return candidates[i].FileDate.After(candidates[j].FileDate)
		})

		selected := candidates[0]
		if selected.DownloadURL == "" {
			return RemoteMod{}, &NoCompatibleFileError{Platform: models.CURSEFORGE, ProjectID: projectID}
		}

		hash, hashErr := getCurseforgeHash(selected.Hashes, curseforge.SHA1)
		if hashErr != nil {
			return RemoteMod{}, &NoCompatibleFileError{Platform: models.CURSEFORGE, ProjectID: projectID}
		}

		return RemoteMod{
			Name:        project.Name,
			FileName:    selected.FileName,
			ReleaseDate: formatTime(selected.FileDate),
			Hash:        hash,
			DownloadURL: selected.DownloadURL,
		}, nil
	}
}

func fetchCurseforgeFiles(ctx context.Context, projectID string, gameVersion string, loader curseforge.ModLoaderType, client curseforgeDoer) (files []curseforge.File, returnErr error) {
	url := fmt.Sprintf("%s/mods/%s/files?gameVersion=%s&modLoaderType=%d", curseforge.GetBaseURL(), projectID, gameVersion, loader)
	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		if httpclient.IsTimeoutError(err) {
			return nil, httpclient.WrapTimeoutError(err)
		}
		return nil, err
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalerrors.ProjectNotFoundError{ProjectID: projectID, Platform: models.CURSEFORGE}
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var filesResponse curseforgeFilesResponse
	if decodeErr := json.NewDecoder(response.Body).Decode(&filesResponse); decodeErr != nil {
		return nil, decodeErr
	}

	return filesResponse.Data, nil
}

func filterCurseforgeFiles(files []curseforge.File, opts FetchOptions, targetVersion string) []curseforge.File {
	filtered := make([]curseforge.File, 0, len(files))
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

func fileHasVersion(file curseforge.File, version string) bool {
	for _, gv := range file.SortableGameVersions {
		if strings.EqualFold(gv.GameVersionName, version) {
			return true
		}
	}
	return false
}

func curseforgeReleaseType(fileType curseforge.FileReleaseType) (models.ReleaseType, bool) {
	switch fileType {
	case curseforge.Release:
		return models.Release, true
	case curseforge.Beta:
		return models.Beta, true
	case curseforge.Alpha:
		return models.Alpha, true
	default:
		return "", false
	}
}

func isAcceptableStatus(status curseforge.FileStatus) bool {
	return status == curseforge.Approved || status == curseforge.Released
}

func curseforgeLoaderFromLoader(loader models.Loader) (curseforge.ModLoaderType, error) {
	switch loader {
	case models.FABRIC:
		return curseforge.Fabric, nil
	case models.QUILT:
		return curseforge.Quilt, nil
	case models.FORGE:
		return curseforge.Forge, nil
	case models.CAULDRON:
		return curseforge.Cauldron, nil
	case models.LITELOADER:
		return curseforge.LiteLoader, nil
	case models.NEOFORGE:
		return curseforge.NeoForge, nil
	default:
		return 0, fmt.Errorf("unsupported loader for curseforge: %s", loader)
	}
}

func getCurseforgeHash(hashes []curseforge.FileHash, algorithm curseforge.FileHashAlgorithm) (string, error) {
	for _, hash := range hashes {
		if hash.Algorithm == algorithm {
			if hash.Hash == "" {
				return "", fmt.Errorf("empty hash")
			}
			return hash.Hash, nil
		}
	}
	return "", fmt.Errorf("hash not found")
}
