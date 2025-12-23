# internal/modrinth

This package speaks to the Modrinth API and returns typed Go models and errors.

It is intentionally low-level: it does not try to decide which file should be installed. If you need "pick the newest compatible file", use `internal/platform` instead.

## Start with the behavior docs

- `docs/platform-apis.md` describes the API behavior we aim to match across implementations.
- Modrinth API docs (external): https://docs.modrinth.com/

## Quick start

Wrap an `httpclient.Doer` with the Modrinth client (it adds headers), then call the API helpers:

```go
client := modrinth.NewClient(httpclient.NewRLClient(limiter))

project, err := modrinth.GetProject("AANobbMI", client)
versions, err := modrinth.GetVersionsForProject(&modrinth.VersionLookup{
	ProjectID:    "AANobbMI",
	Loaders:      []models.Loader{models.FABRIC},
	GameVersions: []string{"1.20.1"},
}, client)
```

## Public API

### Client and base URL

- `NewClient(doer httpclient.Doer) *Client` (adds required headers)
- `GetBaseURL() string`

`GetBaseURL` returns `https://api.modrinth.com`.

### Projects

- `GetProject(projectId string, client httpclient.Doer) (*Project, error)`

### Versions (project lookups)

- `GetVersionsForProject(lookup *VersionLookup, client httpclient.Doer) (Versions, error)`

`VersionLookup` defines the filter for the Modrinth `/version` endpoint (project ID, loaders, and game versions).

### Versions (hash lookups)

- `GetVersionForHash(lookup *VersionHashLookup, client httpclient.Doer) (*Version, error)`

`VersionHashLookup` has unexported fields, so callers outside this package cannot construct it today. The function is currently used only by this package's tests.

## Headers and authentication

The `Client` adds:

- `Accept: application/json`
- `Authorization: <MODRINTH_API_KEY>`
- `User-Agent: github_com/meza/minecraft-mod-manager/<version>`

The version comes from `internal/environment.AppVersion()`, which is replaced at build time for releases.

## Expected errors

Most project-level failures use `internal/globalerrors`:

- `*globalerrors.ProjectNotFoundError` for 404s
- `*globalerrors.ProjectAPIError` for network failures, non-200 status codes, and JSON decode failures

Hash lookups return Modrinth-specific typed errors from `versionErrors.go`:

- `*VersionNotFoundError`
- `*VersionAPIError`

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
