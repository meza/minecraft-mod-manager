# Minecraft Mod Manager (Go) - Audit Report

Date: 2025-12-19

## Scope

Repository-wide audit of:

- Go codebase architecture, correctness, style, hygiene
- Tests and coverage gates
- CI/CD and release automation
- Dependencies and vulnerability posture
- Documentation accuracy vs current behavior
- Security posture (secrets, networking, file I/O, telemetry/privacy)

## Clarifications From Maintainer (applied)

- Help/version replacement strings are injected during semantic-release; local dev builds may keep placeholders (not an issue).
- Local builds requiring secrets (tokens) is acceptable because the app requires those tokens to function.
- API keys are intended to be embedded in distributed binaries.
- Runtime `.env` support is required so users can provide their own Modrinth/CurseForge keys (for example, private projects).
- Telemetry is intended to be opt-out.
- Go port is not complete; some commands documented in legacy docs may not exist yet.
- `CURSEFORGE_API_URL` / `MODRINTH_API_URL` must not be configurable via env vars in production; if tests need overrides, they must be injected via build/test harness (not runtime env).
- Runtime `.env` should be loaded from the current working directory (global config location will be added later).
- Current behavior for `mmm` with no args is "print help" (no-args TUI is not implemented yet).
- Missing no-args TUI is a known gap while pushing for Node parity; once parity is complete, the plan is to add a fully interactive no-args TUI flow.
- Retry should be enabled even for APIs that use POST incorrectly; treat requests as safe to retry, but implement retries safely for requests with bodies.
- Telemetry may include request URLs, but must not include API keys from headers (or other secret header values).
- Perf artifacts (`mmm-perf.json`) are not considered sensitive by the team.

These clarifications reduce some "mismatch" items from blockers to documentation or governance issues, but they do not eliminate underlying security and correctness risks described below.

## Verification Artifacts (executed locally)

- `make coverage` (pass)
- `make build` (pass)
- `go vet ./...` (pass)
- `go mod verify` (pass)
- `make test-race` (fails; data races detected in tests and test-only harness code)
- `staticcheck ./...` (findings; see Tooling)
- `govulncheck -scan=module` and `govulncheck -scan=package ./...` (findings; see Security)
- `govulncheck -scan=symbol ./...` fails with an internal type-checking error (toolchain/library incompatibility; see Security)
- `go-licenses report ./...` (license inventory; see Licensing finding)

Notes on reproducing these runs:

- `staticcheck` and `govulncheck` are not shipped as part of this repo; I installed them via `go install` into `/tmp/bin` using `GOTOOLCHAIN=go1.25.0` so they can analyze a `go 1.25` module.

## Executive Verdict

Not release-ready. There are multiple blocker/critical risks across release automation consistency, documentation accuracy, credential exfiltration risk, network robustness (unbounded hangs), and download integrity.

## Findings (prioritized)

### Blocker - Data races detected under `go test -race` (mmm-63.1)

Evidence:

- `go test -race ./...` reports a race in `cmd/mmm/test` tests due to concurrent `fetchMod` calls updating shared test state without synchronization: `cmd/mmm/test/test_test.go:126` (race observed while executing `cmd/mmm/test/test.go:344` from goroutines created in `cmd/mmm/test/test.go:249`).
- `go test -race ./...` reports a race in lifecycle test harness where `reset()` sets `signalChan=nil` while the listener goroutine reads from it: `internal/lifecycle/lifecycle.go:78` and `internal/lifecycle/lifecycle.go:126`.

Impact:

- Race-enabled CI would fail today.
- Races inside tests indicate concurrency behavior is not being validated safely; this undermines confidence in concurrent command behavior even if production code is largely single-session.

Remediation:

- Make the test suite race-clean and add `go test -race ./...` to CI (either as a separate job or behind a periodic/nightly gate if runtime is too expensive).
- Treat race failures as blockers; do not accept any racy tests.

Confidence: High.

---

### Blocker - Release automation is internally inconsistent / likely non-functional (mmm-63.2)

Evidence:

- CI workflow is named "Verify and Release" but does not actually release (release step is commented out): `.github/workflows/ci.yml:1` and `.github/workflows/ci.yml:40`.
- `.releaserc.json` still references a Node-era file path `./src/version.ts` that does not exist in this Go repo: `.releaserc.json:42`.
- The release prepare script will fail if `./src/` does not exist (it does a direct `cat > "$FILE"`), and `.releaserc.json` passes `./src/version.ts`: `scripts/prepare.sh:6-8` and `.releaserc.json:42`.
- `.releaserc.json` expects `CHANGELOG.md` output but the file is absent and the repo does not show any changelog generation in CI: `.releaserc.json:33`.
- `scripts/prepare.sh` mutates a tracked Go source file (`internal/environment/environment.go`) in-place via `sed`: `scripts/prepare.sh:10`.
- Release artifact selection uses broad globs (`dist/*-windows.zip`, etc) which can match unrelated or stale zip files in `dist/`, risking accidental upload of extra artifacts: `.releaserc.json:49-60`.
- There is no checksum/signature generation or verification in the release scripts or CI workflow (no `.sha256` artifacts, no signing, no provenance attestation): `.github/workflows/ci.yml:1-45`, `scripts/binaries.sh:1-7`.

Impact:

- Releases cannot be reproduced from CI as written.
- Version/help URL injection and artifact generation risk producing inconsistent outputs depending on workflow path.

Remediation (decision required):

- Decide a single authoritative release path:
  - Either re-enable semantic-release in CI and make `.releaserc.json` match the Go repo layout, or
  - Remove/replace Node-era semantic-release config if it is not intended for the Go port.
- Avoid "rewrite tracked Go files in CI" as a release mechanism; prefer `-ldflags -X` injection (same pattern as the embedded tokens in the Makefile).

Confidence: High.

---

### Blocker - Documentation is materially out of sync with current Go behavior (mmm-63.3)

Evidence:

- `README.md` documents Node.js + pnpm contributor workflow, not the actual `make`-based Go workflow: `README.md:522`.
- `docs/tui-design-doc.md` describes "no args starts full TUI", but current behavior prints help (no root `Run` behavior in `cmd/mmm/root.go:23`). Maintainer confirms help is correct for the current version and the no-args TUI is a known gap while pushing for Node parity.
- Command set mismatch: `mmm --help` does not list `change` or `prune`, but README/specs imply them. (Go port not complete per maintainer; therefore docs must reflect current state.)
- Maintainer/package docs are also stale and conflict with code and build tooling:
  - `internal/config/README.md` documents `ReadConfig(fs, meta)` and `EnsureLock(fs, meta)`, but the actual public API is context-aware: `internal/config/config.go:16` and `internal/config/lock.go:14`.
  - `internal/httpClient/README.md` documents `DownloadFile(url, filepath, ...)`, but the actual API includes context: `internal/httpClient/downloader.go:40`.
  - `internal/environment/README.md` claims release/CI injection uses `-ldflags -X` and "not rewriting source files": `internal/environment/README.md:28`, but `scripts/prepare.sh` rewrites `internal/environment/environment.go` in-place via `sed`: `scripts/prepare.sh:10-12` (and `.releaserc.json` still references `./src/version.ts`: `.releaserc.json:42`).

Impact:

- Users and contributors cannot rely on docs to understand current behavior.
- Specs cannot be used as acceptance criteria if they describe unimplemented commands as current.

Remediation:

- Bring user-facing docs (especially `README.md` and `docs/commands/*`) into alignment with the Go binary that currently ships.
- If some commands are not implemented yet, remove them from user docs or clearly mark them as unavailable in the current Go build without framing as "future plans" (per your documentation principles).
- `docs/tui-design-doc.md` must stop claiming current behavior that is not implemented; treat it as an aspirational design doc and ensure user docs match today's behavior (help by default).

Additional evidence - command-by-command docs/specs vs current CLI (help + code):

- Global flags and command set:
  - Specs list only `--config`, `--quiet`, `--debug` and claim they must appear before the command name: `docs/specs/README.md:16-24`.
  - The Go CLI also exposes `--perf` and `--perf-out-dir` global flags: `cmd/mmm/root.go:30-34`.
  - Cobra also exposes a global `-v, --version` flag because `Version` is set: `cmd/mmm/root.go:24-28` (this directly conflicts with docs that claim `add` uses `-v` for its own `--version` flag).
  - Verification: persistent flags are accepted after the command name (contradicting `docs/specs/README.md:24`), for example `mmm install --config ./modlist.json --help` works (exit code 0) and prints help.
  - `mmm --help` includes a `completion` command (cobra default). This is not mentioned in `docs/specs/README.md:5-14` nor in `docs/commands/*`.

- `add`:
  - User docs claim short flags that do not exist:
    - `docs/commands/add.md:23-24` claims `-f, --allow-version-fallback` and `-v, --version`.
    - Actual flags are long-only (no shorthands): `cmd/mmm/add/add.go:148-149`.
    - Additionally, `-v` is already taken by the global `--version` flag (see above), so the docs are not just stale; they are internally inconsistent with the current CLI surface.
  - User docs say Modrinth requires a project slug, but the examples use what looks like a Modrinth project ID (base62-ish), not a slug:
    - `docs/commands/add.md:8-9` uses `AANobbMI`
    - `docs/commands/add.md:68-74` claims "project slug (last part of the URL)" for Modrinth
    - The CLI likely accepts both slug and ID (Modrinth API does), so the docs should state "slug or project ID" to match reality.

- `init`:
  - User docs promise an interactive TUI when running in a terminal and a flag-driven flow in scripts: `docs/commands/init.md:11-13`. This broadly matches the code structure (TUI selection is terminal-dependent), but multiple help strings are incorrect and leak host-specific paths (see the dedicated finding on `init --mods-folder` help text).
  - User docs include a process-oriented "open an issue on GitHub" note, which conflicts with the "no future plans / no internal process" constraint for user-facing docs: `docs/commands/init.md:30`.

- `install`:
  - User docs omit a major behavioral constraint: install can fail due to unmanaged/unresolved files and can suggest running `scan` first.
    - Spec includes this preflight and failure mode: `docs/specs/install.md:6-16`.
    - User docs (`docs/commands/install.md:1-37`) do not mention unmanaged file detection or the "unresolved" failure mode at all.
  - User docs provide no flags/options table (even if the only options are global flags), which makes the page inconsistent with the other command docs and harder to use as a reference: `docs/commands/install.md:1`.

- `list`:
  - Specs claim the command is purely informational and has no prompts: `docs/specs/list.md:13-14`.
  - Implementation can run in a TUI mode (interactive) depending on TTY and `--quiet`: `cmd/mmm/list/list.go:112-133`.
  - Implementation prints non-ASCII check/cross glyphs in list output even when not using the TUI (this is a CLI output contract decision that should be documented): `cmd/mmm/list/list.go:224-233`.
  - User docs are missing core usage/flags documentation and do not mention the TUI mode:
    - `docs/commands/list.md:1-13` has no flags table and no mention of the interactive list UI.

- `remove`:
  - User docs claim `--dry-run` does not create a missing lock file: `docs/commands/remove.md:24-25`. This is true for dry-run specifically.
  - However, non-dry-run will create `modlist-lock.json` if it is missing (via `EnsureLock`) even if there are no matching lock entries to remove: `cmd/mmm/remove/remove.go:178-191`.
  - Implementation prints a hard-coded emoji check mark and bypasses i18n: `cmd/mmm/remove/remove.go:135-172`.

- `scan`:
  - Specs and user docs both claim persistence is skipped when any file is "unsure": `docs/commands/scan.md:20-21` and `docs/specs/scan.md:14-18`. This matches implementation: `cmd/mmm/scan/scan.go:264-276`.
  - The docs still inherit the global `--quiet` description mismatch ("no normal output" vs "suppress all output") because the root flag help text is inaccurate: `cmd/mmm/root.go:31`.

- `test`:
  - User docs describe exit codes for automation: `docs/commands/test.md:26-35`. This aligns with the implementation intent (exit code 2 for same-version): `cmd/mmm/test/test.go:67-71`.
  - CLI help output does not mention the exit code contract anywhere; this is a UX/doc mismatch for an "automation-first" command.
  - Internal spec claims interactive prompting when the latest version lookup fails: `docs/specs/test.md:14`. Current implementation is non-interactive and instead requires an explicit version (see the maintainer clarification and `cmd/mmm/test/test.go:198-207`).

- `update`:
  - User docs and specs both describe "runs install first" behavior: `docs/commands/update.md:13-17` and `docs/specs/update.md:6-16`. This matches implementation: `cmd/mmm/update/update.go:157-165`.

Confidence: High.

---

### Medium - ADR corpus is Node-era and references are inconsistent (governance drift) (mmm-63.4)

Evidence:

- Repo guidance claims ADRs live under `docs/docs/`, but this repository stores them under `doc/adr/`:
  - `AGENTS.md:64` and `AGENTS.md:125`
  - actual ADR path: `doc/adr/0001-record-architecture-decisions.md:1` (and peers)
- Multiple ADRs are clearly written for the legacy Node implementation (pnpm, Renovate, `console.log` rules):
  - `doc/adr/0004-using-pnpm.md:1-24`
  - `doc/adr/0005-using-renovate-bot.md:1-22`
  - `doc/adr/0002-console-log-only-in-actions.md:20-27`
- ADR 0006 explicitly assumes a rate-limited client with retries, but the Go port often disables rate limiting and does not retry request errors:
  - ADR intent: `doc/adr/0006-verifying-minecraft-versions-error-handling.md:21-22`
  - Go behavior: `cmd/mmm/test/test.go:106` (uses `rate.NewLimiter(rate.Inf, 0)`), `internal/httpClient/httpClient.go:80-92` (returns immediately on request error)

Impact:

- Reviewers and contributors do not have a reliable single source of truth for "why" decisions in the Go port.
- ADR 0006 intent is partially violated in the Go implementation (not just documentation drift).

Remediation:

- Fix the ADR location references in contributor guidance to match the repo layout.
- Decide which ADRs still apply to the Go port vs are historical Node-only decisions, and clarify scope in `doc/adr/decisions.md`.
- Write Go-port ADRs for current, security-relevant decisions (telemetry scope, retries, download integrity, base URL override policy) so these constraints are not tribal knowledge.

Confidence: High.

---

### Medium - Markdown "no non-ASCII formatting" rule is violated broadly (mmm-63.5)

