# Existing HTTP VCR helper

Archived reference. This page preserves the helper documentation at retirement;
its implementation-status statements are historical. Use the [testing guide](../README.md)
for the active E2E contract.

This page is a reference for the existing `testutil/vcr` helper. It has no consumers outside its own tests, and the repository has no checked-in cassettes. Ordinary product tests do not currently obtain network isolation from this helper.

The agreed E2E approach is [scenario-owned Go HTTP fixtures](../http-fixtures.md), whose endpoint wiring is not implemented yet. Do not use this helper as the foundation for new E2E scenarios. It operates inside the importing process and cannot intercept requests made by a separately launched MMM executable.

The remaining sections describe the helper's recording and replay behavior, not the selected E2E design. Cassettes capture HTTP responses for later replay.

## Quick start for in-process Go tests

An in-process caller would declare the cassette in the test itself. This example is illustrative; the named cassette is not supplied by the repository:

```go
func TestScanScenario(t *testing.T) {
	vcr.LoadCassette(t, filepath.Join("testdata", "vcr", "scan-basic.yaml"))
	// Test body...
}
```

Run the ordinary Go tests in replay mode:

```bash
make test
```

This workflow applies only to in-process Go tests. HTTP-dependent terminal scenarios instead require the [planned fixture boundary](../http-fixtures.md). The current foundation smoke scenario is local-only.

Record all cassettes (one per test that calls `LoadCassette`):

```bash
make vcr-record
```

## How VCR activates

VCR activates only when your test calls `vcr.LoadCassette`.

- The cassette path is explicit and provided by the test code.
- Use `filepath.Join` to keep paths cross-platform.
- If you pass a filename ending in `.yaml` or `.yml`, VCR strips the extension for the recorder base.
- Importing the `vcr` package installs a test-only transport that rejects non-local live HTTP calls. `LoadCassette` swaps in the recorder for the test and restores the live-call blocker afterward.

These changes affect the process-wide default transport, not every possible HTTP client. Clients holding another transport and separately launched processes are unaffected. A caller must load the cassette before constructing clients that capture the default transport. Only one cassette can be active at a time.

## Matching rules

The current matcher comes from go-vcr's default behavior:

- Requests match by HTTP method and full URL (including query string).
- Each recorded interaction is replayed once. If the same request happens twice, the cassette must contain two interactions.

What this means in practice:

- Keep request URLs stable. If the URL includes timestamps or random query values, VCR will miss.
- If you need different variants, record separate interactions by making those requests during recording.

go-vcr supports custom matching, but this helper does not expose it. Its default matching does not verify request bodies or headers.

## Recording vs replay

- Replay is the default behavior when VCR is enabled.
- A missing cassette produces an error on the first request, not when `LoadCassette` is called. The application may handle that error; it does not independently fail the test. If no request occurs, the missing cassette goes unnoticed.
- Recording only happens when `MMM_RECORD_HTTP=1` (use `make vcr-record`).
  The make target forces `-count=1` so recordings are not skipped by the Go test cache.

To refresh a cassette, delete the file and run the record command again.

## Customizing responses

You can edit the cassette YAML directly to simulate specific HTTP responses.
Each cassette stores a list of `interactions` with a `request` and `response`.

Example: turn a 200 into a 500 for a single interaction:

```yaml
interactions:
  - request:
      method: GET
      url: https://example.invalid/api/mods
    response:
      code: 500
      status: 500 Internal Server Error
      body: '{"error":"server down"}'
```

Tips:

- Keep the request method and URL aligned with what your code issues.
- If you need to add or remove interactions, re-record the cassette instead of hand-editing multiple entries.

## Simulating file downloads

An in-process test could route downloads through the same VCR transport. MMM handles downloads through
`httpclient.DownloadFile`, a helper that fetches a URL to disk and optionally
streams progress to a Bubble Tea program.
If your test exercises install or update flows, it is usually going through this helper.
Use this section when you need deterministic download tests or want to simulate failures.

