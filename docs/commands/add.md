# add

Add a mod to your configuration and download its jar into your mods folder.

Quick start:

```bash
mmm add modrinth AANobbMI
```

## Usage

`mmm add <platform> <id>`

This adds a given mod to the configuration file and downloads the relevant mod file to the configured mods folder.

If the mod cannot be found, the platform is invalid, or no compatible file exists, MMM prompts you to adjust the platform and/or project ID. Use `--quiet` to skip prompts and fail fast.

## Flags

| Flag                           | Meaning                                                             | Allowed values                                 | Example                    |
|--------------------------------|---------------------------------------------------------------------|------------------------------------------------|----------------------------|
| `-f, --allow-version-fallback` | Try lower Minecraft patch versions when no compatible file is found | true/false                                     | `--allow-version-fallback` |
| `-v, --version`                | Install a specific mod version                                      | Modrinth version number or CurseForge filename | `--version 1.3.1`          |

## Installing Specific Versions

If you want to install a specific version of a mod, use the `--version` flag:

`mmm add modrinth FOIvwGKz --version 1.3.1`

The version you specify must exist for your configured Minecraft version.

Modrinth and CurseForge handle versions differently:

### Modrinth

On Modrinth, you pass the version number as it is listed on the website.

![](/doc/images/versions-modrinth.png)

In the example above, `1.3.1` and `1.2.2` are the version numbers. Notice how the filename says `v1.3.1` but the version number Modrinth shows is `1.3.1`. Use the version number shown by Modrinth.

### CurseForge

On CurseForge there is no consistent "mod version" concept, so MMM uses the file name.

Once you find the mod you want to install:
1. Click the "Files" tab.
2. Find the file you want.
3. Open it and copy the file name.

![](/doc/images/versions-curseforge-1.png)
![](/doc/images/versions-curseforge-2.png)

## Platforms

The supported platforms are:
- `curseforge`
- `modrinth`

## How To Find The Mod ID

On CurseForge you need the project ID, shown on the mod page (top right).

![](/doc/images/curseforge.png)

On Modrinth you need the project slug (the last part of the URL).

![](/doc/images/modrinth.png)

Examples:
- Fabric API from CurseForge: `mmm add curseforge 306612`
- Sodium from Modrinth: `mmm add modrinth AANobbMI`
