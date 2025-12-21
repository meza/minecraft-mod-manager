# Support

Use this page to decide where to ask for help, report a bug, or share feedback.
If you are not sure where to start, open a discussion first.

Example command to include in a report:

```bash
mmm --debug --help
```

## Questions and ideas

Start a discussion if you want help using the tool, are not sure a behavior is a bug, or want to propose a change.

- https://github.com/meza/minecraft-mod-manager/discussions

## Bug reports

Open an issue when you can describe a reproducible bug.

Include:

- What you expected to happen
- What actually happened
- Your operating system and version
- The command you ran (copy and paste)
- Any output with `--debug` enabled

Issues:

- https://github.com/meza/minecraft-mod-manager/issues

## Security issues

Follow the reporting instructions in `SECURITY.md`.

## Useful flags

| flag                  | meaning                                | allowed values | example                            |
|-----------------------|----------------------------------------|----------------|------------------------------------|
| `-c, --config <file>` | Path to `modlist.json`                 | file path      | `mmm --config ./modlist.json list` |
| `-q, --quiet`         | Suppress prompts and normal log output | none           | `mmm --quiet list`                 |
| `-d, --debug`         | Print additional debug messages        | none           | `mmm --debug list`                 |
