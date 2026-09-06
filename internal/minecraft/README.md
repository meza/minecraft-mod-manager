# internal/minecraft

This package talks to Mojang's version manifest to answer a few questions the CLI needs during `init` and other flows:

- what is the latest stable Minecraft version?
- is a user-provided version string valid?
- what versions exist (for UI selection and validation)

The manifest is cached in-memory for the application lifecycle to keep repeated calls fast and avoid repeated downloads.

## Public API

- `GetLatestVersion(ctx context.Context, client httpclient.Doer) (string, error)`
- `IsValidVersion(ctx context.Context, version string, client httpclient.Doer) (bool, error)`
- `GetAllMinecraftVersions(ctx context.Context, client httpclient.Doer) []string`
- `NextPatchDown(ctx context.Context, version string, client httpclient.Doer) (string, bool, error)`
- `ClearManifestCache()` (test helper)

## Offline / failure behavior

`IsValidVersion` requires a manifest lookup to validate a version:

- if the manifest request fails, it returns `false` with an error
- if the version string is empty, it returns `false`

This behavior matters for UX: "cannot validate" is not the same as "invalid".

`NextPatchDown` requires a manifest lookup to validate release versions. Mojang releases use a
`1.20` then `1.20.1` pattern without a `.0`, so `1.20.1` falling back to `1.20` is expected:

- if the version is not a manifest release, it returns an error
- if no lower patch exists for the same series key, it returns the original version with `false`

## Tests

See the root [verification guidance](../../CONTRIBUTING.md#verification) for required checks and snapshot update instructions.
