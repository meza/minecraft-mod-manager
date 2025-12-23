# internal/curseforge

This package speaks to the CurseForge API and returns typed Go models and errors. It is intentionally low-level: it does not try to decide which file to download.

If you need "pick the newest compatible file", use `internal/platform` instead.

## Quick start

Create a client by wrapping an `httpclient.Doer` (usually rate-limited), then call the API helpers:

```go
client := curseforge.NewClient(httpclient.NewRLClient(limiter))
project, err := curseforge.GetProject("1234", client)
files, err := curseforge.GetFilesForProject(1234, client)
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

### Fingerprints (hash lookups)

- `GetFingerprintsMatches(fingerprints []int, client httpclient.Doer) (*FingerprintResult, error)`

The API expects CurseForge fingerprints (integers). This is separate from Modrinth SHA-1 lookups.

## Headers and authentication

The `Client` adds:

- `Accept: application/json`
- `x-api-key: <CURSEFORGE_API_KEY>`

The API key is read via `internal/environment.CurseforgeAPIKey()`.

## Expected errors

Most project-level failures use `internal/globalerrors`:

- `*globalerrors.ProjectNotFoundError` for 404s
- `*globalerrors.ProjectAPIError` for network failures, non-200 status codes, and JSON decode failures

Fingerprint lookups return `*FingerprintAPIError` (it includes the lookup input so callers can correlate failures).

## Related docs

`docs/platform-apis.md` captures the behavior we aim to match across implementations.

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