Maintainer clarification:

- Smart quotes, em-dashes, and any non-ASCII formatting characters are banned.
- Non-ASCII is acceptable in TUI examples and emojis.

Evidence:

- Scanning tracked Markdown files found widespread non-ASCII characters outside fenced code blocks and excluding emoji. Summary:
  - tracked markdown files scanned: 69
  - violations outside code fences (non-emoji): 277
- Examples of banned formatting characters in prose (not code fences):
  - em-dash in `AGENTS.md:29`
  - en-dash and smart quotes in `docs/packaging-research.md:11`
  - non-breaking hyphen in `docs/tui-design-doc.md:7` and `docs/tui-design-doc.md:11`

Impact:

- This violates the stated documentation style rule and will keep regressing unless enforced automatically.
- Non-ASCII punctuation causes copy/paste issues in terminals and makes docs inconsistent with project conventions.

Remediation:

- Implement an automated gate that rejects non-ASCII characters outside fenced code blocks, with a narrow allowlist for emoji (if you truly want emoji allowed outside code fences).
- Normalize existing docs by replacing:
  - smart quotes -> `'` / `"`
  - em/en dashes -> `--` / `-`
  - non-breaking hyphens/spaces -> regular `-` / space

Confidence: High (tool-confirmed scan results).

---

### Critical - Credential exfiltration risk via runtime `.env` autoload + API base URL overrides (mmm-63.6)

Context (why this matters):

You asked for more context on my earlier question about API base URL overrides. The issue is not "having overrides" by itself; it is the combination of:

1) The binary auto-loads `.env` at runtime from the current working directory, and
2) The HTTP clients allow overriding the API base URL via env vars, and
3) Those clients always attach auth headers (your embedded keys) to requests.

Evidence:

- Runtime `.env` autoload: `main.go:12` imports `github.com/joho/godotenv/autoload`.
- Modrinth base URL override: `internal/modrinth/modrinthClient.go:39` reads `MODRINTH_API_URL`.
- CurseForge base URL override: `internal/curseforge/curseforgeClient.go:37` reads `CURSEFORGE_API_URL`.
- Auth headers are unconditionally added:
  - Modrinth: `internal/modrinth/modrinthClient.go:26`
  - CurseForge: `internal/curseforge/curseforgeClient.go:25`

Exploit scenario:

- If a user runs `mmm` in a folder containing a malicious `.env` (for example, shipped inside a modpack zip or a repo checkout), that `.env` can set `MODRINTH_API_URL` / `CURSEFORGE_API_URL` to an attacker-controlled host.
- The binary will then send the embedded API keys to that host via `Authorization` / `x-api-key`.

Impact:

- Leaks embedded API keys.
- Keys are intended to be embedded, but that does not make exfiltration acceptable; it broadens the exposure surface and makes accidental leakage easier and harder to detect.

Remediation:

- Remove production support for env-driven API base URL overrides entirely (per maintainer decision).
- Replace test usage of `MODRINTH_API_URL` / `CURSEFORGE_API_URL` with an explicit test hook:
  - either a `SetBaseURLForTesting(...) func() restore` pattern in each package, or
  - a build-time injected default (for example `-ldflags -X ...`) used only in test binaries.
- Also reject empty base URLs defensively; today `internal/curseforge/curseforgeClient.go:39-41` returns an empty string when the env var is set to empty.
- Maintain runtime `.env` support for user-supplied keys (per maintainer decision) loaded from CWD, but do not allow `.env` to affect network destinations; use an allowlist-based loader (only import known MMM env vars) rather than `godotenv/autoload` loading everything from the current directory.

Confidence: High.

---

### Critical - `make build` executes `.env` as shell code on Unix hosts (build-time code execution risk) (mmm-63.7)

Evidence:

- Unix Makefile logic sources `.env` directly into the shell using the `.` builtin: `Makefile:114` (`set -a; . ./.env; set +a`).

Impact:

- A `.env` file is treated as executable shell script, not as data. Any command substitutions or shell statements in `.env` will execute when a developer runs `make build` (or any build target using `GO_BUILD_WITH_EMBEDDED_TOKENS`).
- This is a straightforward local code execution vector if a user runs `make build` in a directory where `.env` was supplied by an untrusted source (malicious repo checkout, extracted modpack, etc).

Remediation:

- Treat `.env` as data, not code: parse `KEY=VALUE` lines instead of sourcing.
- Apply the same parsing behavior across Unix and Windows build paths (Windows already parses line-by-line rather than executing).
- Document that `.env` must never be obtained from untrusted sources if it remains executable.

Confidence: High (direct evidence in Makefile).

---

### Critical - Downloader can overwrite arbitrary files via symlinks; partial writes are not cleaned up (mmm-63.8)

Evidence:

- Downloader writes directly to the destination path via `fs.Create(path)` with no attempt to detect symlinks or write to a temp file: `internal/httpClient/downloader.go:58`.
- On download errors, it returns without removing partially-written files: `internal/httpClient/downloader.go:76-81`.
- Update flow deletes the previous jar after a successful download without validating the replacement content: `cmd/mmm/update/update.go:419-428`.

Impact:

- If an attacker can place a symlink in the mods directory (or influence filenames via path traversal; see separate finding), MMM can be tricked into overwriting arbitrary files that the current user can write to.
- Partial/truncated files can be left behind as `.jar`, leading to "corrupt jar" state and (in update flows) deletion of the previously working jar.

Remediation:

- Always download to a temp file in the same directory and `Rename` into place only after validation.
- Validate HTTP status codes (see "Download integrity") and verify hashes before swap/delete.
- On OS filesystems, detect and reject symlinks for jar destination paths (or open with platform-specific "no-follow" semantics where possible).

Confidence: Medium-High (symlink overwrite depends on attacker control of the mods directory).

---

### Critical - Unsafe filename handling enables path traversal / arbitrary file overwrite or delete (mmm-63.9)

Evidence:

- The tool uses remote or lock-provided filenames directly in filesystem paths via `filepath.Join(...)`:
  - `internal/modsetup/modsetup.go:74` joins `remote.FileName`
  - `internal/modinstall/modinstall.go:58` joins `install.FileName` (lock file content)
  - `cmd/mmm/update/update.go:381` joins `remote.FileName` and later deletes `oldPath` on success: `cmd/mmm/update/update.go:423`
  - `cmd/mmm/remove/remove.go:151` joins lock-provided `FileName` and then deletes it: `cmd/mmm/remove/remove.go:152-154`
  - `cmd/mmm/add/add.go:291` joins `remoteMod.FileName`

Impact:

- If a remote API returns a filename containing path separators (or if the lock file is tampered with), the tool can write outside the mods directory (for example `../somewhere`), and update/remove flows can delete unintended files.
- Relying on "upstream APIs always return safe filenames" is not an acceptable security boundary.

Remediation:

- Enforce that all mod filenames are safe base names before any file I/O:
  - reject values containing path separators, drive letters, or traversal sequences
  - require `filepath.Base(name) == name` (and `path.Base` equivalently for forward slashes)
  - optionally enforce allowed extensions (for example `.jar`) and a maximum length
- Apply the validation at every boundary:
  - remote platform selection output
  - lock file reads
  - config-driven file operations

Confidence: High.

---

### High - `modsFolder` is treated as trusted and can point anywhere on disk (untrusted config risk) (mmm-63.10)

Evidence:

- `Metadata.ModsFolderPath` accepts absolute/rooted paths and returns them unchanged: `internal/config/metadata.go:28-33` and `internal/config/metadata.go:35-40`.
- The command layer uses `meta.ModsFolderPath(cfg)` as the base directory for downloads/writes/removals (examples):
  - `cmd/mmm/install/install.go:302-305`
  - `cmd/mmm/update/update.go:381-429`
  - `cmd/mmm/add/add.go:291-296`

Impact:

- A malicious/untrusted `modlist.json` can direct MMM to write, overwrite, or delete files outside the expected project directory by setting `modsFolder` to an absolute path or a traversal path (for example `/`, `C:\\`, or `../..`), under the current user's privileges.
- This is not an RCE by itself, but it is a sharp edge that can cause real data loss when users run MMM on untrusted modpacks/repos.

Remediation:

- Decide whether `modsFolder` is allowed to be absolute/outside the config directory.
- If not, validate that `modsFolder` resolves under `meta.Dir()` and reject/require explicit confirmation when it does not.
- If it must remain flexible, document the trust boundary: treat `modlist.json` as code-like input (do not run it from untrusted sources) and add guardrails (warnings, `--allow-external-mods-folder` flag, etc.).

Confidence: Medium-High (depends on whether you consider `modlist.json` a trusted local config vs an importable artifact).

---

### High - Atomic writes are not symlink-safe (TOCTOU overwrite risk in untrusted directories) (mmm-63.11)

Evidence:

- Atomic config writes use a predictable sibling temp path and delete it before writing:
  - `internal/config/atomic_write.go:12-24` allocates `targetPath + ".mmm.tmp"` (or `...tmp.N`) and calls `_ = fs.Remove(tempPath)` then `afero.WriteFile(fs, tempPath, ...)`.
- Update in-place swaps use predictable temp paths in the mods directory:
  - `cmd/mmm/update/update.go:405-416` uses `newPath + ".mmm.tmp"` and removes it before downloading.

Impact:

- In attacker-controlled directories (malicious repo checkout, extracted modpack, shared folder), an attacker can race-create a symlink at the temp path after removal and before write, causing MMM to write attacker-chosen content to an arbitrary file the user can write to.
- This is a classic TOCTOU symlink attack pattern; it is hard to exploit reliably locally, but it is well-known and should be avoided for security-oriented tooling.

Remediation:

- Use secure temp file creation:
  - create temp files with randomized names
  - open with O_EXCL semantics when possible
  - avoid following symlinks (platform-specific "no-follow" flags) when opening the destination
- Consider additional hardening when `modsFolder` or `configPath` is outside the project directory.

Confidence: Medium (attack requires local filesystem control/race, but the pattern is objectively unsafe).

---

### Critical - Network calls can hang indefinitely (no timeouts, no deadlines) (mmm-63.12)

Evidence:

- `http.Client` has no `Timeout`: `internal/httpClient/httpClient.go:124` (client created at `:125-131` without timeout).
- `init` uses `http.DefaultClient` directly (also no timeout): `cmd/mmm/init/init.go:66-69`.
- Command-level contexts are derived from `context.Background()` without deadlines: `main.go:119`.
- Request builder errors are routinely ignored (examples):
  - `internal/httpClient/downloader.go:50`
  - `internal/minecraft/minecraft.go:45`

Impact:

- MMM can hang forever in degraded network conditions (automation-hostile).

Remediation:

- Establish a consistent timeout policy (global and per-request):
  - set `http.Client.Timeout`, and/or
  - `context.WithTimeout` per operation.

Confidence: High.

---

### Critical - Download integrity is not enforced; non-200 responses can be written as jars (mmm-63.13)

Evidence:

- Downloader does not check HTTP status code before writing: `internal/httpClient/downloader.go:51` onward.
- No post-download hashing is performed to validate content against expected SHA-1 (lock file contains hashes, but the download path does not verify them).

Impact:

- HTML error pages, truncated downloads, or proxy responses can be written as `.jar`.
- Update flows can delete the previous working file without verifying the replacement.

Remediation:

- Require `StatusCode == 200` and return errors otherwise.
- Hash downloaded content and compare against expected hash before declaring success.
- On update, do not remove the old file until the new file is verified.
- In `internal/modinstall.Service.EnsureLockedFile`, re-hash the downloaded file and fail/cleanup when it does not match the expected hash (today it only hashes the existing local file before deciding to download): `internal/modinstall/modinstall.go:78-97`.

Confidence: High.

---

### High - Download URLs are not validated (can allow insecure HTTP or unexpected schemes) (mmm-63.14)

Evidence:

- Remote/lock-provided download URLs are accepted as-is and passed into `http.NewRequestWithContext`: `internal/httpClient/downloader.go:40-51`.
- Multiple command flows use `remote.DownloadURL` / `install.DownloadUrl` directly (examples):
  - `internal/modinstall/modinstall.go:51-73`
  - `cmd/mmm/install/install.go:302-305`
  - `cmd/mmm/update/update.go:404-429`

Impact:

- If an upstream API (or a tampered lock file) provides an `http://` URL, MMM will download jars over plaintext HTTP (MITM risk).
- If an upstream API provides a URL with an unexpected scheme, behavior becomes unpredictable (errors, crashes if request is nil, or "unsupported protocol" surprises). This is not just theoretical: URLs are an external input.

Remediation:

- Validate download URLs before any request:
  - require `https` scheme (or explicitly decide which schemes are allowed)
  - optionally enforce an allowlist of hosts for platform-controlled downloads (modrinth/curseforge/CDN), or at least log a loud warning for unknown hosts.

Confidence: Medium-High (depends on upstream guarantees, but lock file tampering is always possible).

---

### Medium - Downloader progress bookkeeping can overflow on 32-bit platforms (mmm-63.15)

Evidence:

- Downloader casts `response.ContentLength` (int64) to `int`: `internal/httpClient/downloader.go:65-66`.

Impact:

- On 32-bit platforms (or extremely large downloads), `int(response.ContentLength)` can overflow and produce negative or nonsensical progress ratios. This is primarily a correctness/UX issue, but it can also impact telemetry/perf if ratio is used downstream.

Remediation:

- Keep progress totals as `int64` and only convert to `int` when required by interfaces, with bounds checks.

Confidence: Medium (MMM likely targets 64-bit desktops; still a real bug pattern).

---

### High - HTTP retry implementation is unsafe for requests with bodies (retries can send empty bodies) (mmm-63.16)

Evidence:

- The retry loop reuses the same `*http.Request` across attempts: `internal/httpClient/httpClient.go:51-115`.
- At least one call site issues POST requests with a body: `internal/curseforge/files.go:115` uses `http.NewRequestWithContext(..., bytes.NewBuffer(body))`.

Impact:

- If a request with a body is retried, the body may already be consumed by the first attempt and later attempts can send empty or partial bodies. This can corrupt API behavior and make retries actively harmful.

Remediation:

- Given maintainer decision that retries must remain enabled for POST-like APIs, implement retries safely:
  - Require `request.GetBody != nil` and re-create `request.Body` per attempt via `GetBody` (or build a fresh request each attempt).
  - Ensure header mutation is not compounded across attempts.
  - Consider adding a request ID header (where supported) to reduce duplicate side-effect risk when the upstream actually is non-idempotent.

