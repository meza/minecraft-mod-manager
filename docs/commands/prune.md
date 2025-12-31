# prune

Delete unmanaged `.jar` files from your mods folder. Use this to clean up files that are not present in `modlist-lock.json`.

Quick example:

```
mmm prune
```

The command only considers `.jar` files. It skips files matched by `.mmmignore` (patterns are relative to the mods folder) and always ignores anything ending in `.disabled`.

When running interactively, prune lists unmanaged files and asks for confirmation with a prompt like `? Do you want to delete these files?`. If you pass `--non-interactive`, prune prints a warning, lists unmanaged files, and skips the prompt, assuming no deletion unless you also pass `--force`. If prompts are disabled because stdin or stdout are not TTYs and you did not pass `--force`, prune prints a warning and exits without deleting anything. `--quiet --force` deletes unmanaged files without any output.

## Flags

| Flag | Meaning | Allowed values | Example |
| --- | --- | --- | --- |
| `-f, --force` | Delete unmanaged files without prompting | true/false | `mmm prune --force` |