### What `DownloadFile` expects

`DownloadFile` enforces two constraints that affect cassettes:

- HTTPS only.
- Host allowlist: `cdn.modrinth.com`, `edge.forgecdn.net`, `media.forgecdn.net`.

If the URL in your cassette is not HTTPS or not on the allowlist, the download will fail before VCR gets a chance to replay it.

Progress rendering depends on `Content-Length`. If it is missing, progress stays unknown.

### Recording a real download

Record with a small, stable file on a trusted host. The cassette can live next to the test:

```go
func TestDownloadJar(t *testing.T) {
	vcr.LoadCassette(t, filepath.Join("testdata", "vcr", "download-jar.yaml"))

	client := httpclient.NewRLClient(httpclient.DefaultLimiter())
	dest := filepath.Join(t.TempDir(), "mod.jar")
	err := httpclient.DownloadFile(context.Background(),
		"https://cdn.modrinth.com/data/example.jar",
		dest,
		client,
		nil,
	)
	require.NoError(t, err)
}
```

Run `make vcr-record` to capture the cassette once.

### Editing a download response

You can edit the cassette YAML directly to change the download payload or force errors.
Each interaction stores a `request` and `response` block.

Example: a small payload with a `Content-Length` header for progress:

```yaml
interactions:
  - request:
      method: GET
      url: https://cdn.modrinth.com/data/example.jar
    response:
      code: 200
      status: 200 OK
      headers:
        Content-Length:
          - "8"
      body: "testdata"
```

Example: simulate a 404 download:

```yaml
response:
  code: 404
  status: 404 Not Found
  body: "missing"
```

If you need multiple download attempts, record multiple interactions. VCR replays each interaction once in order.

### Simulating download failures without a cassette

VCR replays responses immediately and does not simulate latency, timeouts, or mid-stream errors.
When you need those behaviors, inject a custom `Doer` and skip `LoadCassette`.

Example: force a timeout error for `DownloadFile`:

```go
type doerFunc func(*http.Request) (*http.Response, error)

func (fn doerFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

timeoutDoer := doerFunc(func(*http.Request) (*http.Response, error) {
	return nil, context.DeadlineExceeded
})

err := httpclient.DownloadFile(context.Background(),
	"https://cdn.modrinth.com/data/example.jar",
	filepath.Join(t.TempDir(), "mod.jar"),
	timeoutDoer,
	nil,
)
require.ErrorIs(t, err, context.DeadlineExceeded)
```

Use this approach for:

- timeouts
- connection errors
- large payloads that would bloat a cassette

## Simulating timeouts and network errors

VCR replays responses immediately (it does not delay based on recorded duration), so it is not the right tool for latency or timeout behavior.
For timeouts or connection errors, inject a custom transport instead of loading a cassette.

Example: force a timeout error for all HTTP calls in a test:

```go
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

originalTransport := http.DefaultTransport
http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
	return nil, context.DeadlineExceeded
})
t.Cleanup(func() {
	http.DefaultTransport = originalTransport
})
```

Set the default transport before you construct any `httpclient.NewRLClient` so the client picks up the override.
If you want a retry or timeout to trigger, ensure the request context uses the appropriate deadline (for example, using the helpers in `internal/httpclient`).

## Configuration

| Variable | Meaning | Allowed values | Example |
| --- | --- | --- | --- |
| `MMM_RECORD_HTTP` | Enable recording instead of replay | `1`, `true`, `yes`, `y`, `on` | `MMM_RECORD_HTTP=1` |

## Redaction rules

The recorder removes API keys and tokens from recorded artifacts.
Currently redacted headers:

- `Authorization`
- `X-Api-Key`

If you add a new secret-bearing header, update the redaction list in `testutil/vcr`.

## Related docs

- `docs/testing/terminal-harness.md` for tui-test terminal E2E usage.
- `docs/testing/bdd.md` for the BDD driver and scenario layout.
