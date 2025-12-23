package modrinth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/pkg/errors"
	"net/http"
	"net/url"
	"time"

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

func GetVersionsForProject(ctx context.Context, lookup *VersionLookup, client httpclient.Doer) (versions Versions, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.modrinth.version.list", perf.WithAttributes(attribute.String("project_id", lookup.ProjectID)))
	defer span.End()

	gameVersionsJSON, err := marshalJSON(lookup.GameVersions)
	if err != nil {
		return nil, err
	}
	loadersJSON, err := marshalJSON(lookup.Loaders)
	if err != nil {
		return nil, err
	}

	baseURL, err := parseURL(fmt.Sprintf("%s/v2/project/%s/version", GetBaseURL(), lookup.ProjectID))
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("game_versions", string(gameVersionsJSON))
	query.Set("loaders", string(loadersJSON))
	baseURL.RawQuery = query.Encode()

	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, "GET", baseURL.String(), nil)
	if err != nil {
		return nil, err
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
		return nil, globalerrors.ProjectAPIErrorWrap(errors.Errorf("unexpected status code: %d", response.StatusCode), lookup.ProjectID, models.MODRINTH)
	}

	if err := json.NewDecoder(response.Body).Decode(&versions); err != nil {
		return nil, globalerrors.ProjectAPIErrorWrap(errors.Wrap(err, "failed to decode response body"), lookup.ProjectID, models.MODRINTH)
	}
	return versions, nil
}

func GetVersionForHash(ctx context.Context, lookup *VersionHashLookup, client httpclient.Doer) (version *Version, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.modrinth.version_file.get", perf.WithAttributes(attribute.String("hash", lookup.hash)))
	defer span.End()

	url := fmt.Sprintf("%s/v2/version_file/%s?algorithm=%s", GetBaseURL(), lookup.hash, lookup.algorithm)

	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, "GET", url, nil)
	if err != nil {
		return nil, err
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
		return nil, VersionAPIErrorWrap(errors.Errorf("unexpected status code: %d", response.StatusCode), *lookup)
	}

	version = &Version{}
	if err := json.NewDecoder(response.Body).Decode(version); err != nil {
		return nil, VersionAPIErrorWrap(errors.Wrap(err, "failed to decode response body"), *lookup)
	}
	return version, nil
}
