# internal/curseforge

This package speaks to the CurseForge API and returns typed Go models and errors. It owns CurseForge-specific selection logic and returns the domain `models.RemoteMod` when asked to resolve a file.

## Quick start

Create a client by wrapping an `httpclient.Doer` (usually rate-limited), then call the API helpers:

```go
client := curseforge.NewClient(httpclient.NewRLClient(limiter))
project, err := curseforge.GetProject("1234", client)
files, err := curseforge.GetFilesForProject(1234, client)
```

To select a downloadable file, use the higher-level helper:

```go
remote, err := curseforge.FetchRemoteMod(ctx, "1234", models.FetchOptions{
	AllowedReleaseTypes: []models.ReleaseType{models.Release},
	GameVersion:         "1.20.1",
	Loader:              models.FABRIC,
}, client)
```

## Public API

### Client and base URL

- `NewClient(doer httpclient.Doer) *Client` (adds required headers)
- `GetBaseURL() string`

`GetBaseURL` returns `https://api.curseforge.com/v1`.

### Projects

- `GetProject(projectId string, client httpclient.Doer) (*Project, error)`

### Files

- `GetFilesForProject(projectId int, client httpclient.Doer) ([]File, error)` (handles pagination)
- `GetFilesForProjectWithFilters(projectId int, filter FileListFilter, client httpclient.Doer) ([]File, error)`
- `FetchRemoteMod(ctx context.Context, projectId string, opts models.FetchOptions, client httpclient.Doer) (models.RemoteMod, error)`

### Fingerprints (hash lookups)

- `GetFingerprintsMatches(fingerprints []uint32, client httpclient.Doer) (*FingerprintResult, error)`

The API expects CurseForge fingerprints (uint32). This is separate from Modrinth SHA-1 lookups.

## Headers and authentication

The `Client` adds:

- `Accept: application/json`
- `x-api-key: <CURSEFORGE_API_KEY>`

The API key is read via `internal/environment.CurseforgeAPIKey()`.

## Expected errors

Most project-level failures use `internal/globalerrors`:

- `*globalerrors.ProjectNotFoundError` for 404s
- `*globalerrors.ProjectAPIError` for request/URL build failures, network failures, non-200 status codes, and JSON decode failures

Fingerprint lookups return `*FingerprintAPIError` (it includes the lookup input so callers can correlate failures, including request/URL build failures).

## Related docs

`docs/platform-apis.md` captures the behavior we aim to match across implementations.

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
