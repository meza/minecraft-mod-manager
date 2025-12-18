# `scan`

Use `mmm scan` to find jar files in your mods folder that are not managed by your `modlist.json` / `modlist-lock.json`.

It is helpful when you copied mods into the folder manually, upgraded launchers, or migrated an instance.

## Example

```bash
mmm scan --prefer modrinth --add
```

## What it does

- Looks for `.jar` files in your mods folder.
- Ignores files matched by `.mmmignore` and anything ending in `.disabled`.
- Skips files that are already managed by the lock file.
- Tries to identify each file by hash on your preferred platform first, and only falls back to the other platform if there are no hits.
- Prints recognized vs unknown vs unsure files.
- With `--add` (or when you confirm the prompt), updates `modlist.json` and `modlist-lock.json` with the discovered mods (unless any file is "unsure").

## Flags

| Flag | Meaning | Allowed values | Example |
| --- | --- | --- | --- |
| `-p, --prefer` | Which platform to check first (default: `modrinth`) | `modrinth`, `curseforge` | `--prefer curseforge` |
| `-a, --add` | Persist results without prompting | `true/false` | `--add` |

## If something goes wrong

If a file cannot be looked up due to a platform error, it is reported as "unsure" and nothing is written to your config/lock, even with `--add`.

If a file is listed as "unknown", it means the file hash did not match anything on either platform.

If you run with `--quiet`, the command does not prompt and does not print normal output.