Confidence: Medium (depends on whether retries are enabled for POST paths and whether 5xx responses occur).

---

### High - RLHTTPClient can panic if constructed with a nil rate limiter (mmm-63.17)

Evidence:

- `NewRLClient` stores the passed limiter directly without validation: `internal/httpClient/httpClient.go:124-131`.
- `Do` unconditionally calls `client.Ratelimiter.Wait(...)`: `internal/httpClient/httpClient.go:65-66`.

Impact:

- Any code path that calls `httpClient.NewRLClient(nil)` and then uses the client will panic at runtime.
- Some READMEs show examples that could easily drift into passing nil (for example, "use default clients") unless constructors are defensive.

Remediation:

- Make `NewRLClient` safe by default: if limiter is nil, set a conservative default limiter or a `rate.NewLimiter(rate.Inf, 0)` explicitly (and then decide whether that is acceptable).

Confidence: Medium (most code uses `platform.DefaultClients` which guards against nil).

---

### High - RLHTTPClient retry loop ignores context cancellation and can hang shutdown (mmm-63.18)

Evidence:

- Retries sleep using `time.Sleep(retryConfig.Interval)` (not context-aware): `internal/httpClient/httpClient.go:102-104`.

Impact:

- If the user cancels the command (Ctrl+C) or the context is canceled, MMM can still block for the full retry interval (and potentially multiple intervals) before exiting.

Remediation:

- Replace `time.Sleep` with a context-aware wait (timer + select on `<-ctx.Done()`), or compute backoff at the caller with a cancellable context.

Confidence: High.

---

### High - i18n global initialization is not concurrency-safe and can race/panic (mmm-63.19)

Evidence:

- `T(...)` checks `localizer == nil` then calls `setup()` with no `sync.Once` / mutex: `internal/i18n/i18n.go:87-89`.
- `setup()` mutates global variables `localizer`, `bundle`, `localeProvider` and uses `panic` on failure: `internal/i18n/i18n.go:30-33`, `internal/i18n/i18n.go:44-47`, `internal/i18n/i18n.go:48-51`, `internal/i18n/i18n.go:71-73`.
- The codebase does call into i18n concurrently in production paths:
  - `update` spawns one goroutine per mod and constructs log events containing `i18n.T(...)` inside that goroutine: `cmd/mmm/update/update.go:195-201` and `cmd/mmm/update/update.go:282-376`.
  - `test` spawns one goroutine per mod and constructs debug/error messages containing `i18n.T(...)` inside that goroutine: `cmd/mmm/test/test.go:246-253` and `cmd/mmm/test/test.go:322-330`.

Impact:

- Any concurrent calls to `i18n.T(...)` early in process startup can race and produce undefined behavior, including panics or partially-initialized translation state.
- This is exacerbated by the TUI/CLI architecture where multiple subsystems may log/translate concurrently (for example, telemetry shutdown hooks plus command output).

Remediation:

- Guard `setup()` behind a `sync.Once` and make `T(...)` non-panicking (see earlier "i18n can panic" finding).

Confidence: High.

---

### High - Minecraft manifest cache is global, has no TTL, and is not concurrency-safe (mmm-63.20)

Evidence:

- Global cache pointer `latestManifest` is read/written without synchronization: `internal/minecraft/minecraft.go:32-33`, `internal/minecraft/minecraft.go:41-43`, `internal/minecraft/minecraft.go:59-60`.
- `ClearManifestCache()` mutates the cache without synchronization: `internal/minecraft/minecraft.go:34-36`.
- `getMinecraftVersionManifest` ignores request creation errors: `internal/minecraft/minecraft.go:45`.
- No HTTP status code validation before decoding JSON: `internal/minecraft/minecraft.go:54-58` (decodes even on non-200).

Impact:

- Potential data races in any command that calls `GetLatestVersion`, `IsValidVersion`, or `GetAllMineCraftVersions` concurrently (for example, parallel operations inside commands).
- Cache staleness: once populated, it is never refreshed unless `ClearManifestCache` is called.
- Treating non-200 bodies as JSON can cache an invalid manifest and poison later behavior.

Remediation:

- Add a `sync.Mutex`/`sync.RWMutex` or `sync.Once` + immutable caching approach; add a TTL if correctness requires fresh manifests.
- Validate request creation errors and HTTP status codes.

Confidence: High for thread-safety defect; Medium for real-world impact (depends on concurrency patterns).

---

### Medium - internal/packager is an empty module (dead code / misleading structure) (mmm-63.21)

Evidence:

- `internal/packager/` exists but contains no Go files: `find internal/packager -maxdepth 2 -type f` returns nothing.

Impact:

- Confuses maintainers and suggests incomplete or abandoned functionality without any comment/ADR explaining it.

Remediation:

- Remove the empty directory or add a minimal README explaining intended future purpose (if it must remain).

Confidence: High.

---
### High - HTTP retry/rate-limiting behavior is weak and does not match documented intent (mmm-63.22)

Evidence:

- Retry loop returns immediately on request error and does not retry network errors: `internal/httpClient/httpClient.go:80-92`.
- Rate limiting is typically disabled in commands by using `rate.NewLimiter(rate.Inf, 0)` (for example `cmd/mmm/test/test.go:106`).
- No handling for 429 or rate limit headers; only 5xx triggers retry: `internal/httpClient/httpClient.go:94-106`.

Impact:

- The CLI can overload upstream APIs and is more fragile under transient network failures than implied by docs.

Remediation:

- Implement retries for transient network errors (with backoff and context awareness).
- Implement 429 handling and platform-specific rate-limit header behavior (if this is a requirement for parity with Node).
- Establish a real default limiter (not infinite) at command boundaries.

Confidence: High.

---

### High - JSON decode errors are ignored in multiple network response paths (mmm-63.23)

Evidence (examples):

- Modrinth project decode ignores errors: `internal/modrinth/project.go:81-83`.
- Modrinth versions decode ignores errors: `internal/modrinth/version.go:128-130` and `internal/modrinth/version.go:156-158`.

Impact:

- Malformed or truncated API responses can be treated as valid zero values, leading to confusing downstream failures or silent incorrect behavior.

Remediation:

- Always handle JSON decode errors and propagate as typed API errors with context.

Confidence: High.

---

### High - Modrinth file selection ignores the "primary" file and can download the wrong artifact (mmm-63.24)

Evidence:

- Platform selection always uses `selectedVersion.Files[0]` without preferring `Primary`: `internal/platform/modrinth.go:46-52`.
- Other code explicitly prefers the primary Modrinth file: `cmd/mmm/scan/scan.go:708-714`.

Impact:

- For Modrinth versions that publish multiple files, MMM can install a secondary/non-primary file. This can be the wrong loader variant, a non-JAR artifact, or otherwise not the intended download.

Remediation:

- Centralize a single "choose Modrinth file" helper (prefer `Primary`, otherwise fall back deterministically), and use it in both scan/install/update/add flows.

Confidence: Medium-High (logic discrepancy is objective; impact depends on upstream projects publishing multiple files).

---

### Medium - CurseForge platform selection bypasses the low-level client helpers and loses typed error behavior (mmm-63.25)

Evidence:

- Platform layer issues its own GET request and returns a plain error for non-200 responses: `internal/platform/curseforge.go:79-103` and `internal/platform/curseforge.go:93-100`.
- Low-level CurseForge package wraps API failures into typed `globalErrors` failures: `internal/curseforge/files.go:61-82`.
- Low-level CurseForge package also explicitly paginates file listings, while the platform layer does not: `internal/curseforge/files.go:86-104` and `internal/platform/curseforge.go:79-103`.

Impact:

- Callers see inconsistent error types depending on whether they hit `internal/platform` vs `internal/curseforge`, undermining the intended "typed expected errors" contract described in `internal/platform/README.md`.
- If the filtered file listing endpoint is paginated by default (common), the platform selection may miss candidate files and incorrectly report "no compatible file" or pick an older/newer-than-intended file from the first page only.

Remediation:

- Prefer calling `internal/curseforge.GetFilesForProject` (or a filtered/paginated variant) from the platform layer so error typing and pagination behavior stays consistent.

Confidence: Medium.

---

### High - i18n can panic at runtime (availability and UX risk) (mmm-63.26)

Evidence:

- Translation setup uses `panic` on embedded file read failure: `internal/i18n/i18n.go:48-51` and on bundle load failure: `internal/i18n/i18n.go:71-73`.
- Translation function panics if more than one arg is passed: `internal/i18n/i18n.go:91-93`.

Impact:

- Any bug in a call site that accidentally passes two `Tvars` values will crash the entire program instead of returning a best-effort string.
- This is especially risky in user-facing CLI flows where translation keys are pervasive (help templates, UI labels, error messages).

Remediation:

- Make `T(...)` total and non-panicking:
  - return a fallback string on setup failures (for example `key` or `formatKeyAndArgs(key, args...)`)
  - validate args defensively (ignore extras or merge) rather than panic
- Restrict `MMM_TEST` behavior to test binaries rather than runtime env, or at least ensure `.env` cannot accidentally flip it (runtime `.env` autoload makes this easier).

Confidence: High.

---

### Medium - Locale detection is broken for common `LANG` formats (translations will not activate) (mmm-63.27)

Evidence:

- If `LANG` is set, it is used verbatim: `internal/i18n/i18n.go:111-116`.
- Locale parsing drops any `LANG` value that is not a well-formed BCP47-ish tag: `internal/i18n/i18n.go:156-159`.

Impact:

- Many systems set `LANG` to values like `fr_FR.UTF-8` / `de_DE.UTF-8` or `C.UTF-8`. These values do not parse as language tags, so they are skipped and the effective locale list becomes empty.
- Net effect: users on non-English systems will still get the default locale because their real locale is ignored.

Remediation:

- Normalize `LANG` before parsing by stripping common suffixes and modifiers (for example split on `.` and `@` first), then parse the base locale token.
- Add tests for `LANG=fr_FR.UTF-8`, `LANG=C.UTF-8`, and `LANG=en_US.UTF-8` to pin expected behavior.

Confidence: High (direct code-path evidence; behavior is a known property of `golang.org/x/text/language.Parse`).

---

### High - Modrinth API client ignores JSON decode errors (treats malformed API replies as success) (mmm-63.28)

Evidence:

- Response JSON decode errors are discarded and treated as success:
  - Project: `internal/modrinth/project.go:81-83`
  - Versions list: `internal/modrinth/version.go:128-130`
  - Version-by-hash: `internal/modrinth/version.go:156-158`

Impact:

- A transient upstream issue (truncated response, HTML error page with 200, proxy injection, partial body) can silently turn into "empty" structs/slices, which will then be interpreted as legitimate "not found / no compatible file / missing fields" behavior at higher layers.
- This weakens both correctness and observability because callers lose the real failure signal and may emit misleading user-facing errors.

Remediation:

- Check `Decode(...)` errors and wrap them in the existing platform error types (`globalErrors.ProjectApiErrorWrap(...)` and `VersionApiErrorWrap(...)`), preserving the underlying decode error for debugging.

Confidence: High.

---

### Medium - Config/lock existence checks ignore filesystem errors (misclassifies errors as "missing") (mmm-63.29)

Evidence:

- `ReadConfig` ignores `afero.Exists` errors and treats them as "not found": `internal/config/config.go:20-23`.
- `EnsureLock` ignores `afero.Exists` errors and treats them as "missing": `internal/config/lock.go:18-26`.

Impact:

- Permission errors, invalid paths, or I/O failures can be misreported as "file not found", producing confusing UX and potentially causing write attempts in unexpected states (for example trying to create a lock when the underlying error was a permission/FS failure).

Remediation:

- Treat `Exists` errors as errors (return them) rather than defaulting to `exists=false`.

Confidence: High.

---

### High - HTTP request construction errors are ignored (nil request panic risk) (mmm-63.30)

Evidence:

- Multiple call sites discard the `error` returned by `http.NewRequestWithContext(...)` and proceed as if the request is valid:
  - Downloader: `internal/httpClient/downloader.go:50-53`
  - Minecraft manifest: `internal/minecraft/minecraft.go:45-49`
  - Modrinth: `internal/modrinth/project.go:62-66`, `internal/modrinth/version.go:110-113`, `internal/modrinth/version.go:139-143`
  - CurseForge: `internal/curseforge/project.go:26-31`, `internal/curseforge/files.go:57-63`, `internal/curseforge/files.go:112-119`
  - Platform wrapper: `internal/platform/curseforge.go:80-85`

Impact:

- If any constructed URL is malformed (for example due to unexpected characters in a project ID, a corrupted lock file download URL, or a bad base URL), `http.NewRequestWithContext` can return `(nil, err)`.
- These paths then dereference `request` (for example in `client.Do(request)`), leading to a crash (nil pointer dereference) rather than a user-facing error.

Remediation:

- Handle and propagate `http.NewRequestWithContext` errors at every call site.
- Validate/normalize untrusted inputs before interpolating them into URLs (project IDs, download URLs, and any test-only base URL seams).

Confidence: High.

---

### High - Toolchain vulnerability posture is currently failing (mmm-63.31)

Evidence:

- `govulncheck -scan=module` reports multiple standard library vulnerabilities for `go1.25.0` (fixed in `go1.25.1+` / `go1.25.2+` / `go1.25.3` / `go1.25.5` depending on vuln; see IDs below).
- `govulncheck -scan=symbol ./...` fails with an internal type-checking error, so the strongest scan mode is not usable in this repo currently.

Impact:

- Your security posture cannot be credibly validated via the standard Go vuln tool at symbol-level.
- Even if the program does not exercise all vulnerable packages, shipping on a vulnerable toolchain without a plan is hard to justify.

Remediation:

- Upgrade Go toolchain to a patched `go1.25.x` release (at least `go1.25.5` per govulncheck output).
- Consider adding a `toolchain go1.25.5` directive to `go.mod` so hosts with older Go versions do not auto-select `go1.25.0` (the vulnerable .0 toolchain) when `GOTOOLCHAIN=auto` is in effect.
- Track govulncheck/x_tools compatibility with your chosen Go version and indirect deps.

Confidence: Medium-High.

---

### High - govulncheck evidence (current toolchain) (mmm-63.32)

Evidence:

- `govulncheck -scan=module` reports 13 stdlib vulnerabilities (examples):
  - `GO-2025-4175` (crypto/x509) fixed in `go1.25.5`
  - `GO-2025-4155` (crypto/x509) fixed in `go1.25.5`
  - `GO-2025-4007` (crypto/x509) fixed in `go1.25.3`
  - `GO-2025-3955` (net/http) fixed in `go1.25.1`
  - multiple others fixed in `go1.25.2`
- `govulncheck -scan=package ./...` reports 11 stdlib vulnerabilities in packages MMM depends on.
- `govulncheck -scan=symbol ./...` fails with:
  - `internal error: package "golang.org/x/sys/unix" without types was imported from "github.com/mattn/go-isatty"`

Impact:

- The current CI/workstation toolchain has known fixed vulnerabilities and cannot be fully symbol-scanned for reachability.

Remediation:

- Upgrade Go toolchain and re-run govulncheck; if symbol scan still fails, pin `golang.org/x/vuln` / `golang.org/x/tools` versions (or track upstream bug) until symbol scan works.

Confidence: High (direct evidence from local runs).

---

### Medium - No third-party license attribution / SBOM artifact for distributed binaries (mmm-63.33)

Evidence:

- A dependency license inventory exists (generated during this audit) but is not produced by CI or release tooling today:
  - `go-licenses report ./...` output captured at `/tmp/go-licenses.txt` (52 modules) with warnings at `/tmp/go-licenses.err` (for example, non-Go asm files and module HEAD URL warning).
- Release packaging only zips binaries and does not include any license/notice bundle:
  - `scripts/binaries.sh:5-7` creates `dist/*.zip` with only the executable (`zip -j ... build/.../mmm[.exe]`).
- No in-repo `THIRD_PARTY_NOTICES*`, `NOTICE*`, or `SBOM*` files were found (verified by searching tracked files during this audit).

Impact:

- This is a compliance and supply chain transparency gap for distributed artifacts, especially given the repo is GPL-3.0 (`LICENSE` at repo root) and vendors many third-party dependencies with varying licenses (MIT/BSD/Apache/MPL observed in the inventory).
- Users and downstream redistributors have no machine-readable or human-readable way to see the dependency license set for a given release artifact.

Remediation:

- Add a reproducible license/SBOM artifact as part of release builds (and ideally CI):
  - Generate `THIRD_PARTY_NOTICES.txt` (or similar) from `go-licenses report` (plus any manual notices required).
  - Consider generating an SBOM (SPDX or CycloneDX) for each release build and attaching it to GitHub Releases.
  - Ensure `dist/*.zip` includes `LICENSE` and the third-party notices file (or ship them as sibling assets with clear labeling).
- Add a CI gate that fails if license inventory generation fails or changes unexpectedly (at least on release branches).

Confidence: Medium-High (inventory evidence is strong; compliance obligations depend on your distribution model and legal interpretation).

---

### Critical - Telemetry shutdown is not time-bounded in the production code path (hang risk) (mmm-63.34)

Evidence:

- Telemetry timeout behavior is only applied when `ctx == nil`: `internal/telemetry/telemetry.go:381-400`.
- Production always passes a non-nil context (no deadline) into `telemetry.Shutdown`: `main.go:155` (the context is derived from `context.Background()` at `main.go:119`).

Impact:

- `defaultFlushTimeout` / `baseFlushTimeout` is effectively dead in production; telemetry close can block forever if the PostHog client hangs during shutdown.
- Because shutdown runs inside the lifecycle handler, a hang here can prevent process exit and prevent perf export (`main.go:156-160`).

Remediation:

- Always bound shutdown time by wrapping the passed context with a timeout (for example `ctx, cancel := context.WithTimeout(ctx, snapshot.flushTimeout)`), or by applying a timeout when the provided context has no deadline.
- Add a regression test that asserts `Shutdown(nonNilContextWithoutDeadline)` returns within `flushTimeout`.

Confidence: High (direct code-path evidence). Actual hang likelihood: Medium (depends on posthog-go behavior and network conditions).

---

### High - Telemetry "anonymized" claim is not accurate as implemented (mmm-63.35)

Evidence:

- Telemetry uses machine ID as `DistinctId`: `internal/telemetry/telemetry.go:120` and sends it: `internal/telemetry/telemetry.go:167-171`.
- Telemetry ships performance traces (a full span tree) and error strings:
  - `internal/telemetry/telemetry.go:355-372`
  - `internal/telemetry/telemetry.go:470-472`

Impact:

- This is pseudonymous at best. If docs claim anonymized, that is inaccurate.

Remediation:

- Update docs to describe exactly what is collected (and why).
- Consider minimizing payload (truncate errors, strip path-like fields, hash IDs).

Confidence: High.

---

### High - Telemetry payloads can leak local filesystem paths and potentially sensitive user inputs (mmm-63.36)

Evidence:

- Telemetry stores the full error string in the session payload: `internal/telemetry/telemetry.go:222-225` and `internal/telemetry/telemetry.go:470-472`.
- Telemetry includes the full perf span export tree on every shutdown (not only when `--perf` is set): `internal/telemetry/telemetry.go:355-371`.
- Perf tracing is enabled unconditionally in the main entrypoint: `main.go:115-119`.
- Perf tracing uses `trace.AlwaysSample()` (maximum collection) when enabled: `internal/perf/perf.go:42-45`.
- Perf spans commonly include full request URLs and local file paths:
  - `internal/httpClient/httpClient.go:31-36` records `url` and `host` for every request.
  - `internal/modrinth/modrinthClient.go:24` records `url` for Modrinth requests.
  - `internal/curseforge/curseforgeClient.go:23` records `url` for CurseForge requests.
  - `internal/httpClient/downloader.go:41-46` records `url` and `path` for downloads.
- Perf export normalization rewrites "path-like" keys but does not scrub `url` (including query strings): `internal/perf/export.go:291-340`.
- No evidence in this repo that auth header values are recorded into perf attributes or telemetry payloads:
  - Perf attributes record request URLs/hosts, not headers: `internal/httpClient/httpClient.go:31-36`.
  - Platform HTTP wrappers set auth headers but only record the URL in spans: `internal/modrinth/modrinthClient.go:23-36`, `internal/curseforge/curseforgeClient.go:22-35`.
- Some error strings explicitly include local filesystem paths:
  - `ConfigFileNotFoundException.Error()` embeds `Path`: `internal/config/ConfigErrors.go:18-20`.
- Some command telemetry includes user-provided values that can be absolute paths or otherwise sensitive:
  - `init` includes `modsFolder` in `Arguments`: `cmd/mmm/init/init.go:241-247`.
  - `remove` includes the user-provided lookup strings (mod names/IDs) in `Arguments`: `cmd/mmm/remove/remove.go:93-97`.

Impact:

- Even with telemetry opt-out available, the default ("on") behavior can upload machine-local path information and user-entered strings to PostHog. (Maintainer clarified that request URLs are intended to be included, but secret header values must not be.)

Remediation:

- Treat telemetry payload fields as untrusted/sensitive:
  - Do not send raw error strings; send error categories and small allowlisted details only.
  - Normalize or redact filesystem paths (prefer "is_absolute" booleans or relative-to-config paths).
  - Avoid sending arbitrary user-entered strings; use counts/enums instead.
- Do not send the full perf span tree by default. If you need performance telemetry:
  - send aggregate metrics only (durations, counts, coarse status codes)
  - scrub URLs to host + path only (drop query strings), or avoid URLs entirely
  - gate trace attachment behind an explicit opt-in flag/env var
  - document the attachment behavior clearly
- Add an explicit invariant test: ensure telemetry payloads and perf exports never include API key values (for example: if `MODRINTH_API_KEY=test-secret`, no captured JSON should contain `test-secret`). This is the only robust way to enforce the "URLs yes, headers no" policy over time, especially if instrumentation changes.

Confidence: High.

---

### Low - `mmm-perf.json` uses permissive permissions (team considers perf artifacts non-sensitive) (mmm-63.37)

Evidence:

- Perf export creates the output directory with mode `0755`: `internal/perf/export.go:67-69`.
- Perf export writes `mmm-perf.json` with mode `0644`: `internal/perf/export.go:71-78`.
- Perf spans include full request URLs and local paths (see the telemetry findings above), and the perf exporter does not scrub URL values: `internal/perf/export.go:291-340`.

Impact:

- On multi-user systems, other local users may be able to read `mmm-perf.json` and learn API hosts/endpoints, download URLs, and some local path context. Maintainer clarified these artifacts are not considered sensitive, so treat this primarily as an operational/defense-in-depth note rather than a security blocker.

Remediation:

- Default perf artifacts to least-privilege permissions (for example: `0700` directories and `0600` files).
- Consider scrubbing or omitting `url` values (or at minimum strip query strings) before writing the perf tree to disk.

Confidence: High.

---

### High - TUI key bindings are partially broken under non-English locales (mmm-63.38)

Evidence:

- Key binding definitions use localized strings as key identifiers, not just as help text:
  - `internal/tui/keybindings.go:101` uses `i18n.T(\"key.pgdown\")` as a key name
  - `internal/tui/keybindings.go:108` uses `i18n.T(\"key.pgup\")` as a key name
- Locale files translate those key identifiers into human labels (not Bubble Tea key names), for example German uses localized labels: `internal/i18n/lang/de-DE.json:94-99`.

Impact:

- When the locale is not English, bindings for page navigation and some other keys will not match actual key events (Bubble Tea key event strings are not localized), so PgUp/PgDn navigation can stop working.
- This is both a correctness defect and an accessibility defect (keybindings should be stable regardless of language).

Remediation:

- Never localize key identifiers passed to `key.WithKeys(...)`. Use canonical key strings only (for example `pgdown`, `pgup`, `home`, `end`).
- Localize only the help/label side (`key.WithHelp(...)`).

Confidence: High.

---

### High - CI supply chain risk due to unpinned GitHub Actions (mmm-63.39)

Evidence:

- CI uses an action pinned to a moving branch reference: `.github/workflows/ci.yml:20` uses `meza/action-go-setup@main`.
- CI uses a floating Go toolchain version: `.github/workflows/ci.yml:24` sets `go-version: 'latest'`.
- Renovate configuration extends a centrally managed team preset: `.github/renovate.json:3` uses `github>stateshifters/renovate-common` (maintainer clarified this upstream config is owned by the team and changes are intentional).

Impact:

- A compromised action repository or a malicious force-push to `main` can compromise CI and release artifacts.
- `go-version: latest` makes builds non-reproducible and can break unexpectedly when Go releases (and it undermines response to toolchain-level vulnerabilities).
- A centrally managed Renovate preset can change dependency update policy without a change in this repo; that is acceptable if you treat it as an intentional, governed org-level policy, but it reduces per-repo auditability unless you version/tag preset changes.

Remediation:

- Pin actions to immutable commit SHAs (preferred) or at least version tags.
- Pin Go to a specific patch version compatible with `go.mod` (and keep it updated), for example `1.25.x`.
- Reduce workflow permissions to least privilege (current workflow requests broad write permissions while not actually releasing): `.github/workflows/ci.yml:3-7`.
- Consider using `make build-dev` (or `BUILD_REQUIRE_TOKENS=0`) for `pull_request` builds so forks are not blocked by missing secrets (current workflow runs `make build` and expects secrets): `.github/workflows/ci.yml:33-38` and `Makefile:9`.
- If you want immutable audit trails of Renovate policy per repo, pin the preset by tag/commit; otherwise explicitly document that Renovate policy is centralized and changes are expected.

Confidence: High.

---

### Medium - Automated "approve Renovate PRs without review" guidance conflicts with supply chain safety (mmm-63.40)

Evidence:

- Repository-provided code review instructions explicitly say to "stand-down and approve" Renovate dependency update PRs: `.github/instructions/github/code-review.md:23-24`.
- The Renovate preset this repo extends (`github>stateshifters/renovate-common`) enables broad automerge behavior for Go module updates:
  - It sets `automerge: true` for `gomod` updates, including `<1.0.0` dependencies: external preset `https://raw.githubusercontent.com/stateshifters/renovate-common/main/golang.json` (repro: `curl -fsSL .../golang.json`) at lines 9-23.
  - It only adds an "approval" requirement for major updates, not for minor/patch: same file lines 30-36.

Impact:

- This removes human review pressure from one of the highest-risk change categories (dependency updates).
- It is especially risky given:
  - CI currently does not run any vulnerability scanning as claimed in `SECURITY.md` (see the next CI gate finding).
- Renovate policy (including automerge) is centrally managed by the team (per maintainer clarification), which is fine, but it increases the importance of having machine-verifiable CI security gates because changes can merge without meaningful review.
- Even if Renovate is "fully automated", auto-approving without verification is not compatible with a strict security posture. At minimum, there should be a machine-verifiable gate (dependency review action, govulncheck, etc.) that is required before merge.

Remediation:

- Replace "auto-approve renovate" guidance with "approve only after passing security gates" guidance, and document what those gates are.
- If the intent is truly "no human review", add hard CI policies: dependency review, govulncheck, and (optionally) CodeQL or equivalent.

Confidence: Medium (impact depends on how strictly the team follows these AI instructions in practice, but the guidance is unambiguous).

---

### High - CI does not enforce basic quality/security gates it claims or implies (mmm-63.41)

Evidence:

- Only CI workflow runs `make coverage`, `make test-race`, and `make build`: `.github/workflows/ci.yml:27-39`.
- CI does not run:
  - `make fmt` (or any gofmt gate)
  - `go vet` / static analysis gate (`staticcheck` or `golangci-lint`)
  - `govulncheck` (contradicts `SECURITY.md:3` claims)

Impact:

- Formatting drift, lint regressions, and vulnerable dependency/toolchain updates can land without any CI signal.
- This undermines the project's "strict hygiene" posture and makes audits less actionable because regressions are not prevented.

Remediation:

- Add explicit CI steps for formatting (`make fmt` or a gofmt check), lint (`staticcheck` or `golangci-lint`), and vuln scanning (govulncheck at least `-scan=module`/`-scan=package` until symbol scan is stable).

Confidence: High.

---

### Medium - CI workflow conflates CI and release concerns and uses excessive permissions/secrets (mmm-63.42)

Evidence:

