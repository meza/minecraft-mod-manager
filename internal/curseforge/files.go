package curseforge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/urlbuilder"
	"github.com/pkg/errors"

	"go.opentelemetry.io/otel/attribute"
)

type getFilesResponse struct {
	Data       []File     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

const defaultFilesPageSize = 50

// FileListFilter describes filters for the CurseForge files list endpoint.
type FileListFilter struct {
	GameVersion   string
	ModLoaderType ModLoaderType
	PageSize      int
}

func (filter FileListFilter) pageSize() int {
	if filter.PageSize <= 0 {
		return defaultFilesPageSize
	}
	return filter.PageSize
}

type getFingerprintsRequest struct {
	Fingerprints []uint32 `json:"fingerprints"`
}

type fingerprintMatch struct {
	ProjectID   int    `json:"id"`
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

func getPaginatedFilesForProject(ctx context.Context, projectID int, client httpclient.Doer, cursor int) (filesResponse *getFilesResponse, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.curseforge.project.files.list",
		perf.WithAttributes(
			attribute.Int("project_id", projectID),
			attribute.Int("cursor", cursor),
		),
	)
	defer span.End()

	request, cancel, err := buildPaginatedFilesRequest(ctx, projectID, cursor)
	if err != nil {
		return nil, err
	}
	defer cancel()

	response, err := client.Do(request)
	if err != nil {
		if httpclient.IsTimeoutError(err) {
			return nil, httpclient.WrapTimeoutError(err)
		}
		return nil, globalerrors.ProjectAPIErrorWrap(err, strconv.Itoa(projectID), models.CURSEFORGE)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalerrors.ProjectNotFoundError{
			ProjectID: strconv.Itoa(projectID),
			Platform:  models.CURSEFORGE,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, globalerrors.ProjectAPIErrorWrap(httpclient.NewResponseError(response), strconv.Itoa(projectID), models.CURSEFORGE)
	}

	decodedFilesResponse, err := decodeFilesResponse(response)
	if err != nil {
		return nil, globalerrors.ProjectAPIErrorWrap(errors.Wrap(err, "failed to decode response body"), strconv.Itoa(projectID), models.CURSEFORGE)
	}

	return decodedFilesResponse, nil
}

func getPaginatedFilesForProjectWithFilters(ctx context.Context, projectID int, filter FileListFilter, client httpclient.Doer, cursor int) (filesResponse *getFilesResponse, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.curseforge.project.files.list",
		perf.WithAttributes(
			attribute.Int("project_id", projectID),
			attribute.Int("cursor", cursor),
		),
	)
	defer span.End()

	request, cancel, err := buildPaginatedFilesRequestWithFilters(ctx, projectID, cursor, filter)
	if err != nil {
		return nil, err
	}
	defer cancel()

	response, err := client.Do(request)
	if err != nil {
		if httpclient.IsTimeoutError(err) {
			return nil, httpclient.WrapTimeoutError(err)
		}
		return nil, globalerrors.ProjectAPIErrorWrap(err, strconv.Itoa(projectID), models.CURSEFORGE)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalerrors.ProjectNotFoundError{
			ProjectID: strconv.Itoa(projectID),
			Platform:  models.CURSEFORGE,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, globalerrors.ProjectAPIErrorWrap(httpclient.NewResponseError(response), strconv.Itoa(projectID), models.CURSEFORGE)
	}

	decodedFilesResponse, err := decodeFilesResponse(response)
	if err != nil {
		return nil, globalerrors.ProjectAPIErrorWrap(errors.Wrap(err, "failed to decode response body"), strconv.Itoa(projectID), models.CURSEFORGE)
	}

	return decodedFilesResponse, nil
}

func buildPaginatedFilesRequest(ctx context.Context, projectID int, cursor int) (*http.Request, func(), error) {
	requestURL, err := buildPaginatedFilesURL(projectID, cursor)
	if err != nil {
		return nil, func() {}, err
	}
	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	request, err := newRequestWithContext(timeoutCtx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		cancel()
		return nil, func() {}, err
	}
	return request, cancel, nil
}

func buildPaginatedFilesRequestWithFilters(ctx context.Context, projectID int, cursor int, filter FileListFilter) (*http.Request, func(), error) {
	requestURL, err := buildPaginatedFilesURLWithFilters(projectID, cursor, filter)
	if err != nil {
		return nil, func() {}, err
	}
	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	request, err := newRequestWithContext(timeoutCtx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		cancel()
		return nil, func() {}, err
	}
	return request, cancel, nil
}

func decodeFilesResponse(response *http.Response) (*getFilesResponse, error) {
	var decodedFilesResponse getFilesResponse
	if err := json.NewDecoder(response.Body).Decode(&decodedFilesResponse); err != nil {
		return nil, err
	}
	return &decodedFilesResponse, nil
}

func GetFilesForProject(ctx context.Context, projectID int, client httpclient.Doer) ([]File, error) {
	var files []File
	cursor := 0
	for {
		filesResponse, err := getPaginatedFilesForProject(ctx, projectID, client, cursor)
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

func GetFilesForProjectWithFilters(ctx context.Context, projectID int, filter FileListFilter, client httpclient.Doer) ([]File, error) {
	var files []File
	cursor := 0
	for {
		filesResponse, err := getPaginatedFilesForProjectWithFilters(ctx, projectID, filter, client, cursor)
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

// GetFingerprintsMatches resolves CurseForge fingerprints into matches/unmatched sets.
// It expects CurseForge fingerprint hashes (uint32) and returns both matched files
// and the unmatched list for follow-up handling.
//
// It returns FingerprintAPIError when the request fails, the status is non-200, or
// the response cannot be decoded.
//
// Example:
//
//	result, err := curseforge.GetFingerprintsMatches(ctx, []uint32{fingerprint}, client)
func GetFingerprintsMatches(ctx context.Context, fingerprints []uint32, client httpclient.Doer) (result *FingerprintResult, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.curseforge.fingerprints.match", perf.WithAttributes(attribute.Int("fingerprints_count", len(fingerprints))))
	defer span.End()

	request, cancel, err := newFingerprintMatchRequest(ctx, fingerprints)
	if err != nil {
		return nil, err
	}
	defer cancel()

	response, err := doFingerprintMatchRequest(client, request, fingerprints)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	fingerprintsResponse, err := decodeFingerprintMatchesResponse(response, fingerprints)
	if err != nil {
		return nil, err
	}

	result, err = buildFingerprintResult(fingerprintsResponse, fingerprints)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func newFingerprintMatchRequest(ctx context.Context, fingerprints []uint32) (*http.Request, func(), error) {
	requestURL, err := buildFingerprintMatchURL()
	if err != nil {
		return nil, func() {}, err
	}

	body, err := marshalJSON(getFingerprintsRequest{Fingerprints: fingerprints})
	if err != nil {
		return nil, func() {}, err
	}

	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	request, err := newRequestWithContext(timeoutCtx, http.MethodPost, requestURL.String(), bytes.NewBuffer(body))
	if err != nil {
		cancel()
		return nil, func() {}, err
	}
	request.Header.Add("Content-Type", "application/json")
	return request, cancel, nil
}

func buildPaginatedFilesURL(projectID int, cursor int) (*url.URL, error) {
	baseURL, err := parseURL(GetBaseURL())
	if err != nil {
		return nil, err
	}
	requestURL := urlbuilder.JoinEscapedPath(baseURL, "mods", strconv.Itoa(projectID), "files")
	query := url.Values{}
	query.Set("index", fmt.Sprintf("%d", cursor))
	requestURL.RawQuery = query.Encode()
	return requestURL, nil
}

func buildPaginatedFilesURLWithFilters(projectID int, cursor int, filter FileListFilter) (*url.URL, error) {
	baseURL, err := parseURL(GetBaseURL())
	if err != nil {
		return nil, err
	}
	requestURL := urlbuilder.JoinEscapedPath(baseURL, "mods", strconv.Itoa(projectID), "files")
	query := url.Values{}
	query.Set("index", fmt.Sprintf("%d", cursor))
	query.Set("pageSize", fmt.Sprintf("%d", filter.pageSize()))
	if filter.GameVersion != "" {
		query.Set("gameVersion", filter.GameVersion)
	}
	if filter.ModLoaderType != Any && filter.GameVersion != "" {
		query.Set("modLoaderType", fmt.Sprintf("%d", filter.ModLoaderType))
	}
	requestURL.RawQuery = query.Encode()
	return requestURL, nil
}

func buildFingerprintMatchURL() (*url.URL, error) {
	baseURL, err := parseURL(GetBaseURL())
	if err != nil {
		return nil, err
	}
	requestURL := urlbuilder.JoinEscapedPath(baseURL, "fingerprints", fmt.Sprintf("%d", Minecraft))
	return requestURL, nil
}

func doFingerprintMatchRequest(client httpclient.Doer, request *http.Request, fingerprints []uint32) (*http.Response, error) {
	response, err := client.Do(request)
	if err != nil {
		if httpclient.IsTimeoutError(err) {
			return nil, httpclient.WrapTimeoutError(err)
		}
		return nil, &FingerprintAPIError{
			Lookup: fingerprints,
			Err:    err,
		}
	}
	return response, nil
}

func decodeFingerprintMatchesResponse(response *http.Response, fingerprints []uint32) (*getFingerprintsMatchesResponse, error) {
	if response.StatusCode != http.StatusOK {
		return nil, &FingerprintAPIError{
			Lookup: fingerprints,
			Err:    httpclient.NewResponseError(response),
		}
	}

	var fingerprintsResponse getFingerprintsMatchesResponse
	if err := json.NewDecoder(response.Body).Decode(&fingerprintsResponse); err != nil {
		return nil, &FingerprintAPIError{
			Lookup: fingerprints,
			Err:    errors.Wrap(err, "failed to decode response body"),
		}
	}
	return &fingerprintsResponse, nil
}

func buildFingerprintResult(response *getFingerprintsMatchesResponse, fingerprints []uint32) (*FingerprintResult, error) {
	result := &FingerprintResult{
		Matches:   make([]File, 0),
		Unmatched: make([]uint32, 0),
	}

	for _, item := range response.Data.ExactMatches {
		file := item.File
		if file.Fingerprint == 0 && file.FileFingerprint != 0 {
			file.Fingerprint = file.FileFingerprint
		}
		result.Matches = append(result.Matches, file)
	}

	unmatched, decodeErr := decodeUnmatchedFingerprints(response.Data.UnmatchedFingerprints)
	if decodeErr != nil {
		return nil, &FingerprintAPIError{
			Lookup: fingerprints,
			Err:    errors.Wrap(decodeErr, "failed to decode unmatchedFingerprints"),
		}
	}
	result.Unmatched = append(result.Unmatched, unmatched...)
	return result, nil
}

func decodeUnmatchedFingerprints(raw json.RawMessage) ([]uint32, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var list []uint32
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

func parseUnmatchedMapKeys[V any](m map[string]V) ([]uint32, error) {
	out := make([]uint32, 0, len(m))
	for key := range m {
		value, err := strconv.ParseUint(key, 10, 32)
		if err != nil {
			continue
		}
		out = append(out, uint32(value))
	}
	return out, nil
}
