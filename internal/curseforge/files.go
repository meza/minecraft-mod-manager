package curseforge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

type getFilesResponse struct {
	Data       []File     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type getFingerprintsRequest struct {
	Fingerprints []int `json:"fingerprints"`
}

type fingerprintMatch struct {
	ProjectId   int    `json:"id"`
	File        File   `json:"file"`
	LatestFiles []File `json:"latestFiles"`
}

type fingerprintsMatchResult struct {
	ExactMatches             []fingerprintMatch `json:"exactMatches"`
	ExactFingerprints        []int              `json:"exactFingerprints"`
	PartialMatches           []fingerprintMatch `json:"partialMatches"`
	PartialMatchFingerprints []int              `json:"partialMatchFingerprints"`
	UnmatchedFingerprints    []int              `json:"unmatchedFingerprints"`
	InstalledFingerprints    []int              `json:"installedFingerprints"`
}

type getFingerprintsMatchesResponse struct {
	Data fingerprintsMatchResult `json:"data"`
}

func getPaginatedFilesForProject(projectId int, client httpClient.Doer, cursor int) (*getFilesResponse, error) {
	region := perf.StartRegionWithDetails("api.curseforge.project.files.list", &perf.PerformanceDetails{
		"project_id": projectId,
		"cursor":     cursor,
	})
	defer region.End()

	url := fmt.Sprintf("%s/mods/%d/files?index=%d", GetBaseUrl(), projectId, cursor)
	request, _ := http.NewRequest(http.MethodGet, url, nil)

	response, err := client.Do(request)
	if err != nil {
		return nil, globalErrors.ProjectApiErrorWrap(err, strconv.Itoa(projectId), models.CURSEFORGE)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalErrors.ProjectNotFoundError{
			ProjectID: strconv.Itoa(projectId),
			Platform:  models.CURSEFORGE,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, globalErrors.ProjectApiErrorWrap(errors.Errorf("unexpected status code: %d", response.StatusCode), strconv.Itoa(projectId), models.CURSEFORGE)
	}

	var filesResponse getFilesResponse
	err = json.NewDecoder(response.Body).Decode(&filesResponse)
	if err != nil {
		return nil, globalErrors.ProjectApiErrorWrap(errors.Wrap(err, "failed to decode response body"), strconv.Itoa(projectId), models.CURSEFORGE)
	}

	return &filesResponse, nil
}

func GetFilesForProject(projectId int, client httpClient.Doer) ([]File, error) {
	var files []File
	cursor := 0
	for {
		filesResponse, err := getPaginatedFilesForProject(projectId, client, cursor)
		if err != nil {
			return nil, err
		}

		files = append(files, filesResponse.Data...)
		if (cursor + filesResponse.Pagination.ResultCount) >= filesResponse.Pagination.TotalCount {
			break
		}

		cursor += filesResponse.Pagination.ResultCount
	}

	return files, nil
}

func GetFingerprintsMatches(fingerprints []int, client httpClient.Doer) (*FingerprintResult, error) {
	region := perf.StartRegionWithDetails("api.curseforge.fingerprints.match", &perf.PerformanceDetails{
		"fingerprints": fingerprints,
	})
	defer region.End()

	gameId := Minecraft

	url := fmt.Sprintf("%s/fingerprints/%d", GetBaseUrl(), gameId)

	body, _ := json.Marshal(getFingerprintsRequest{Fingerprints: fingerprints})
	request, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	request.Header.Add("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return nil, &FingerprintApiError{
			Lookup: fingerprints,
			Err:    err,
		}
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, &FingerprintApiError{
			Lookup: fingerprints,
			Err:    errors.Errorf("unexpected status code: %d", response.StatusCode),
		}
	}

	var fingerprintsResponse getFingerprintsMatchesResponse
	err = json.NewDecoder(response.Body).Decode(&fingerprintsResponse)
	if err != nil {
		return nil, &FingerprintApiError{
			Lookup: fingerprints,
			Err:    errors.Wrap(err, "failed to decode response body"),
		}
	}

	result := &FingerprintResult{
		Matches:   make([]File, 0),
		Unmatched: make([]int, 0),
	}

	for _, item := range fingerprintsResponse.Data.ExactMatches {
		result.Matches = append(result.Matches, item.File)
	}

	for _, item := range fingerprintsResponse.Data.UnmatchedFingerprints {
		result.Unmatched = append(result.Unmatched, item)
	}

	return result, nil

}
