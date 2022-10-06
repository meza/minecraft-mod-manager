# <p align="center">🚀 Minecraft Mod Manager 🚀</p>

<p align="center">A command line utility to add and install Minecraft mods.</p>

<br/><br/>

It's a helpful utility for modpack creators and players who want to add and install mods in their Minecraft installation
who
are using launchers that aren't capable of that.

It's also a great tool for server owners to periodically get the latest versions of the mods without having to manually
download them individually.

It currently uses the [CurseForge API](https://authors.curseforge.com/docs/api) and
the [Modrinth API](https://docs.modrinth.com/api-spec/)
to fetch the mods that are described in the `modlist.json` file.

If you want support for other platforms, please feel free to submit a pull request or a feature request.

It's purposefully made to have a very explicit configuration file to avoid any "magic". This allows you to have full
control
over the mods that are installed.

## Installation

Go to the [releases page](https://github.com/meza/minecraft-mod-manager/releases) and download the latest version for your platform.

## Usage

The Minecraft Mod Manager uses the `modlist.json` file to configure how and what mods are installed.

### Adding a mod

To add a mod, you need to specify which platform the mod is from
and then the project/mod's ID.

`mmm add <type> <modId>`

#### How to find the mod ID?

[//]: # (TODO: Add a section on how to find the mod ID for each platform)

#### Examples

Adding the Fabric API: `mmm add curseforge 306612`

Adding Sodium: `mmm add modrinth AANobbMI`

### `modlist.json`

The `modlist.json` file is a JSON file that contains an array of mod objects. Each mod object has the following fields:

<br/><hr/>

## <p align="center">Contribute to the project</p>

Feel free to contribute to the project but please read the [contribution guidelines](CONTRIBUTING.md) first before
making any changes.

### Setup

#### Prerequisites

- [Node.js](https://nodejs.org/en/) (v18 or higher)
- [Yarn](https://yarnpkg.com/) (v1.22 or higher)

#### Install dependencies

```bash
yarn install
```

#### Test

```bash
yarn test
 ```

### Code Considerations

#### Conventional commmit messages

We use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) to generate the changelog and
to automatically bump the version number.

> **Changes not following the Conventional Commits specification will be rejected.**

#### Using `console.log` and `console.error`

> To make sure that we communicate with the user on the right level,
> all invocations to the `console.log` and the `console.error` functions should
> be done in the `actions` folder. This means that the `console.log` and the
> `console.error` functions should not be used in the `lib` folder.

Read more about this in the [Architecture Decision Record](doc/adr/0002-console-log-only-in-actions.md).
