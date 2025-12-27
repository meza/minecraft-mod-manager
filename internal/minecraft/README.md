# internal/minecraft

This package talks to Mojang's version manifest to answer a few questions the CLI needs during `init` and other flows:

- what is the latest stable Minecraft version?
- is a user-provided version string valid?
- what versions exist (for UI selection and validation)

The manifest is cached in-memory to keep repeated calls fast. The cache expires after 15 minutes.

## Public API

- `GetLatestVersion(ctx context.Context, client httpclient.Doer) (string, error)`
- `IsValidVersion(ctx context.Context, version string, client httpclient.Doer) (bool, error)`
- `GetAllMineCraftVersions(ctx context.Context, client httpclient.Doer) []string`
- `ClearManifestCache()` (test helper)

## Offline / failure behavior

`IsValidVersion` requires a manifest lookup to validate a version:

- if the manifest request fails, it returns `false` with an error
- if the version string is empty, it returns `false`

This behavior matters for UX: "cannot validate" is not the same as "invalid".

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
