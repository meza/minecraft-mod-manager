# `init`

Initialises your configuration file. This will create a `modlist.json` file in the current folder by default.

You usually run `init` once when you're setting up a modpack, or when you want to start fresh.

Example:

`mmm init -l fabric`

If you run `mmm init` in a terminal, it opens a short TUI so you can pick the loader, game version, release types, and mods folder (it seeds any flags you already set). In scripts, pass the flags you need and `--quiet` to skip prompts.

If you leave the default `--game-version=latest`, the command tries to look up the latest Minecraft release.
If that lookup fails (for example, you're offline), run the command again with `-g/--game-version`.
Make sure the mods folder you point at already exists; `init` stops if the path is missing or is a file.

## Flags

| Short | Long              | Meaning                                | Allowed values                                      | Example                              |
|------:|-------------------|----------------------------------------|-----------------------------------------------------|--------------------------------------|
|  `-l` | `--loader`        | Mod loader to use                      | A valid loader from the list of [loaders](#loaders) | `mmm init -l fabric`                 |
|  `-g` | `--game-version`  | Minecraft version to target            | A Minecraft version, or `latest`                    | `mmm init -l fabric -g 1.21.1`       |
|  `-r` | `--release-types` | Mod release types you allow to install | Comma-separated list of: `alpha`, `beta`, `release` | `mmm init -l fabric -r release,beta` |
|  `-m` | `--mods-folder`   | Folder to download mods into           | An absolute or relative path to your minecraft      | `mmm init -l fabric -m ./mods`       |

### Loaders

Loaders are the mod loader systems that help you run mods.

Both Modrinth and Curseforge support a different set of loaders. `mmm` supports all the loaders that are supported by these platforms.

> If you want to see a loader added, please open an issue on GitHub.

#### Supported by Curseforge AND Modrinth BOTH:

`fabric`, `forge`, `quilt`, `liteloader`, `neoforge`

#### Supported by Curseforge ONLY:

`cauldron`

#### Supported by Modrinth ONLY:

`bukkit`, `bungeecord`, `datapack`, `folia`, `modloader`, `paper`, `purpur`, `rift`, `spigot`, `sponge`, `velocity`, `waterfall`
