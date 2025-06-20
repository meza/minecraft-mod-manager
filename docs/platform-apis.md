# Platform API Integration

This guide describes how the current TypeScript implementation communicates with the CurseForge and Modrinth services. The Go port should reproduce these behaviours.

## Repository Abstraction

Every platform implements a `Repository` interface with two operations:

- `fetchMod(projectId, allowedReleaseTypes, gameVersion, loader, allowFallback, version?)` – returns `RemoteModDetails` for a single mod file.
- `lookup(fingerprints)` – resolves SHA‑1 file hashes to potential projects.

Both `Curseforge` and `Modrinth` classes live under `src/repositories/` and use a rate‑limited fetch helper to avoid overwhelming remote APIs. Authentication keys come from the environment (`CURSEFORGE_API_KEY` and `MODRINTH_API_KEY`).

## CurseForge

All requests include an `x-api-key` header. When the API responds with `X-Ratelimit-Remaining` below ten requests, the rate limiter delays further calls based on `X-Ratelimit-Reset`.

### Fetching a Mod

1. **Project metadata** – `GET https://api.curseforge.com/v1/mods/{projectId}` returns the mod name.
2. **File list** – `GET https://api.curseforge.com/v1/mods/{projectId}/files?gameVersion={gameVersion}&modLoaderType={loader}` lists available files.
3. Filter files by `sortableGameVersions`, release type and status (`fileStatus` 4 or 10). The latest suitable file is chosen.
4. If a fixed version is provided the file name must match exactly. When no file matches and `allowFallback` is set, retry using the next lower Minecraft version via `fallbackVersion.ts`.
5. The resulting file must contain a download URL and a SHA‑1 hash. Absence of either causes an error (`NoRemoteFileFound` or `CurseforgeDownloadUrlError`).

### Lookup by Fingerprint

`POST https://api.curseforge.com/v1/fingerprints` with `{ fingerprints: [sha1, ...] }` returns all matching files. Each entry becomes a `PlatformLookupResult` with the project ID and file details derived from `hashes` and `downloadUrl`.

## Modrinth

Requests send a custom `User-Agent` (`github_com/meza/minecraft-mod-manager/{version}`) and an `Authorization` header containing the API key.

### Fetching a Mod

1. **Project name** – `GET https://api.modrinth.com/v2/project/{projectId}` retrieves the display name.
2. **Version list** – `GET https://api.modrinth.com/v2/project/{projectId}/version?game_versions=["{gameVersion}"]&loaders=["{loader}"]` fetches available versions.
3. Candidate versions must match the loader, release type and game version. They are sorted by `date_published` with newest first.
4. A fixed version bypasses filtering and simply selects the matching `version_number`.
5. If no suitable version exists and `allowFallback` is true, retry with the next lower Minecraft version.
6. The selected version's first file provides the download URL and SHA‑1 hash which become the `RemoteModDetails`.

### Lookup by Hash

Each hash triggers
`GET https://api.modrinth.com/v2/version_file/{hash}?algorithm=sha1`. Successful responses contain the mod version with file metadata; failures are ignored. All successful results are collected into `PlatformLookupResult` objects.

## Rate-Limited Networking

`src/lib/rateLimiter` queues requests per host and retries failed attempts up to three times. When the server signals rate limiting the job waits according to the reset timer. This ensures both APIs are used politely and shields the CLI from temporary failures.

## Update Check

On startup the CLI queries GitHub Releases (`https://api.github.com/repos/meza/minecraft-mod-manager/releases`) using the rate‑limited fetch helper. If a newer semver tag exists the user is informed about the available update after the command completes.