- Workflow is named "Verify and Release" but release is commented out: `.github/workflows/ci.yml:1` and `.github/workflows/ci.yml:40-45`.
- Workflow requests broad write permissions for all events (including pull requests): `.github/workflows/ci.yml:3-7`.
- Workflow runs on `pull_request` but relies on secrets for `make build`: `.github/workflows/ci.yml:9-12` and `.github/workflows/ci.yml:33-38`.

Impact:

- Increased blast radius: write permissions on PR runs amplify supply chain risk (a compromised action or dependency can write to repo contents, issues, PRs).
- External contributors/fork PRs will likely fail build due to missing secrets, reducing contribution viability (even if secrets are required for "real" builds).

Remediation:

- Split workflows:
  - CI workflow for PRs with least privilege (read-only permissions, no secrets required; skip/soften build step or use `BUILD_REQUIRE_TOKENS=0`).
  - Release workflow on protected branches/tags with required secrets and carefully pinned actions.

Confidence: High.

---

### Medium - Repo hygiene: local tooling config is committed (mmm-63.43)

Evidence:

- `.claude/settings.local.json` is tracked in git (file name implies it is local-only).

Impact:

- Adds noise and encourages committing machine-local agent configuration; can become a secret or policy leak vector over time.

Remediation:

- Decide whether this is intended to be a committed project policy file (rename accordingly) or remove it from version control and ensure it is ignored.

Confidence: Medium.

---

### Low - Repo hygiene: `.editorconfig` is tracked as executable (mode bit drift) (mmm-63.44)

Evidence:

- `.editorconfig` is stored in git with mode `100755` (executable): verified during this audit via `git ls-files -s .editorconfig` showing `100755 ... .editorconfig`.

Impact:

- Misleading metadata: tools and downstream packaging can treat `.editorconfig` as an executable file.
- Indicates broader risk of mode-bit drift being missed, especially since many environments set `core.filemode=false` (this audit checkout has `git config core.filemode` set to `false`).

Remediation:

- Normalize tracked modes so only actual scripts/binaries are executable, and ensure CI or hooks prevent mode-bit drift.

Confidence: High (direct repository metadata evidence).

---

### Low - Lefthook configuration bypasses Makefile targets (workflow drift risk) (mmm-63.45)

Evidence:

- Lefthook is configured to run some checks via the Go toolchain directly, not via documented `make` targets:
  - `go mod tidy` runs directly: `lefthook.yml:12-15`.
  - `go vet` runs directly: `lefthook.yml:16-21`.
- Only one hook uses the Makefile gate (`make coverage`): `lefthook.yml:9-11`.

Impact:

- Inconsistent developer experience: local hooks can behave differently from CI (and from documented workflow), and may apply repo-specific flags inconsistently.
- `go mod tidy` as a pre-commit step can create churn and unexpected diffs (especially if run under different Go versions).

Remediation:

- Prefer driving hook behavior through the Makefile (for example: `make fmt`, `make coverage`, `make test-race`, and a `make tidy` target) so local and CI behavior stays aligned.
- If `go mod tidy` remains a hook, document the expected Go toolchain version and the intended policy for go.mod/go.sum changes.

Confidence: High (direct file evidence).

---

### Low - Repo documentation violates the "ASCII-only punctuation" rule (non-ASCII dashes) (mmm-63.46)

Evidence:

- `AGENTS.md` includes non-ASCII punctuation characters:
  - `AGENTS.md:29` uses an em dash in "during the session-not just at the end".
  - `AGENTS.md:144` uses an en dash in "1-2 sentences".
  - `AGENTS.md:170` uses an em dash in "guideline-it's".
- `memory.md` (tracked markdown in repo root) also contains non-ASCII punctuation (examples):
  - `memory.md:3` contains an em dash in "... cmd bug-cross-reference ..."
  - `memory.md:8` contains an em dash in "... defaults-need ldflags-driven ..."

Impact:

- This directly violates the stated repository rule that markdown must be ASCII (and your clarified preference to ban smart punctuation like em dashes and en dashes).
- It also makes automated linting for ASCII-only docs harder if the policy doc itself violates the rule.

Remediation:

- Replace non-ASCII punctuation in markdown files with ASCII equivalents:
  - Use `-` instead of en/em dashes.
  - Use an ASCII hyphen between ranges (for example, write `1-2` using ASCII characters only).

Confidence: High (direct evidence).

---

### Low - `.gitattributes` line ending rules likely do not apply to `.cmd` and `.bat` files (mmm-63.47)

Evidence:

- `.gitattributes:10-11` uses brace expansion-style patterns:
  - `*.{cmd,[cC][mM][dD]} text eol=crlf`
  - `*.{bat,[bB][aA][tT]} text eol=crlf`

Impact:

- Git attribute patterns do not support shell brace expansion; these rules likely do not match any files.
- If the intent is to force CRLF for Windows batch scripts (as described in `.gitattributes:8-11`), the repo is not actually enforcing it.

Remediation:

- Replace these patterns with valid git attribute patterns, for example:
  - `*.[cC][mM][dD] text eol=crlf`
  - `*.[bB][aA][tT] text eol=crlf`

Confidence: Medium-High (git attribute pattern behavior is well-defined; confirm by adding a small `.cmd` file and observing attribute application in a maintainer-controlled environment).

---

### Medium - Release scripts are not robust/portable and can silently produce wrong outputs (mmm-63.48)

Evidence:

- `scripts/prepare.sh`:
  - Does not use strict mode (`set -euo pipefail`) and does not validate arguments: `scripts/prepare.sh:1-13`.
  - Uses `sed -i` without portability handling; BSD/macOS `sed` requires a different `-i` form: `scripts/prepare.sh:10-11`.
  - Uses `$HELP_URL` without validating it is set, which can replace the placeholder with an empty string: `scripts/prepare.sh:11`.
  - Mutates a tracked Go source file in-place via `sed` as part of release preparation: `scripts/prepare.sh:10-11`.
- `scripts/binaries.sh`:
  - Does not enable strict mode and does not verify inputs or ensure `dist/` exists before writing: `scripts/binaries.sh:1-7`.

Impact:

- Local/off-CI release runs can fail or generate inconsistent artifacts depending on host OS/tooling (especially macOS sed behavior).
- Lack of strict mode increases the chance of partial/incomplete placeholder injection with a "successful" exit, which is the worst failure mode for release automation.

Remediation:

- Harden scripts:
  - Add strict mode and explicit argument validation.
  - Make `sed` usage portable or constrain release execution to a known Linux environment (and document that explicitly).
  - Prefer `-ldflags -X` injection over in-place rewriting of tracked Go sources.
  - Ensure `dist/` is cleaned/created deterministically before zipping, and consider adding checksums/signatures as part of the same pipeline.

Confidence: High (direct code-path evidence).

---

### High - GitHub Actions workflow likely depends on undocumented behavior (no explicit checkout) (mmm-63.49)

Evidence:

- The only workflow does not include an explicit `actions/checkout` step: `.github/workflows/ci.yml:19-39`.
- The workflow uses a custom action `meza/action-go-setup@main` (unpinned) and then immediately runs `make coverage` / `make build`: `.github/workflows/ci.yml:20-38`.

Impact:

- If `meza/action-go-setup` does not also check out the repository, this workflow is non-functional (there will be no source tree to run `make` against).
- Even if it does check out, hiding repository checkout inside a custom "setup" action is a maintenance hazard:
  - it obscures the trust boundary (checkout + token usage + go setup all in one place)
  - it makes it harder to reason about permissions and what code runs before the repo is present
  - it increases blast radius if the action is compromised

Remediation:

- Add an explicit `actions/checkout@<pinned>` step (pinned to SHA) and keep the custom action focused on Go installation only.
- Document the intended behavior of `meza/action-go-setup` (what it does with `GH_TOKEN`, whether it checks out code, and what outputs it provides).

Confidence: Medium (the workflow may still work if the custom action checks out the repo, but the current state is not auditable from this repository).

---

### Medium - Stale bot configuration is present but there is no evidence it is active (mmm-63.50)

Evidence:

- Stale policy exists: `.github/stale.yml:1-20`.
- There is no workflow referencing stale automation; the workflow directory contains only `.github/workflows/ci.yml`.

Impact:

- The repo may be relying on an external GitHub App / bot configuration that is not visible here.
- Contributors cannot predict whether/when issues will be auto-labeled or auto-closed, and maintainers cannot reproduce the behavior from the repo alone.

Remediation:

- If this is intended to be enforced by a GitHub App, document which app and how it is configured (and confirm it actually consumes `.github/stale.yml`).
- If this is intended to be enforced by Actions, add a workflow that uses `actions/stale` (pinned) and remove ambiguity.

Confidence: Medium.

---

### Medium - Security vulnerability reporting flow encourages public disclosure (mmm-63.51)

Evidence:

- `SECURITY.md` instructs reporters to open a public issue for vulnerabilities: `SECURITY.md:8-10`.

Impact:

- This increases the chance of 0-day vulnerabilities being disclosed publicly before a fix exists.
- It conflicts with common GitHub best practices (private security advisories / private reporting channel).

Remediation:

- Provide a private reporting channel (GitHub Security Advisories / private report link, or a security email) and reserve issues for post-fix, post-embargo tracking.
- Document supported versions and how fixes are backported (or explicitly say "only latest is supported").

Confidence: High.

---

### Low - Issue templates are missing key debugging fields and do not steer security reports (mmm-63.52)

Evidence:

- Bug report template does not ask for logs (`--debug` output) or repro steps beyond freeform text; it does request command/config/lock: `.github/ISSUE_TEMPLATE/bug_report.yml:9-71`.
- Issue template config enables blank issues: `.github/ISSUE_TEMPLATE/config.yml:1`.
- There is no `contact_links` entry for security reporting (only discussions): `.github/ISSUE_TEMPLATE/config.yml:2-5`.

Impact:

- Maintainers are likely to spend extra time extracting the minimum info needed to reproduce.
- Allowing blank issues increases triage noise; security reports may get filed as public issues instead of being routed privately.

Remediation:

- Add a dedicated "Security vulnerability" contact link (private channel) and point users to `SECURITY.md`.
- Add explicit fields for:
  - expected vs actual
  - repro steps with copy/paste command lines
  - sanitized debug logs (explicitly warn users not to paste API keys)
- Consider disabling blank issues to force templated intake.

Confidence: Medium.

---

### Low - CODEOWNERS is missing (review ownership and escalation unclear) (mmm-63.53)

Evidence:

- No `CODEOWNERS` file exists in the repository (none under repo root or `.github/`).

Impact:

- Review ownership and escalation paths are informal and can drift.
- In security-sensitive areas (networking, filesystem, release), lack of explicit ownership increases the chance of unreviewed changes landing.

Remediation:

- Add a `CODEOWNERS` file (for example under `.github/CODEOWNERS`) covering:
  - release/CI (`.github/workflows/*`, `scripts/*`)
  - networking/telemetry (`internal/httpClient/*`, `internal/telemetry/*`, `internal/perf/*`)
  - filesystem/data safety (`internal/config/*`, `internal/modinstall/*`, `cmd/mmm/*`)

Confidence: High.

---

### Low - Missing PR template and support policy file (triage and contributor UX) (mmm-63.54)

Evidence:

- No PR template file exists (no `.github/pull_request_template.md` and no `PULL_REQUEST_TEMPLATE.md` under repo root).
- No `SUPPORT.md` file exists to route common support requests (the only explicit routing is a discussions link in `.github/ISSUE_TEMPLATE/config.yml:2-5`).

Impact:

- PRs may arrive without consistent checklists (tests run, docs updated, risk notes), increasing review burden and reducing auditability.
- Users may file issues for support questions, increasing triage noise.

Remediation:

- Add `.github/pull_request_template.md` with a minimal checklist aligned to repo gates (`make coverage`, `make test-race`, `make build`) and explicit "no secrets in logs" reminder.
- Add `SUPPORT.md` (or equivalent) pointing users to discussions for questions, issues for confirmed bugs, and the private security reporting channel for vulnerabilities.

Confidence: Medium (absence-of-file finding is easy to verify, but impact depends on current maintainer workflow).

---

### Medium - `FUNDING.yml` location may not be recognized by GitHub (mmm-63.55)

Evidence:

- `FUNDING.yml` exists at repo root: `FUNDING.yml:1`.

Impact:

- GitHub's sponsor button configuration is conventionally read from `.github/FUNDING.yml`. If GitHub does not read the root-level file, funding links may not show up as intended.

Remediation:

- Confirm whether GitHub is picking up the current `FUNDING.yml`. If not, move it to `.github/FUNDING.yml`.

Confidence: Medium (depends on GitHub behavior; requires confirmation in the hosted repo UI).

---

### Medium - `CONTRIBUTING.md` does not reflect the actual Go workflow and is not enforceable as written (mmm-63.56)

Evidence:

- `CONTRIBUTING.md` is generic and does not reference the project's actual `make` targets or Go tooling, but requires "tests and linters" without specifying how to run them: `CONTRIBUTING.md:8-15`.
- It requires Conventional Commits: `CONTRIBUTING.md:12`, but the semantic-release step is currently commented out in CI: `.github/workflows/ci.yml:40-45`.

Impact:

- Contributors are not given the correct, repo-specific path to validate changes.
- Requirements become "paper policy": easy to violate unintentionally, and hard for maintainers to enforce consistently.

Remediation:

- Update `CONTRIBUTING.md` to reflect current Go workflow (`make fmt`, `make coverage`, `make test-race`, `make build`) and clarify whether Conventional Commits are actually enforced today.

Confidence: High.

---

### Low - `CLAUDE.md` contains conflicting agent instructions (repo hygiene) (mmm-63.57)

Evidence:

- `CLAUDE.md:1` instructs using a different agent persona ("senior-engnieer") regardless of context, which conflicts with the explicit Auditor persona requested for this audit.

Impact:

- Confusing or conflicting meta-instructions increase the chance of inconsistent automation behavior and reduce trust in repo guidance files.

Remediation:

- Either delete `CLAUDE.md` if it is obsolete, or rewrite it to be consistent with `AGENTS.md` and the repo's desired tooling workflow.

Confidence: Medium.

---

### High - Documentation and tests depend on env-driven API base URLs (must be refactored) (mmm-63.58)

Evidence:

- Internal docs explicitly document base URL overrides via env vars:
  - `internal/platform/README.md:121-125`
  - `internal/modrinth/README.md:34`
  - `internal/curseforge/README.md:24`
- Tests use env vars to redirect API calls to `httptest` servers (examples include `internal/platform/platform_test.go` and `internal/modrinth/versions_test.go`).

Impact:

- After removing env-driven base URL overrides (required per maintainer decision), both docs and tests will break unless migrated to an explicit injection mechanism.

Remediation:

- Replace env-var test strategy with a stable, explicit test seam:
  - package-level base URL variable plus `SetBaseURLForTesting(...) func() restore`, or
  - dependency injection (pass base URL into constructors), or
  - build-time injected test defaults (only for tests).

Confidence: High.

---

### Medium - Coverage enforcement does not cover command packages (mmm-63.59)

Evidence:

- `make coverage` enforces 100% coverage (see `Makefile` target `coverage` and `tools/coverage`).

Impact:

- The project policy states 100 percent coverage; the current gate enforces 100 percent only for internal packages, not `cmd/...` or root package behavior.

Remediation:

- Expand the coverage target to include all packages (or add a second gate for command packages) and ensure the enforcement logic matches the stated policy.

Confidence: High.

---

### Medium - Command packages are overlarge and mix concerns (maintainability and correctness risk) (mmm-63.60)

Evidence:

- Several command implementation files are very large and contain multiple layers of concerns (CLI parsing, TUI, domain logic, filesystem I/O, networking, telemetry):
  - `cmd/mmm/install/install.go` (684 lines in this workspace)
  - `cmd/mmm/scan/scan.go` (803 lines in this workspace)
  - `cmd/mmm/update/update.go` (564 lines in this workspace)
  - `cmd/mmm/add/add.go` (544 lines in this workspace) plus `cmd/mmm/add/tui.go` (605 lines in this workspace)
  - `cmd/mmm/init/init.go` (520 lines in this workspace)
- These files also duplicate multiple helper behaviors across commands (see the duplication finding above), which is a common symptom of missing shared domain services.

Impact:

- Harder to audit and reason about security invariants (path safety, retries, URL validation) because the logic is scattered and intertwined with UI concerns.
- Higher regression risk: small changes require touching large files with broad responsibilities.

Remediation:

- Split command packages into thin orchestration layers and move reusable domain logic into internal packages with tight unit tests.
- Establish a consistent "deps + service" pattern per command (the repo already uses deps structs; extend the pattern by pulling business logic out of cobra RunE closures).

Confidence: High.

---

### Medium - `--quiet` flag description does not match behavior (mmm-63.61)

Evidence:

- Flag says "Suppress all output": `cmd/mmm/root.go:31`.
- Logger always prints errors regardless of `quiet`: `internal/logger/logger.go:38-44`.
- Multiple commands intentionally force important output even when `--quiet` is set:
  - `list` always prints the rendered view with `forceShow=true`: `cmd/mmm/list/list.go:131-133`.
  - `test` prints failure summaries with `forceShow=true` (while suppressing the success message in quiet mode): `cmd/mmm/test/test.go:283-303`.
- `--debug` also weakens "quiet" by design: `Logger.Log` prints when `debug=true` even if `quiet=true` (it only suppresses when `quiet && !forceShow && !debug`): `internal/logger/logger.go:24-36`.

Impact:

- Automation users can rely on `--quiet` and still receive stderr output, which may be acceptable, but then the flag description is incorrect (and expectations are violated).
 - The current behavior is closer to "suppress prompts and non-essential output, but still print errors and required results" which should be documented consistently.

Remediation:

- Either change the flag description to "suppress non-error output" (or similar), or change behavior to fully silence output (usually a bad idea for CLIs, so prefer correcting the description and being explicit).

Confidence: High.

---

### Medium - Help formatting assumes stdout is a TTY; non-TTY output can be malformed (mmm-63.62)

Evidence:

- Help template wrapping width is derived from `term.GetSize(int(os.Stdout.Fd()))` with ignored errors: `cmd/mmm/root.go:92-97`.

Impact:

- When stdout is not a TTY (piped output, CI logs), `term.GetSize` can fail and `width` can be `0`, leading to unpredictable wrapping behavior.

Remediation:

- On error or width <= 0, fall back to a sane default (for example 80) or disable wrapping modifications entirely when stdout is not a TTY.

Confidence: Medium.

---

### Medium - .mmmignore semantics break when the mods folder is outside the config directory; path matching is not Windows-robust (mmm-63.63)

Evidence:

- Ignore patterns are rooted at `meta.Dir()` (config directory), but mod files may live outside that root when `modsFolder` is absolute or points elsewhere: `cmd/mmm/install/install.go:625-658`.
- `IsIgnored` enforces "path must be under rootDir" using `strings.HasPrefix` on cleaned paths: `internal/mmmignore/mmmignore.go:42-47` (case-sensitive, path-string-based check).

Impact:

- If `modsFolder` is outside the config directory, `.mmmignore` rules will never match mod files.
- On Windows, case-insensitive paths can cause false negatives (string prefix checks are case-sensitive), reducing reliability of ignore behavior.

Remediation:

- Clarify what `.mmmignore` roots to (config dir vs mods dir) and implement accordingly.
- Replace the prefix check with a robust path containment check (volume-aware on Windows; consider `filepath.Rel` + error handling).

Confidence: Medium.

---

### Medium - `init --mods-folder` help/prompt text is incorrect and leaks machine-specific paths (mmm-63.64)

Evidence:

- The `--mods-folder` flag help text interpolates the current working directory into the usage string: `cmd/mmm/init/init.go:112-114` with translation key `cmd.init.usage.mods-folder` in `internal/i18n/lang/en-GB.json:33`.
- Running `mmm init --help` shows a machine-specific path hint (example from this environment): "full or relative path from /work".
- Actual path resolution for `modsFolder` is relative to the config directory, not CWD: `internal/config/metadata.go:28-33` (relative to `meta.Dir()`).
- The TUI prompt text also claims "relative path from the current directory": `internal/i18n/lang/en-GB.json:29`.

Impact:

- Users are misled about what a "relative" path means when `--config` points to a file in another directory (the most common non-default usage).
- The help output becomes non-portable and environment-specific, and it can leak local directory paths in pasted help output/screenshots.

Remediation:

- Make help text stable and correct:
  - describe relative paths as "relative to the config file directory" (or whatever the intended contract is)
  - avoid interpolating runtime-specific absolute paths into help output
- Align TUI prompt wording with the actual resolution logic.

Confidence: High.

---

### Low - Branding/grammar issues in i18n strings reduce perceived quality and trust (mmm-63.65)

Evidence:

- Root description uses "Curseforge" instead of "CurseForge": `internal/i18n/lang/en-GB.json:2`.
- Help command short text has a grammar error: `internal/i18n/lang/en-GB.json:7` ("Get help for the any command").

Impact:

- Reduces polish and user trust, especially for a CLI intended for automation and broad distribution.

Remediation:

- Fix the English locale strings and add a quick "spellcheck" review pass for user-facing strings as part of release readiness.

Confidence: High.

---

### Medium - User-facing strings bypass i18n and include emoji (mmm-63.66)

Evidence:

- Remove command uses hard-coded English strings and an emoji check mark:
  - `cmd/mmm/remove/remove.go:135`
  - `cmd/mmm/remove/remove.go:143`
  - `cmd/mmm/remove/remove.go:171`
- Init command logs an English string directly:
  - `cmd/mmm/init/init.go:407`

Impact:

- Violates the stated i18n discipline in the project TUI guidelines and creates inconsistent UX across commands/locales.
- Emoji output can be undesirable in automation logs and is not guaranteed to render consistently across terminals.

Remediation:

- Route user-facing strings through `internal/i18n` and use existing `internal/tui` icon helpers where appropriate.

Confidence: High.

---

### Medium - CLI output uses emoji and non-ASCII glyphs even in non-TTY (automation UX risk) (mmm-63.67)

Evidence:

- Shared icon helpers always return emoji, even when `colorize=false` (non-TTY): `internal/tui/icons.go:3-17`.
- Multiple non-TUI code paths use these emoji icons in normal output:
  - `cmd/mmm/scan/scan.go:764-781` and `cmd/mmm/scan/scan.go:786-797` (uses `tui.SuccessIcon` / `tui.ErrorIcon` with `colorize` derived from `IsTerminalWriter`)
  - `cmd/mmm/install/install.go:325`
  - `cmd/mmm/update/update.go:243`
- `list` prints non-ASCII check/cross glyphs as part of its normal output (not gated on TTY): `cmd/mmm/list/list.go:224-233`.
- `remove` prints an emoji check mark directly (bypassing shared icon helpers): `cmd/mmm/remove/remove.go:171`.

Impact:

- For an automation-first CLI, non-ASCII output can cause practical issues:
  - logs can become harder to parse and grep reliably across environments
  - some terminals/fonts render these inconsistently (or as missing glyph boxes)
  - users piping output into ASCII-only systems (or enforcing ASCII policies) will be surprised

Remediation:

- Decide on a single output contract:
  - If the project wants "pretty by default", gate emoji/unicode behind `IsTerminalWriter(out)` and use ASCII equivalents when not a TTY (for example `[OK]` / `[ERR]` or `OK` / `ERR`).
  - Alternatively, add a global `--no-icons` / `--ascii` flag and document it for automation users.

Confidence: High.

---

### Medium - Ignored errors and unsafe assumptions in core I/O paths (mmm-63.68)

Evidence:

- Request creation errors ignored (nil request risk if URL is malformed):
  - `internal/httpClient/downloader.go:50`
  - `internal/minecraft/minecraft.go:45`
  - `internal/modrinth/project.go:62`
  - `internal/modrinth/version.go:110`
  - `internal/modrinth/version.go:139`
  - `internal/curseforge/project.go:27`
  - `internal/curseforge/files.go:58`
  - `internal/curseforge/files.go:115`
  - `internal/platform/curseforge.go:81`
- JSON decode errors ignored:
  - `internal/modrinth/project.go:82`
  - `internal/modrinth/version.go:129`
  - `internal/modrinth/version.go:157`
- JSON marshal errors ignored (lock/config writes):
  - `internal/config/config.go:42`
  - `internal/config/lock.go:52`
- URL parse errors ignored:
  - `internal/modrinth/version.go:104`
- JSON marshal errors ignored (request construction):
  - `internal/modrinth/version.go:101-102`
  - `internal/curseforge/files.go:114`
- `afero.Exists` errors ignored in production code:
  - `internal/config/config.go:20`
  - `internal/config/lock.go:19`
  - `internal/fileutils/fileutils.go:12`
  - `cmd/mmm/init/init.go:360`

Impact:

- Turns malformed URLs, corrupted responses, or filesystem errors into silent failures or misleading downstream behavior.

Remediation:

- Treat all of the above as hard errors and propagate them as typed failures with context.
- In tests, explicitly assert error paths for malformed URLs and bad JSON to prevent regressions.

Confidence: High.

---

### Low - Error wrapping strategy is inconsistent (`github.com/pkg/errors` vs standard library) (mmm-63.69)

Evidence:

- Some packages use `github.com/pkg/errors` (`internal/modrinth/version.go:12`, `internal/curseforge/files.go:13`) while most of the repo uses `fmt.Errorf(\"...: %w\", err)`.

Impact:

- Inconsistent behavior for wrapping/unwrapping and stack traces; makes it harder to reason about error handling and user-facing error messages.

Remediation:

- Standardize on the Go standard library error wrapping patterns unless there is a specific, documented reason to retain `pkg/errors`.

Confidence: Medium.

---

### Low - Naming and API hygiene issues (nitpicks with real costs) (mmm-63.70)

Evidence (examples):

- Exported function name has inconsistent capitalization: `internal/minecraft/minecraft.go:95` (`GetAllMineCraftVersions`).
  - Additional occurrences of the incorrect "MineCraft" casing in the repo (spreads the style defect across code/docs/tests):
    - Caller: `cmd/mmm/init/gameVersion.go:111` (`minecraft.GetAllMineCraftVersions`)
    - Tests: `internal/minecraft/minecraft_test.go:69`, `internal/minecraft/minecraft_test.go:101`, `internal/minecraft/minecraft_test.go:137`, `internal/minecraft/minecraft_test.go:145`
    - Package README: `internal/minecraft/README.md:15`
- `internal/httpClient/httpClient.go:26` uses field name `Ratelimiter` (mixed casing; prefer `RateLimiter`).
- Go package naming deviates from standard practice (mixed case package names):
  - `internal/httpClient/httpClient.go:1` (`package httpClient`)
  - `internal/globalErrors` (`package globalErrors` across files)
- Inconsistent initialism handling in core models:
  - `internal/models/mod.go:5` uses `ID`
  - `internal/models/modInstall.go:5` uses `Id`
  - `internal/models/modInstall.go:10` uses `DownloadUrl` instead of `DownloadURL`

Impact:

- These reduce clarity, make exported APIs look unprofessional, and create long-term friction (renaming later is breaking).

Remediation:

- Fix naming before wider adoption. For exported symbols, consider deprecating the old name and introducing the corrected name if backwards compatibility matters.

Confidence: Medium.

---

### Low - Duplicate helper logic across commands increases drift risk (mmm-63.71)

Evidence:

- `lockIndexFor` is re-implemented in multiple commands:
  - `cmd/mmm/remove/remove.go:240`
  - `cmd/mmm/install/install.go:343`
  - `cmd/mmm/update/update.go:497`
- `effectiveAllowedReleaseTypes` is re-implemented in multiple commands:
  - `cmd/mmm/install/install.go:336`
  - `cmd/mmm/update/update.go:490`
  - `cmd/mmm/test/test.go:364`
- `downloadClient` is duplicated across commands:
  - `cmd/mmm/add/add.go:468`
  - `cmd/mmm/install/install.go:679`
  - `cmd/mmm/update/update.go:521`
- `noopSender` is duplicated across internal and command packages:
  - `internal/modsetup/modsetup.go:287`
  - `internal/modinstall/modinstall.go:113`
  - `cmd/mmm/install/install.go:675`
  - `cmd/mmm/update/update.go:517`

Impact:

- Increases the chance of subtle behavioral drift (for example: one command fixes a bug, another does not).
- Makes security hardening (for example: filename/path validation) harder to implement consistently.

Remediation:

- Consolidate shared helpers into a small internal package (or reuse existing internal packages) and cover the shared behavior with focused unit tests.

Confidence: High.

---

### Low - Platform API clients set auth headers with `Header.Add` (prefer `Header.Set`) and inconsistent header key casing (mmm-63.72)

