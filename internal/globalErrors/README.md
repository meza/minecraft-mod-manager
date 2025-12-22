# internal/globalErrors

This package holds error types that are shared across platform integrations.

The goal is consistent UX: callers can detect "not found" and "API failure" in a platform-agnostic way using `errors.Is`/`errors.As`, then decide what to show the user.

## Public API

### Project errors

- `ProjectNotFoundError` (project ID does not exist on a platform)
- `ProjectApiError` (request failed due to network/API/decoding issues)
- `ProjectApiErrorWrap(err error, projectId string, platform models.Platform) error`

`ProjectApiError` implements `Unwrap` so the underlying error is still available.

## Quick start

```go
project, err := curseforge.GetProject("1234", client)
if errors.Is(err, &globalErrors.ProjectNotFoundError{ProjectID: "1234", Platform: models.CURSEFORGE}) {
	// show a "mod not found" message
}
```

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
