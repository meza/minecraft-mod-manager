# internal/minecraft

This package talks to Mojang's version manifest to answer a few questions the CLI needs during `init` and other flows:

- what is the latest stable Minecraft version?
- is a user-provided version string valid?
- what versions exist (for UI selection and validation)

The manifest is cached in-memory to keep repeated calls fast.

## Public API

- `GetLatestVersion(client httpclient.Doer) (string, error)`
- `IsValidVersion(version string, client httpclient.Doer) bool`
- `GetAllMineCraftVersions(client httpclient.Doer) []string`
- `ClearManifestCache()` (test helper)

## Offline / failure behavior

`IsValidVersion` is deliberately permissive when the manifest cannot be fetched:

- if the manifest request fails, it returns `true` so the user can still try to proceed offline
- if the version string is empty, it returns `false`

This behavior matters for UX: "cannot validate" is not the same as "invalid".

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.
