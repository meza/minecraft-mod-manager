# internal/modrinth

This package speaks to the Modrinth API and returns typed Go models and errors.

It is intentionally low-level: it does not try to decide which file should be installed. If you need "pick the newest compatible file", use `internal/platform` instead.

## Start with the behavior docs

- `docs/platform-apis.md` describes the API behavior we aim to match across implementations.
- Modrinth API docs (external): https://docs.modrinth.com/

## Quick start

Wrap an `httpClient.Doer` with the Modrinth client (it adds headers), then call the API helpers:

```go
client := modrinth.NewClient(httpClient.NewRLClient(limiter))

project, err := modrinth.GetProject("AANobbMI", client)
versions, err := modrinth.GetVersionsForProject(&modrinth.VersionLookup{
	ProjectId:    "AANobbMI",
	Loaders:      []models.Loader{models.FABRIC},
	GameVersions: []string{"1.20.1"},
}, client)
```

## Public API

### Client and base URL

- `NewClient(doer httpClient.Doer) *Client` (adds required headers)
- `GetBaseUrl() string`

`GetBaseUrl` returns `https://api.modrinth.com`.

### Projects

- `GetProject(projectId string, client httpClient.Doer) (*Project, error)`

### Versions (project lookups)

- `GetVersionsForProject(lookup *VersionLookup, client httpClient.Doer) (Versions, error)`

`VersionLookup` defines the filter for the Modrinth `/version` endpoint (project ID, loaders, and game versions).

### Versions (hash lookups)

- `GetVersionForHash(lookup *VersionHashLookup, client httpClient.Doer) (*Version, error)`

`VersionHashLookup` has unexported fields, so callers outside this package cannot construct it today. The function is currently used only by this package's tests.

## Headers and authentication

The `Client` adds:

- `Accept: application/json`
- `Authorization: <MODRINTH_API_KEY>`
- `User-Agent: github_com/meza/minecraft-mod-manager/<version>`

The version comes from `internal/environment.AppVersion()`, which is replaced at build time for releases.

## Expected errors

Most project-level failures use `internal/globalErrors`:

- `*globalErrors.ProjectNotFoundError` for 404s
- `*globalErrors.ProjectApiError` for network failures, non-200 status codes, and JSON decode failures

Hash lookups return Modrinth-specific typed errors from `versionErrors.go`:

- `*VersionNotFoundError`
- `*VersionApiError`

## Tests

Run from the repo root:

```bash
make test
```
