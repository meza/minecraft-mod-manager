package modrinth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
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
	Sha1   VersionAlgorithm = "sha1"
	Sha512 VersionAlgorithm = "sha512"
)

type VersionFileHash struct {
	Sha1   string `json:"sha1"`
	Sha512 string `json:"sha512"`
}

type VersionFile struct {
	FileName string          `json:"filename"`
	Hashes   VersionFileHash `json:"hashes"`
	Primary  bool            `json:"primary"`
	Size     int64           `json:"size"`
	Url      string          `json:"url"`
}

type VersionDependency struct {
	FileName  string         `json:"file_name"`
	ProjectId string         `json:"project_id"`
	Type      DependencyType `json:"type"`
	VersionId string         `json:"version_id"`
}

type Version struct {
	Changelog     string              `json:"changelog"`
	DatePublished time.Time           `json:"date_published"`
	Dependencies  []VersionDependency `json:"dependencies"`
	Files         []VersionFile       `json:"files"`
	GameVersions  []string            `json:"game_versions"`
	Loaders       []models.Loader     `json:"loaders"`
	Name          string              `json:"name"`
	ProjectId     string              `json:"project_id"`
	Status        VersionStatus       `json:"status"`
	Type          models.ReleaseType  `json:"version_type"`
	VersionId     string              `json:"id"`
	VersionNumber string              `json:"version_number"`
}

type Versions []Version

type VersionLookup struct {
	ProjectId    string          `json:"project_id"`
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

func GetVersionsForProject(ctx context.Context, lookup *VersionLookup, client httpClient.Doer) (Versions, error) {
	ctx, span := perf.StartSpan(ctx, "api.modrinth.version.list", perf.WithAttributes(attribute.String("project_id", lookup.ProjectId)))
	defer span.End()

	gameVersionsJSON, _ := json.Marshal(lookup.GameVersions)
	loadersJSON, _ := json.Marshal(lookup.Loaders)

	baseURL, _ := url.Parse(fmt.Sprintf("%s/v2/project/%s/version", GetBaseUrl(), lookup.ProjectId))
	query := url.Values{}
	query.Set("game_versions", string(gameVersionsJSON))
	query.Set("loaders", string(loadersJSON))
	baseURL.RawQuery = query.Encode()

	request, _ := http.NewRequestWithContext(ctx, "GET", baseURL.String(), nil)
	response, err := client.Do(request)
	if err != nil {
		return nil, globalErrors.ProjectApiErrorWrap(err, lookup.ProjectId, models.MODRINTH)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalErrors.ProjectNotFoundError{
			ProjectID: lookup.ProjectId,
			Platform:  models.MODRINTH,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, globalErrors.ProjectApiErrorWrap(errors.Errorf("unexpected status code: %d", response.StatusCode), lookup.ProjectId, models.MODRINTH)
	}

	result := &Versions{}
	_ = json.NewDecoder(response.Body).Decode(result)
	return *result, nil
}

func GetVersionForHash(ctx context.Context, lookup *VersionHashLookup, client httpClient.Doer) (*Version, error) {
	ctx, span := perf.StartSpan(ctx, "api.modrinth.version_file.get", perf.WithAttributes(attribute.String("hash", lookup.hash)))
	defer span.End()

	url := fmt.Sprintf("%s/v2/version_file/%s?algorithm=%s", GetBaseUrl(), lookup.hash, lookup.algorithm)

	request, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	response, err := client.Do(request)
	if err != nil {
		return nil, VersionApiErrorWrap(err, *lookup)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, &VersionNotFoundError{
			Lookup: *lookup,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, VersionApiErrorWrap(errors.Errorf("unexpected status code: %d", response.StatusCode), *lookup)
	}

	result := &Version{}
	_ = json.NewDecoder(response.Body).Decode(result)
	return result, nil
}
