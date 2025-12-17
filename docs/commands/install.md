# install

Make sure every mod in your `modlist.json` is downloaded and ready to use.

Quick start:

```bash
mmm install
```

## Usage

`mmm install` or `mmm i`

Run this when you want to make sure the mods you already added are actually present on disk.

You typically use it:

- after you add or remove mods
- after you pull someone else's modpack changes
- before you start a server or launch Minecraft

## How it works

`install` works off `modlist-lock.json`, which contains the exact file details for each mod.

If `modlist-lock.json` does not exist yet, `install` downloads the latest compatible version of each mod unless you pinned a specific version with `mmm add --version` (see `add.md#installing-specific-versions`).

This is a limitation of the Minecraft modding ecosystem and the lack of enforced versioning.

Note: If you are in charge of Modrinth or CurseForge, please mandate the use of semver.

If `modlist-lock.json` exists, `install` always downloads the exact same versions listed inside it.

## Sharing

If you want other people (or CI) to install the same mod versions, commit and share both `modlist.json` and `modlist-lock.json`.
