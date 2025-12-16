# internal/httpClient

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

`RLHTTPClient.Do`:

- waits on the provided rate limiter before each request
- retries server errors (HTTP 5xx) up to `MaxRetries`
- drains and closes the response body between retries to avoid leaking connections

### File download with progress

- `DownloadFile(url string, filepath string, client Doer, program Sender, filesystem ...afero.Fs) error`

`DownloadFile` writes the response body to `filepath` and sends progress updates to `program.Send(...)`. It is used by interactive commands that want to surface download progress in the TUI.

## Tests

Run from the repo root:

```bash
make test
```

