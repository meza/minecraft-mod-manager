# <p align="center">🚀 Minecraft Mod Updater 🚀</p>

<p align="center">A command line utility to add and update Minecraft mods.</p>

<br/><br/>

It's a helpful utility for modpack creators and players who want to add and update mods in their Minecraft installation who
are using launchers that aren't capable of that.

It's also a great tool for server owners to periodically get the latest versions of the mods without having to manually
download them individually.

It currently uses the [CurseForge API](https://authors.curseforge.com/docs/api) and the [Modrinth API](https://docs.modrinth.com/api-spec/)
to fetch the mods that are described in the `modlist.json` file.

If you want support for other platforms, please feel free to submit a pull request or a feature request.

It's purposefully made to have a very explicit configuration file to avoid any "magic". This allows you to have full control
over the mods that are installed.

## Installation
Go to the releases page and download the latest version for your platform.

## Usage

### `modlist.json`

The `modlist.json` file is a JSON file that contains an array of mod objects. Each mod object has the following fields:
