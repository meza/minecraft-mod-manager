package modrinth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/urlbuilder"

	"go.opentelemetry.io/otel/attribute"
)

type DependencyType string
type VersionStatus string
type VersionAlgorithm string

const (
	Listed    VersionStatus = "listed"
	Archived  VersionStatus = "archived"
	Draft     VersionStatus = "draft"
	Unlisted  VersionStatus = "unlisted"
	Scheduled VersionStatus = "scheduled"
	Unknown   VersionStatus = "unknown"
)
const (
	RequiredDependency DependencyType = "required"
	OptionalDependency DependencyType = "optional"
)

const (
	SHA1   VersionAlgorithm = "sha1"
	Sha512 VersionAlgorithm = "sha512"
)

type VersionFileHash struct {
	SHA1   string `json:"sha1"`
	Sha512 string `json:"sha512"`
}

type VersionFile struct {
	FileName string          `json:"filename"`
	Hashes   VersionFileHash `json:"hashes"`
	Primary  bool            `json:"primary"`
	Size     int64           `json:"size"`
	URL      string          `json:"url"`
}

type VersionDependency struct {
	FileName  string         `json:"file_name"`
	ProjectID string         `json:"project_id"`
	Type      DependencyType `json:"type"`
	VersionID string         `json:"version_id"`
}

type Version struct {
	Changelog     string              `json:"changelog"`
	DatePublished time.Time           `json:"date_published"`
	Dependencies  []VersionDependency `json:"dependencies"`
	Files         []VersionFile       `json:"files"`
	GameVersions  []string            `json:"game_versions"`
	Loaders       []models.Loader     `json:"loaders"`
	Name          string              `json:"name"`
	ProjectID     string              `json:"project_id"`
	Status        VersionStatus       `json:"status"`
	Type          models.ReleaseType  `json:"version_type"`
	VersionID     string              `json:"id"`
	VersionNumber string              `json:"version_number"`
}

type Versions []Version

type VersionLookup struct {
	ProjectID    string          `json:"project_id"`
	Loaders      []models.Loader `json:"loaders"`
	GameVersions []string        `json:"game_versions"`
}

type VersionHashLookup struct {
	hash      string
	algorithm VersionAlgorithm
}

func NewVersionHashLookup(hash string, algorithm VersionAlgorithm) *VersionHashLookup {
	return &VersionHashLookup{
		hash:      hash,
		algorithm: algorithm,
	}
}

// GetVersionsForProject fetches Modrinth versions for a project using the lookup filters.
// The lookup is sent to the API as-is; this function does not apply additional
// client-side filtering beyond decoding the response.
//
// It returns ProjectNotFoundError when the project does not exist and ProjectAPIError
// when the request fails, the status is non-200, or the response cannot be decoded.
//
// Example:
//
//	versions, err := modrinth.GetVersionsForProject(ctx, &modrinth.VersionLookup{
//	  ProjectID:    "AANobbMI",
//	  Loaders:      []models.Loader{models.FABRIC},
//	  GameVersions: []string{"1.20.1"},
//	}, client)
func GetVersionsForProject(ctx context.Context, lookup *VersionLookup, client httpclient.Doer) (versions Versions, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.modrinth.version.list", perf.WithAttributes(attribute.String("project_id", lookup.ProjectID)))
	defer span.End()

	baseURL, err := buildVersionListURL(lookup)
	if err != nil {
		return nil, globalerrors.ProjectAPIErrorWrap(fmt.Errorf("failed to build version list URL: %w", err), lookup.ProjectID, models.MODRINTH)
	}

	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, "GET", baseURL.String(), nil)
	if err != nil {
		return nil, globalerrors.ProjectAPIErrorWrap(fmt.Errorf("failed to build version list request: %w", err), lookup.ProjectID, models.MODRINTH)
	}
	response, err := client.Do(request)
	if err != nil {
		if httpclient.IsTimeoutError(err) {
			return nil, httpclient.WrapTimeoutError(err)
		}
		return nil, globalerrors.ProjectAPIErrorWrap(err, lookup.ProjectID, models.MODRINTH)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalerrors.ProjectNotFoundError{
			ProjectID: lookup.ProjectID,
			Platform:  models.MODRINTH,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, globalerrors.ProjectAPIErrorWrap(httpclient.NewResponseError(response), lookup.ProjectID, models.MODRINTH)
	}

	if err := json.NewDecoder(response.Body).Decode(&versions); err != nil {
		return nil, globalerrors.ProjectAPIErrorWrap(fmt.Errorf("failed to decode response body: %w", err), lookup.ProjectID, models.MODRINTH)
	}
	return versions, nil
}

func buildVersionListURL(lookup *VersionLookup) (*url.URL, error) {
	gameVersionsJSON, err := marshalJSON(lookup.GameVersions)
	if err != nil {
		return nil, err
	}
	loadersJSON, err := marshalJSON(lookup.Loaders)
	if err != nil {
		return nil, err
	}

	baseURL, err := buildVersionListBaseURL(lookup.ProjectID)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("game_versions", string(gameVersionsJSON))
	query.Set("loaders", string(loadersJSON))
	baseURL.RawQuery = query.Encode()

	return baseURL, nil
}

// GetVersionForHash fetches a Modrinth version by hash and algorithm.
// Use NewVersionHashLookup to construct the lookup.
//
// It returns VersionNotFoundError only when the API responds with 404 and
// VersionAPIError when the request fails, the status is non-200, or the response
// cannot be decoded.
//
// Example:
//
//	lookup := modrinth.NewVersionHashLookup(sha1, modrinth.SHA1)
//	version, err := modrinth.GetVersionForHash(ctx, lookup, client)
func GetVersionForHash(ctx context.Context, lookup *VersionHashLookup, client httpclient.Doer) (version *Version, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.modrinth.version_file.get", perf.WithAttributes(attribute.String("hash", lookup.hash)))
	defer span.End()

	requestURL, err := buildVersionFileURL(lookup)
	if err != nil {
		return nil, VersionAPIErrorWrap(fmt.Errorf("failed to build version file URL: %w", err), *lookup)
	}

	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, "GET", requestURL.String(), nil)
	if err != nil {
		return nil, VersionAPIErrorWrap(fmt.Errorf("failed to build version file request: %w", err), *lookup)
	}
	response, err := client.Do(request)
	if err != nil {
		if httpclient.IsTimeoutError(err) {
			return nil, httpclient.WrapTimeoutError(err)
		}
		return nil, VersionAPIErrorWrap(err, *lookup)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	if response.StatusCode == http.StatusNotFound {
		return nil, &VersionNotFoundError{
			Lookup: *lookup,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, VersionAPIErrorWrap(httpclient.NewResponseError(response), *lookup)
	}

	version = &Version{}
	if err := json.NewDecoder(response.Body).Decode(version); err != nil {
		return nil, VersionAPIErrorWrap(fmt.Errorf("failed to decode response body: %w", err), *lookup)
	}
	return version, nil
}

func buildVersionListBaseURL(projectID string) (*url.URL, error) {
	baseURL, err := parseURL(GetBaseURL())
	if err != nil {
		return nil, err
	}
	requestURL := urlbuilder.JoinEscapedPath(baseURL, "v2", "project", projectID, "version")
	return requestURL, nil
}

func buildVersionFileURL(lookup *VersionHashLookup) (*url.URL, error) {
	baseURL, err := parseURL(GetBaseURL())
	if err != nil {
		return nil, err
	}
	requestURL := urlbuilder.JoinEscapedPath(baseURL, "v2", "version_file", lookup.hash)
	query := url.Values{}
	query.Set("algorithm", string(lookup.algorithm))
	requestURL.RawQuery = query.Encode()
	return requestURL, nil
}