Evidence:

- Modrinth client sets headers via a map and `request.Header.Add(...)`: `internal/modrinth/modrinthClient.go:26-34`.
  - Uses lowercase header keys like `user-agent`, `Accept`, `Authorization`.
- CurseForge client sets headers via a map and `request.Header.Add(...)`: `internal/curseforge/curseforgeClient.go:25-34`.
  - Uses lowercase `x-api-key`.

Impact:

- `Header.Add` can create duplicate header values if a request object is accidentally reused (or wrapped multiple times), which makes failures harder to debug and can create unexpected upstream behavior.
- Inconsistent header key casing is not functionally wrong (HTTP header names are case-insensitive) but it is style noise and can complicate debugging/grepping.
- Given the explicit policy "URLs yes, header secrets no", minimizing header weirdness helps reduce the chance of accidental exposure if instrumentation ever evolves to capture headers.

Remediation:

- Use `Header.Set` for these single-valued headers and standardize header key casing (canonical `User-Agent`, `Accept`, `Authorization`, `X-API-Key`).

Confidence: High.

---

### Low - Test assertions contain copy/paste mistakes in failure messages (hygiene signal) (mmm-63.73)

Evidence:

- Tests validate the `x-api-key` header but print `Authorization` in the failure message:
  - `internal/curseforge/project_test.go:128-130`
  - `internal/curseforge/files_test.go:118-120`
  - `internal/curseforge/files_test.go:584-585`

Impact:

- Reduces trust in tests as documentation of intent and slows debugging when failures occur.
- In the worst case, if a contributor points tests at real credentials and a failure message prints the wrong header value, it increases the chance of accidental secret disclosure in CI logs (even if current tests set mock values).

Remediation:

- Fix the assertion messages and consider a lightweight review pass for copy/paste issues in tests, especially around auth and networking.

Confidence: High.

---

### Medium - `SECURITY.md` claims are not backed by repository evidence (mmm-63.74)

Evidence:

- `SECURITY.md:3` claims continuous vulnerability scanning and automatic dependency updates.
- The only visible CI workflow (`.github/workflows/ci.yml:1`) runs tests/coverage/build but does not run vulnerability scanning (no govulncheck, no SCA, no CodeQL, no dependency review job).

Impact:

- Users are told there is continuous security scanning when there is no visible mechanism enforcing it. This is a trust and governance problem.

Remediation:

- Either implement the stated scanning (and document what runs where), or rewrite `SECURITY.md` to accurately reflect current practice.

Confidence: High.

---

## Tooling Findings

### staticcheck

`staticcheck ./...` reports issues including:

- Dead assignments / unused values:
  - `cmd/mmm/add/add.go:328` (SA4006)
- Redundant formatting:
  - `cmd/mmm/init/init.go:426` and `cmd/mmm/init/init.go:440` (S1025)
  - `cmd/mmm/init/loader.go:81` (S1025)
- Unused functions/types in production and tests:
  - `cmd/mmm/init/releaseTypes.go:133` (U1000)
  - `cmd/mmm/scan/scan_test.go:27` and `cmd/mmm/scan/scan_test.go:32` (U1000)
  - `cmd/mmm/version/version_test.go:10` and `cmd/mmm/version/version_test.go:14` (U1000)
  - `internal/tui/terminal_test.go:11` and `internal/tui/terminal_test.go:13` (U1000)
- Deprecated APIs:
  - `cmd/mmm/install/install.go:575`, `cmd/mmm/list/list.go:239`, `cmd/mmm/scan/scan.go:762` use deprecated `lipgloss.Style.Copy` (SA1019)
  - `internal/perf/perf.go:37` and `internal/perf/perf.go:72` use deprecated OTel noop tracer provider APIs (SA1019)
- Error naming:
  - `internal/minecraft/minecraftErrors.go:5` and `internal/minecraft/minecraftErrors.go:6` (ST1012)
- Nil contexts in tests:
  - `internal/perf/durations_test.go:23` and multiple `internal/perf/perf_test.go` and `internal/telemetry/telemetry_test.go` call sites (SA1012)

Judgment:

- These are mostly hygiene, but the volume indicates linting is not enforced in CI. Add a lint gate or accept accumulating debt explicitly.

### gofmt / go vet

- `go vet ./...` passes.
- `gofmt -l` reports no formatting drift.

## Recommended Go tooling to automate audit concerns

This section lists Go-centric tools that can be used to continuously enforce the expectations implied by this audit (security, correctness, hygiene, and release integrity). It is intentionally practical: each item includes what it catches and how it should be automated.

### Baseline quality gates (should run on every PR)

- Formatting:
  - `gofmt` (canonical formatting; already effectively covered).
  - Optional: `goimports` (import grouping and cleanup) if you want stronger standardization than gofmt alone.
  - Automation pattern: a CI check that fails if formatting changes would be produced (do not auto-commit).
- Static analysis:
  - `go vet` (basic correctness checks; already run in this audit, not currently a CI gate).
  - `staticcheck` (strong general-purpose lints; already run in this audit, not currently a CI gate).
  - Alternative: `golangci-lint` as a wrapper to run multiple linters consistently; if used, pin the version and keep the enabled linters minimal and intentional to avoid noisy false positives.
- Tests:
  - `go test ./...` (already a CI gate).
  - `go test -race ./...` (should be a CI gate once races are fixed; today it fails and is a blocker).
  - Coverage enforcement: ensure the policy you claim is what you enforce (current `make coverage` enforces 100% coverage via `tools/coverage`).

### Security automation (should run on every PR or at least daily)

- Vulnerability scanning:
  - `govulncheck` (official Go vuln tool). Given current symbol-scan instability in this repo, run `-scan=module` and `-scan=package` in CI until symbol scan is reliable again.
  - Also gate on the Go toolchain patch level. Today the effective toolchain is `go1.25.0` and govulncheck reports fixed stdlib vulnerabilities; pin to a patched `go1.25.x` toolchain.
- Dependency change review (GitHub, not Go-specific but directly relevant):
  - `actions/dependency-review-action` to block PRs that introduce known-vulnerable dependencies (pairs well with Renovate automerge).
- Optional static security analysis:
  - `gosec` can be useful for obvious mistakes, but expect false positives and treat it as advisory unless you tune it.

### Reliability and robustness (targets specific risks in this repo)

- Fuzzing:
  - Use Go's built-in fuzzing (`go test -fuzz=...`) for parsers and boundary logic (modlist/lock parsing, URL parsing/normalization, filename validation, and any JSON decoding of upstream API responses).
  - Automate as a time-bounded CI job (nightly) to avoid slowing PRs.
- HTTP correctness and timeouts:
  - Prefer tests that assert all outbound HTTP uses timeouts (unit tests can validate constructed `http.Client` timeouts and request context deadlines).
  - Add a regression test suite that intentionally sends malformed JSON and non-200 statuses to ensure decode/status errors are propagated (this targets the "ignored decode error" findings).

### Release artifact integrity (Go ecosystem tooling that fits this repo)

- Packaging, checksums, and provenance:
  - Consider `goreleaser` to consistently build cross-platform artifacts, generate checksums, and attach release assets in a predictable way (even if you keep semantic-release for versioning).
  - Consider signing and provenance (for example, cosign/SLSA-style attestations). This is not strictly "Go tooling", but it is commonly integrated into Go release pipelines and addresses the current "zip only, no integrity metadata" gap.

### Licensing and SBOM (automate compliance and transparency)

- License inventory:
  - `go-licenses` for a repeatable dependency license report (already executed during this audit).
  - Automate generation of a `THIRD_PARTY_NOTICES.txt` (or similar) for each release and attach it to the release (or bundle it in `dist/*.zip`).
- SBOM generation (Go-centric options):
  - `cyclonedx-gomod` to generate a CycloneDX SBOM directly from `go.mod`/`go.sum`.
  - Alternative: `syft` can generate SPDX/CycloneDX SBOMs, but it is not Go-specific.

### Tool version pinning (avoid "works on my machine" tooling drift)

- Pin tool versions the same way you pin dependencies:
  - Add a `tools.go` (build-tagged) to track tool dependencies.
  - Install tools in CI using `go install <module>@<version>` so the toolchain is deterministic.
- Avoid using floating versions in CI (Go toolchain and Actions). Prefer pinned patch versions and pinned action SHAs.

## Recommended Remediation Plan (order)

1. Remove env-driven API base URL overrides and replace with test/build injection; reconsider runtime `.env` autoload scope (see Critical exfiltration finding).
2. Fix path traversal risks by validating all filenames from remote/lock/config before any filesystem operation.
3. Make the suite race-clean and add `go test -race ./...` to CI.
4. Fix release automation consistency: either enable and align semantic-release for Go, or remove the Node-era config.
5. Make docs truthful to the current Go binary and remove references to unimplemented commands from user docs.
6. Add bounded timeouts and robust retry behavior across all HTTP operations.
7. Harden downloads: validate HTTP status, verify file hashes, prevent symlink overwrites, and make update swaps atomic.
8. Add CI gates for staticcheck (or golangci-lint) and govulncheck (at least module/package scan until symbol scan is stable).
9. Add release/CI artifacts for license attribution (THIRD_PARTY_NOTICES) and consider an SBOM for each release.
10. Update telemetry documentation and minimize sensitive payload fields while keeping opt-out.

## Remediation checklist (acceptance criteria and evidence)

This section turns the highest-impact findings into a maintainer-executable checklist. Each item includes:

- the required change outcome (what must be true)
- acceptance criteria (what to run / what must pass)
- artifacts to attach to prove closure (what evidence is required)

Where practical, prefer adding `make` targets for these checks and using those targets in CI, so local and CI behavior cannot drift.

### Definition of done: "release-ready"

This project can be considered "release-ready" only when all items below are true and backed by verifiable evidence (CI logs, test outputs, and release asset listings). This is intentionally strict because the current findings include security-impacting footguns and release automation inconsistencies.

- [ ] No blockers remain in `## Findings (prioritized)` (race-clean, release pipeline consistent, API base URL exfiltration control, and path safety addressed).
- [ ] Every item in `## Remediation checklist (acceptance criteria and evidence)` is closed with the required evidence attached (or the maintainers explicitly document a risk acceptance decision and why).
- [ ] CI on `main` enforces (and is required by branch protection):
  - `make coverage`
  - Coverage enforcement gate that matches the stated policy
  - `make test-race` (or equivalent race gate)
  - lint gate (`staticcheck` or curated `golangci-lint`)
  - vulnerability scan gate (`govulncheck` at least module/package)
- [ ] Release automation is reproducible:
  - a dry-run release job (no publish) succeeds end-to-end and produces exactly the expected assets
  - release artifacts include integrity/metadata deliverables (checksums/signatures/provenance as chosen, plus `LICENSE` and third-party notices)
- [ ] Telemetry invariants are enforced by tests (URLs allowed, secret headers forbidden) and shutdown is time-bounded in the real code path.

### 1) Remove runtime API base URL overrides (credential exfiltration control)

- [ ] Outcome: production code has no runtime configuration for API base URLs (CurseForge/Modrinth base URLs are fixed, not env-driven).
- [ ] Acceptance criteria:
  - `rg -n "MODRINTH_API_URL|CURSEFORGE_API_URL" .` returns no hits outside tests/tools (define and enforce what "test-only" means).
  - All tests that previously depended on env URL overrides are migrated to an explicit injection seam (constructor param or test-only setter).
  - `make coverage` passes.
- [ ] Evidence to attach:
  - A short log showing the `rg` search output (no runtime hits).
  - A note describing the chosen test seam (API base URL injection mechanism) and where it is defined.

### 2) Fix filesystem path safety (path traversal, symlink/reparse-point safety, predictable temp names)

- [ ] Outcome: no untrusted remote metadata (filenames, URLs, IDs) can cause writes/deletes outside the intended mods folder or config directory.
- [ ] Acceptance criteria:
  - Add unit tests that prove the sanitizer rejects:
    - relative traversal (`../x.jar`, `..\\x.jar`)
    - absolute paths (`/etc/passwd`, `C:\\Windows\\system32\\x`)
    - UNC paths (`\\\\server\\share\\x`)
    - path separators in "filenames" from remote sources
  - Add unit tests that prove write paths are symlink-safe (do not follow a symlink that points outside the directory) on supported platforms, or explicitly document platform limitations and add tests for those constraints.
  - `make coverage` passes.
  - If Windows is a supported target: add tests for Windows rename/replace semantics and reparse-point behavior (at minimum: do not overwrite through symlink-like constructs).
- [ ] Evidence to attach:
  - New/updated test names proving traversal rejection and symlink safety.
  - A short explanation of the chosen safe-write strategy (how you prevent TOCTOU and symlink exploitation).

### 3) Make the suite race-clean and gate it

- [ ] Outcome: `go test -race` is clean and remains clean.
- [ ] Acceptance criteria:
  - Add a `make test-race` target that runs `go test -race ./...`.
  - `make test-race` passes locally.
  - CI runs `make test-race` and it passes on main.
- [ ] Evidence to attach:
  - CI job link or log excerpt showing the race job passed.
  - A note listing the fixed race sites (by package/file) and what synchronization change was applied.

### 4) Fix release automation consistency and artifact integrity

- [ ] Outcome: release pipeline is deterministic, consistent with the Go repo layout, and does not silently produce wrong artifacts.
- [ ] Acceptance criteria:
  - Decide and document the release authority:
    - either semantic-release is authoritative and configured for the Go repo, or it is removed/disabled for this repo.
  - If semantic-release remains:
    - `scripts/prepare.sh` must be robust (strict mode, validated inputs, portable or constrained to a documented runner OS).
    - Asset selection globs in `.releaserc.json` must not pick up stale/unrelated zips (clean `dist/` before packaging or narrow globs).
  - Add a CI "release dry run" on main (no publishing) that verifies the pipeline can run end-to-end and produce expected artifacts.
- [ ] Evidence to attach:
  - The command(s) used for the dry run and their output.
  - A list of the expected release assets (filenames) and a proof they are the only assets produced.

### 5) Bound all HTTP operations (timeouts and safe retries)

- [ ] Outcome: no HTTP call can hang indefinitely; retries are safe even when upstream uses POST.
- [ ] Acceptance criteria:
  - Add explicit timeouts (client and/or per-request context) and prove via tests that a stuck server fails within a bounded window.
  - Ensure retries:
    - retry network errors as well as selected status codes (as intended)
    - do not reuse a consumed request body (use a replayable body strategy)
  - `make coverage` passes.
