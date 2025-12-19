# internal/platform

This module is the small "adapter" layer that lets the rest of the CLI fetch mod files from different hosting platforms through one API.

If you are working on `add`, `install`, `update`, or `scan`, this is the package that answers: "Given a platform + project ID, which file should we download for this loader/game version?"

## Quick start

`FetchMod` is the entry point. It returns a `RemoteMod` with a name, filename, SHA-1 hash, release date, and download URL.

```go
limiter := rate.NewLimiter(rate.Every(250*time.Millisecond), 1)
clients := platform.DefaultClients(limiter)

remote, err := platform.FetchMod(models.MODRINTH, "P7dR8mSH", platform.FetchOptions{
	AllowedReleaseTypes: []models.ReleaseType{models.Release},
	GameVersion:         "1.20.1",
	Loader:              models.FABRIC,
	AllowFallback:       true,
}, clients)
```

Imports you will typically need for this:

```go
import (
	"time"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"golang.org/x/time/rate"
)
```

In commands that need to show a helpful message, you typically branch on the typed errors:

```go
switch err.(type) {
case *platform.UnknownPlatformError, *platform.ModNotFoundError, *platform.NoCompatibleFileError:
	// Show a friendly message to the user.
default:
	// Treat as an unexpected failure.
}
```

## Public API (what other packages should use)

`internal/platform` is intentionally small. Most callers only need these:

- `FetchMod(platform models.Platform, projectID string, opts FetchOptions, clients Clients) (RemoteMod, error)`
- `DefaultClients(limiter *rate.Limiter) Clients`
- `FetchOptions` (selection inputs)
- `RemoteMod` (selection output)
- `UnknownPlatformError`, `ModNotFoundError`, `NoCompatibleFileError` (expected failure modes)

## Perf instrumentation

The public entrypoint `FetchMod(...)` is wrapped with a perf region:

- `platform.fetch_mod`

This sits above provider-specific `api.*` and `net.http.*` regions so you can tell whether time is spent in platform orchestration (selection, fallback iteration) vs the underlying HTTP calls.

### FetchOptions

`FetchOptions` describes how to pick a file:

- `AllowedReleaseTypes`: which release types are acceptable (`release`, `beta`, `alpha`)
- `GameVersion`: the target Minecraft version to resolve against (for example `1.20.1`)
- `Loader`: the mod loader (for example `fabric`, `forge`)
- `FixedVersion`: when set, pins the selection to a single version identifier:
  - Modrinth: matches `version_number`
  - CurseForge: matches the file name (case-insensitive)
- `AllowFallback`: when true, retry with a lower patch version if nothing matches

### RemoteMod

`RemoteMod` is the normalized "download this" shape used by the command layer:

- `Name`: display name for UX (project title/name)
- `FileName`: jar filename to write to the mods directory
- `ReleaseDate`: RFC3339 timestamp string
- `Hash`: SHA-1 hash string
- `DownloadURL`: direct URL to download the jar

## How selection works

The intent is "pick the newest compatible file."

For both platforms:

1. Fetch project metadata to get a human-friendly name.
2. Fetch the available files/versions for the requested game version and loader.
3. Filter candidates by `AllowedReleaseTypes` and `FixedVersion` (if provided).
4. Sort by publish date descending and take the newest.
5. Require a download URL and SHA-1 hash.

### Fallback behavior

Fallback is deliberately conservative: it only decreases the patch component.

- `1.20.2` falls back to `1.20.1`
- `1.20.1` does not fall back (there is no `1.20.0` attempt)
- `1.20` does not fall back (no patch component to decrement)

This is implemented by `nextVersionDown` in `internal/platform/fallback.go`.

## Clients, headers, and environment

`FetchMod` does not build its own HTTP clients. Callers provide `platform.Clients` so commands can:

- reuse a shared rate limiter
- stub the network in tests
- keep platform auth/header logic in the platform-specific packages

`DefaultClients` returns two rate-limited HTTP `Doer`s. Platform-specific packages wrap those `Doer`s to add headers:

- Modrinth uses `MODRINTH_API_KEY` and a `User-Agent` derived from `internal/environment.AppVersion()`
- CurseForge uses `CURSEFORGE_API_KEY` via the `x-api-key` header

API base URLs are not configurable at runtime. For tests, provide `Clients` with `Doer` implementations that route requests to `httptest` servers.

## Expected errors (and what they mean)

These error types are part of the contract with the command layer:

- `UnknownPlatformError`: the platform enum is not recognized by this module
- `ModNotFoundError`: the project ID does not exist on that platform (mapped from a 404)
- `NoCompatibleFileError`: the project exists, but nothing matches the selection rules (or the matching file is missing a URL/SHA-1)

The rest of the errors are treated as unexpected failures (network issues, API errors, bad JSON, etc).

## Adding a new platform

When you add support for another platform, keep the surface area the same:

1. Add a new `models.Platform` value (in `internal/models`) and ensure it has a stable string form for UX.
2. Implement a `fetch<Platform>` function that returns `RemoteMod` and uses the same filtering intent as the existing platforms.
3. Add a new `case` in `FetchMod`'s switch.
4. Map platform-specific "not found" responses to `ModNotFoundError` so the UX stays consistent.
5. Add `httptest` coverage in `internal/platform/platform_test.go` for success, not-found, no-compatible-file, and fallback.

## Tests

This package is tested with local `httptest` servers via injected `Doer`s. Run the repo test suite from the root:

```bash
make test
```
