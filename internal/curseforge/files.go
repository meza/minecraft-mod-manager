package curseforge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/pkg/errors"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
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
	ExactFingerprints        json.RawMessage    `json:"exactFingerprints"`
	PartialMatches           []fingerprintMatch `json:"partialMatches"`
	PartialMatchFingerprints json.RawMessage    `json:"partialMatchFingerprints"`
	UnmatchedFingerprints    json.RawMessage    `json:"unmatchedFingerprints"`
	InstalledFingerprints    json.RawMessage    `json:"installedFingerprints"`
}

type getFingerprintsMatchesResponse struct {
	Data fingerprintsMatchResult `json:"data"`
}

func getPaginatedFilesForProject(ctx context.Context, projectId int, client httpClient.Doer, cursor int) (filesResponse *getFilesResponse, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.curseforge.project.files.list",
		perf.WithAttributes(
			attribute.Int("project_id", projectId),
			attribute.Int("cursor", cursor),
		),
	)
	defer span.End()

	url := fmt.Sprintf("%s/mods/%d/files?index=%d", GetBaseUrl(), projectId, cursor)
	timeoutCtx, cancel := httpClient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		if httpClient.IsTimeoutError(err) {
			return nil, httpClient.WrapTimeoutError(err)
		}
		return nil, globalErrors.ProjectApiErrorWrap(err, strconv.Itoa(projectId), models.CURSEFORGE)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalErrors.ProjectNotFoundError{
			ProjectID: strconv.Itoa(projectId),
			Platform:  models.CURSEFORGE,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, globalErrors.ProjectApiErrorWrap(errors.Errorf("unexpected status code: %d", response.StatusCode), strconv.Itoa(projectId), models.CURSEFORGE)
	}

	var decodedFilesResponse getFilesResponse
	err = json.NewDecoder(response.Body).Decode(&decodedFilesResponse)
	if err != nil {
		return nil, globalErrors.ProjectApiErrorWrap(errors.Wrap(err, "failed to decode response body"), strconv.Itoa(projectId), models.CURSEFORGE)
	}

	return &decodedFilesResponse, nil
}

func GetFilesForProject(ctx context.Context, projectId int, client httpClient.Doer) ([]File, error) {
	var files []File
	cursor := 0
	for {
		filesResponse, err := getPaginatedFilesForProject(ctx, projectId, client, cursor)
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

func GetFingerprintsMatches(ctx context.Context, fingerprints []int, client httpClient.Doer) (result *FingerprintResult, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.curseforge.fingerprints.match", perf.WithAttributes(attribute.Int("fingerprints_count", len(fingerprints))))
	defer span.End()

	gameId := Minecraft

	url := fmt.Sprintf("%s/fingerprints/%d", GetBaseUrl(), gameId)

	body, err := marshalJSON(getFingerprintsRequest{Fingerprints: fingerprints})
	if err != nil {
		return nil, err
	}
	timeoutCtx, cancel := httpClient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	request.Header.Add("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		if httpClient.IsTimeoutError(err) {
			return nil, httpClient.WrapTimeoutError(err)
		}
		return nil, &FingerprintApiError{
			Lookup: fingerprints,
			Err:    err,
		}
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

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

	result = &FingerprintResult{
		Matches:   make([]File, 0),
		Unmatched: make([]int, 0),
	}

	for _, item := range fingerprintsResponse.Data.ExactMatches {
		file := item.File
		if file.Fingerprint == 0 && file.FileFingerprint != 0 {
			file.Fingerprint = file.FileFingerprint
		}
		result.Matches = append(result.Matches, file)
	}

	unmatched, decodeErr := decodeUnmatchedFingerprints(fingerprintsResponse.Data.UnmatchedFingerprints)
	if decodeErr != nil {
		return nil, &FingerprintApiError{
			Lookup: fingerprints,
			Err:    errors.Wrap(decodeErr, "failed to decode unmatchedFingerprints"),
		}
	}
	result.Unmatched = append(result.Unmatched, unmatched...)

	return result, nil

}

func decodeUnmatchedFingerprints(raw json.RawMessage) ([]int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var list []int
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}

	var asBoolMap map[string]bool
	if err := json.Unmarshal(raw, &asBoolMap); err == nil {
		return parseUnmatchedMapKeys(asBoolMap)
	}

	var asAnyMap map[string]any
	if err := json.Unmarshal(raw, &asAnyMap); err == nil {
		return parseUnmatchedMapKeys(asAnyMap)
	}

	return nil, errors.Errorf("unsupported type: %s", string(raw))

}

func parseUnmatchedMapKeys[V any](m map[string]V) ([]int, error) {
	out := make([]int, 0, len(m))
	for key := range m {
		value, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		out = append(out, value)
	}
	return out, nil
}
