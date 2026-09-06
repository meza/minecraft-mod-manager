# internal/httpclient

This package is where we keep the shared HTTP primitives for the CLI:

- a small `Doer` interface so networking can be mocked
- a rate-limited, retrying client wrapper
- a file downloader that can report progress to a Bubble Tea program

If you are adding a command that talks to an external API, start here.

## Public API

### Doer and rate-limited client

- `type Doer interface { Do(*http.Request) (*http.Response, error) }`
- `type RLHTTPClient struct { ... }` (implements `Doer`)
- `NewRLClient(limiter *rate.Limiter) *RLHTTPClient`
- `RetryConfig` and `NoRetries() *RetryConfig`
- `DefaultLimiter()` for the default rate-limit policy

`RLHTTPClient.Do`:

- waits on the provided rate limiter before each request
- retries transient network errors and HTTP 5xx/429 responses up to `MaxRetries` with backoff
- drains and closes the response body between retries to avoid leaking connections
- requires `request.GetBody` when retrying requests with bodies so each attempt gets a fresh body
- honors `Retry-After` and `X-Ratelimit-*` headers to delay subsequent requests when rate limits are low
- wraps timeout errors with an i18n-backed message instructing retry/connection checks

### File download with progress

- `DownloadFile(ctx context.Context, url string, filepath string, client Doer, program Sender, filesystem ...afero.Fs) error`

`DownloadFile` validates that download URLs use https and point at trusted hosts (`cdn.modrinth.com`, `edge.forgecdn.net`, `media.forgecdn.net`), writes the response body to `filepath`, and sends progress updates to `program.Send(...)`. It requires a successful 2xx response and returns an error for non-2xx statuses. It is used by interactive commands that want to surface download progress in the TUI.

See the terminal contract and implementation guidance:

- [progress bars](../../docs/contributing/interactions/interaction-guidelines.md#progress-bars)
- [execution modes and operator intent](../../docs/intent.md#execution-modes-and-operator-intent)
- [active display and permanent transcript](../../docs/intent.md#active-display-and-permanent-transcript)
- [guide to working with the terminal](../../docs/contributing/guide-to-working-with-the-terminal.md)

### Timeout policy

Per-request timeouts are applied via helpers in this package:

- `WithMetadataTimeout(ctx)` uses a 15s deadline for API/metadata calls.
- `WithDownloadTimeout(ctx)` uses a 5m deadline for downloads.

Call sites should wrap each request with the appropriate helper instead of relying on a global `http.Client.Timeout`. These defaults are the CLI baseline and can be adjusted in code if requirements change.

## Tests

See the root [verification guidance](../../CONTRIBUTING.md#verification) for required checks and snapshot update instructions.