- [ ] Evidence to attach:
  - A test that simulates a non-responding handler and asserts bounded failure time.
  - A test that retries a request with a body without data corruption.

### 6) Stop ignoring decode/status/request-build errors (correctness and observability)

- [ ] Outcome: invalid responses and malformed URLs fail fast with typed, user-relevant errors (no silent empty structs).
- [ ] Acceptance criteria:
  - For each API client:
    - `http.NewRequestWithContext` errors are handled (no discarded error values).
    - Non-2xx statuses are treated as errors before decode.
    - JSON decode errors are returned (not ignored).
  - Add tests for each of the above in Modrinth and CurseForge clients.
  - `make coverage` passes.
- [ ] Evidence to attach:
  - Links to the specific tests covering "bad URL", "non-200", and "bad JSON" per client.

### 7) Telemetry safety gates (URLs allowed, secret headers forbidden)

- [ ] Outcome: telemetry and perf export can never include API keys or other secret header values, even under error conditions.
- [ ] Acceptance criteria:
  - Add a test that sets a known secret value (for example `MODRINTH_API_KEY=test-secret`) and asserts:
    - recorded telemetry payloads contain request URL fields only as allowed
    - no payload contains `test-secret` anywhere (string search over JSON payload)
  - Ensure shutdown is time-bounded even when passed a non-nil context without deadline (fix the current "timeout only when ctx==nil" behavior).
  - `make coverage` passes.
- [ ] Evidence to attach:
  - The redaction/leak-prevention tests and a short description of the redaction strategy.

### 8) Toolchain and CI hardening (make it impossible to regress)

- [ ] Outcome: CI prevents the regressions identified by this audit.
- [ ] Acceptance criteria:
  - Add CI jobs (prefer via `make` targets):
    - formatting check (gofmt/goimports policy)
    - lint (`staticcheck` or `golangci-lint`)
    - vulnerability scan (`govulncheck` at least module/package)
    - dependency review (GitHub) if you allow Renovate automerge
  - Pin:
    - GitHub Actions by SHA
    - Go toolchain to a patched `go1.25.x` (and ensure the version used in CI matches the pinned version)
  - CI permissions are least-privilege for PRs (avoid broad write permissions on PR runs).
- [ ] Evidence to attach:
  - Workflow diffs and a CI run showing all new gates passing.
  - A note documenting the chosen Go patch version and the rationale (must address the govulncheck-reported stdlib fixes).

### 9) Licensing and SBOM artifacts for releases

- [ ] Outcome: distributed artifacts have transparent licensing and (optionally) SBOM metadata.
- [ ] Acceptance criteria:
  - Add a reproducible artifact generation step:
    - `THIRD_PARTY_NOTICES.txt` (from `go-licenses report`, plus any manual notices required)
    - optional: SBOM (CycloneDX or SPDX) for each release
  - Ensure release artifacts include `LICENSE` and third-party notices (either bundled in zips or attached as separate assets with clear naming).
- [ ] Evidence to attach:
  - A sample release asset listing showing notices/SBOM present.
  - The exact command(s) used to generate the notices/SBOM and how they are versioned.

## Appendix - Package-by-package judgement (spotlight)

This is not a complete re-listing of all findings above; it is a quick "where the bodies are buried" index so reviewers can jump straight to the highest-risk packages.

- `main.go`: runtime `.env` autoload (`main.go:12`), perf always enabled (`main.go:115`), telemetry shutdown not time-bounded (`main.go:155` plus `internal/telemetry/telemetry.go:381-400`).
- `cmd/mmm/root.go`: help/version surface has drift and confusing overlaps (cobra default `completion`, global `-v/--version` plus `version` command), `--quiet` help text overpromises ("Suppress all output") vs actual behavior.
- `cmd/mmm/add/*`: large command surface with duplicated helpers and at least one staticcheck smell (`cmd/mmm/add/add.go:328` assignments unused).
- `cmd/mmm/init/*`: uses `http.DefaultClient` (`cmd/mmm/init/init.go:66-69`) and has user-visible help text issues (see existing init findings).
- `cmd/mmm/install/*`: downloads jars to paths derived from untrusted metadata (remote filenames) and emits non-ASCII output even when not a TTY (see existing download/path traversal and output-contract findings).
- `cmd/mmm/list/*`: violates "purely informational/no prompts" spec by running TUI flows (`cmd/mmm/list/list.go:112-133`) and uses non-ASCII glyphs in non-TTY output (`cmd/mmm/list/list.go:224-233`).
- `cmd/mmm/remove/*`: can remove paths derived from lock metadata (`cmd/mmm/remove/remove.go:151-171`) and prints emoji in standard output (`cmd/mmm/remove/remove.go:171`).
- `cmd/mmm/scan/*`: prints emoji via `tui.SuccessIcon`/`tui.ErrorIcon` even when `colorize=false` (`cmd/mmm/scan/scan.go:764-804`, `internal/tui/icons.go:3-17`).
- `cmd/mmm/test/*`: concurrency-heavy; the test suite is not race-clean (`go test -race` fails in this package), and production goroutines call `i18n.T(...)` (see earlier concurrency finding).
- `cmd/mmm/update/*`: update swap is not atomic and uses predictable temp/backup names (`cmd/mmm/update/update.go:389-456`), compounding the download integrity and symlink/TOCTOU issues.

- `internal/config/*`: atomic write uses predictable sibling names (`internal/config/atomic_write.go:11-57`) and `afero.Exists` errors are ignored (`internal/config/config.go:20-23`, `internal/config/lock.go:18-26`).
- `internal/httpClient/*`: no timeouts (`internal/httpClient/httpClient.go:124-131`), partial retry semantics and blocking sleep (`internal/httpClient/httpClient.go:51-105`), downloader writes without status checks and is not symlink-safe (`internal/httpClient/downloader.go:50-83`).
- `internal/modrinth/*`: base URL override via env (must be removed from production) and JSON decode errors ignored (`internal/modrinth/project.go:81-83`, `internal/modrinth/version.go:128-130`, `internal/modrinth/version.go:156-158`).
- `internal/curseforge/*`: base URL override via env (must be removed from production) and request build errors ignored (`internal/curseforge/project.go:26-31`, `internal/curseforge/files.go:57-63`, `internal/curseforge/files.go:112-119`).
- `internal/platform/*`: default clients disable rate limiting (`internal/platform/platform.go:35-44`) and inherit the API client weaknesses above.
- `internal/i18n/*`: global lazy init without synchronization (race risk), panics on setup and arg count, and common `LANG` formats are ignored (`internal/i18n/i18n.go:111-116`, `internal/i18n/i18n.go:156-159`).
- `internal/telemetry/*`: opt-out telemetry includes full performance span trees and error strings; shutdown timeout is effectively dead in production (`internal/telemetry/telemetry.go:341-404`).
- `internal/tui/*`: emoji icons are unconditional (`internal/tui/icons.go:3-17`); terminal detection seams exist but are not used to gate ASCII output in commands.
- `internal/minecraft/*`: caches global manifest pointer, collapses distinct failure modes into sentinel errors, ignores request build errors (`internal/minecraft/minecraft.go:45-49`).

## Appendix - Environment variables inventory (code-derived)

Direct lookups observed in code:

- `MODRINTH_API_KEY` (auth; runtime override for embedded key): `internal/environment/environment.go:13-20`
- `CURSEFORGE_API_KEY` (auth; runtime override for embedded key): `internal/environment/environment.go:22-29`
- `POSTHOG_API_KEY` (telemetry; runtime override for embedded key): `internal/environment/environment.go:31-38`
- `MMM_DISABLE_TELEMETRY` (opt-out): `internal/telemetry/telemetry.go:588`
- `MACHINE_ID` (distinct id override): `internal/telemetry/telemetry.go:574`
- `LANG` (locale selection): `internal/i18n/i18n.go:112-116`
- `MMM_TEST` (test-mode i18n behavior; risky if set at runtime): `internal/i18n/i18n.go:81-85`
- `MODRINTH_API_URL` (base URL override; must be removed from production): `internal/modrinth/modrinthClient.go:40`
- `CURSEFORGE_API_URL` (base URL override; must be removed from production): `internal/curseforge/curseforgeClient.go:38`

Judgment:

- Key-related vars (`*_API_KEY`) and telemetry vars are expected and align with your desired runtime `.env` behavior.
- Base URL vars (`*_API_URL`) are a security footgun (credential exfiltration), and per maintainer decision must be deleted or made test-only via explicit injection.

## Resolved maintainer decisions (applied)

1. Runtime `.env` loading scope: load `.env` from CWD (global config will be added later).
2. "No args starts TUI" contract: current version should print help; no-args TUI is a known gap while pushing for Node parity and will be added after parity is complete.
3. Retry semantics: retries must remain enabled even for APIs using POST; implement safe request-body replay semantics.
4. Telemetry may include request URLs, but must not include API keys from headers (or other secret header values).
5. Perf artifacts (`mmm-perf.json`) are not considered sensitive.

## Audit coverage closure (deep audits completed here)

This audit performed deep review of all in-repo repository setup surfaces (not only Go code):

- GitHub automation and community health files (in-repo): `.github/workflows/ci.yml`, `.github/renovate.json`, `.github/stale.yml`, `.github/ISSUE_TEMPLATE/*`, `.github/instructions/*`, `SECURITY.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `FUNDING.yml`, `LICENSE`, `README.md`.
- Dependency supply chain:
  - Vulnerability posture via govulncheck (see Security findings).
  - License inventory via `go-licenses report ./...` (see Licensing finding above).
  - Renovate preset behavior validated by inspecting the referenced external preset (see CI/supply chain findings).
- Beads process health: validated via `bd` CLI only (per project policy), not by reading `.beads/` directly.
- Release automation and artifact integrity (in-repo): `.releaserc.json`, `scripts/*`, `Makefile`, and `dist/` artifact naming/globs.

## Evidence not present in-repo (requires maintainer confirmation)

Items that materially affect real-world security/release readiness but cannot be audited from this checkout because they live in GitHub/org settings or external services:

- Branch protection rules and required checks (required reviews, required status checks, signed commits, force-push rules).
- Which GitHub Apps are installed/configured (Renovate app settings, stale bot/probot, CodeQL, secret scanning/push protection, dependency review, etc.).
- Secret governance (where tokens live, rotation policy, who can access them, whether PRs from branches can access secrets).
- Release provenance/signing (checksums, signatures, attestations, and whether they are verified by consumers).
- Telemetry/compliance posture from a legal standpoint (GDPR/PII policies, retention, DPA, user-facing consent text).

## Audit limitations (honesty)

- This report is evidence-backed but cannot literally guarantee every single issue in the entire codebase has been enumerated; it reflects best-effort repo-wide review using local execution, static analysis, and targeted manual inspection of high-risk boundaries (networking, filesystem, config, i18n, CI/release).

## Auditor self-check (persona requirement)

This section is a consistency and reasoning check for the top findings. It does not introduce new requirements; it exists to ensure the report is internally coherent and evidence-backed.

### Top 3 findings: chain of reasoning

1) Runtime API base URL overrides (credential exfiltration footgun)

- Evidence: base URLs can be overridden via env vars: `internal/modrinth/modrinthClient.go:40` and `internal/curseforge/curseforgeClient.go:38`.
- Risk chain:
  - runtime `.env` loading from CWD is enabled (maintainer decision)
  - base URL env override is therefore easy to influence (malicious `.env`, accidental config)
  - auth headers are attached to requests
  - result: API keys can be sent to attacker-controlled endpoints
- Why it is prioritized: it is a direct secret exfiltration path that does not depend on upstream compromise.
- Confidence: High (direct code evidence, clear attack path).

2) Filesystem path safety (path traversal and write/delete boundaries)

- Evidence: commands derive filesystem paths from untrusted/remote metadata and lock contents (see the file safety findings in `## Findings (prioritized)` and the package spotlight references for install/update/remove).
- Risk chain:
  - remote metadata and lock files are treated as authoritative inputs
  - filenames/paths can contain traversal or absolute path forms
  - writes/deletes can occur outside the intended directory if not normalized/validated
  - result: data loss or unintended overwrite is possible
- Why it is prioritized: it affects integrity of user systems and is easy to exploit if any untrusted metadata is accepted.
- Confidence: Medium-High (exact exploitability depends on current filename/URL handling per command, but the boundary is clearly not consistently defended).

3) Release automation consistency (risk of shipping wrong artifacts)

- Evidence: semantic-release and scripts appear misaligned with the Go repo layout and rely on brittle scripts:
  - `.releaserc.json:42` references `./src/version.ts` which does not exist here
  - `scripts/prepare.sh:6-13` rewrites tracked files, uses `sed -i` portability-sensitive behavior, and uses `$HELP_URL` without validation
  - `scripts/binaries.sh:5-7` zips binaries without notices/integrity metadata
- Risk chain:
  - inconsistent or non-functional release pipeline means releases can be non-reproducible
  - brittle scripts can silently generate wrong output (placeholders, missing URLs, wrong assets)
  - broad artifact globs can upload stale or unintended zips
  - result: end users may receive incorrect binaries and maintainers cannot reliably reproduce them
- Why it is prioritized: it undermines trust in releases and makes security fixes harder to ship reliably.
- Confidence: High (direct configuration/script evidence).

### Internal consistency validation

- Maintainer clarifications were applied:
  - embedded API keys are intended (still requires preventing URL override exfiltration)
  - runtime `.env` from CWD is intended (increases importance of hardening env parsing and removing URL overrides)
  - telemetry is opt-out and may include URLs but must not include secret headers (report recommends tests to enforce this invariant)
  - no-args behavior is help today; no-args TUI is a known future goal (docs must reflect current behavior)
  - retries must remain enabled even for POST (report recommends safe body replay for retries)
- Renovate preset ownership was updated: team-owned upstream changes are treated as governance/auditability considerations, not an external supply chain risk.

### Major conclusions and confidence

- "Not release-ready" verdict: High confidence (multiple independent blockers with direct evidence).
- Highest-risk security boundary issues (URL override + path safety + unbounded HTTP/telemetry shutdown): Medium-High confidence (direct code evidence; some real-world exploitability depends on runtime environment and usage patterns, but the footguns are present).
