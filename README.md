# <p align="center">🚀 Minecraft Mod Manager 🚀</p>

<p align="center">A command line utility to install and update Minecraft mods (for the Java edition) without a launcher.</p>
<p align="center">
  <a href="https://github.com/users/meza/projects/5/views/4" target="_blank">Roadmap</a> |
  <a href="https://github.com/users/meza/projects/5/views/1" target="_blank">Project Board</a> |
  <a href="https://github.com/meza/minecraft-mod-manager/milestones" target="_blank">Upcoming Milestones</a>
  <br/><br/></p>

<p align="center">
<img src="https://img.shields.io/badge/dynamic/json.svg?style=plastic&color=2096F3&label=locize&query=%24.translatedPercentage&url=https://api.locize.app/badgedata/f946883c-b1ab-4486-90f2-68cf6c25026f&suffix=%+translated&link=https://www.locize.com" />
</p>

Minecraft Mod Manager is a helpful utility for players, modpack creators and server owners who want to keep their
Minecraft mods up to date without the need for a launcher or having to manually check and download new files.

It can currently use mods from [Curseforge](https://curseforge.com/minecraft) and [Modrinth](https://modrinth.com/).
If you want support for other platforms, please feel free to submit a pull request or a feature request.


You currently can

- [add mods](docs/commands/add.md)
- [remove mods](#remove)
- [automatically update mods](#update)
- [change minecraft versions and uppdate the mods](#change)
- [automatically recognize manually added files](#scan)

Upcoming features:

- [manage mods with dependencies](https://github.com/meza/minecraft-mod-manager/issues/203)
- consolidate mods to the same platform
- use github as the source for mods
- self-update

It's purposefully made to have a very explicit configuration file to avoid any "magic". This allows you to have full
control over the mods that are installed.

<p align="center">
<strong>Minecraft Mod Manager</strong> strives to be a tool that is easy to use and understand.
<br/>You can help translate it on Crowdin.
<br/>
<br/>
</p>

---

### Table Of Contents

* [Installation](#installation)
* [Running](#running)
* [How it works](#how-it-works)
  * [INIT](docs/commands/init.md)
  * [ADD](docs/commands/add.md)
    * [Platforms](docs/commands/add.md#platforms)
    * [How To Find The Mod ID](docs/commands/add.md#how-to-find-the-mod-id)
  * [REMOVE](docs/commands/remove.md)
  * [INSTALL](docs/commands/install.md)
  * [UPDATE](#update)
  * [CHANGE](#change)
  * [LIST](docs/commands/list.md)
  * [TEST](#test)
  * [PRUNE](#prune)
  * [SCAN](#scan)
* [Explaining the configuration](#explaining-the-configuration)
  * [modlist-lock.json](#modlist-lockjson)
  * [modlist.json](#modlistjson)
    * [loader](#loader-_required)
    * [gameVersion](#gameversion-required)
    * [modsFolder](#modsfolder-required)
    * [defaultAllowedReleaseTypes](#defaultallowedreleasetypes-required)
    * [allowVersionFallback](#allowversionfallback-optional)
  * [.mmmignore](#ignore-file)
* [Using with MultiMC](#using-with-multimc)
* [Contribute to the project](#contribute-to-the-project)
  * [Setup](#setup)
    * [Prerequisites](#prerequisites)
    * [Install dependencies](#install-dependencies)
    * [Validate](#validate)
  * [Code Considerations](#code-considerations)
    * [Conventional commmit messages](#conventional-commmit-messages)
    * [Using `console.log` and `console.error`](#using-consolelog-and-consoleerror)

<!-- TOC -->

---

## Installation

Go to the [releases page](https://github.com/meza/minecraft-mod-manager/releases), find the latest release,
**click on the Assets word** and download the latest version for your platform.

For the best results, put the downloaded executable into your minecraft folder in the same level as the `mods` folder.

![Folder Structure](/doc/images/mmm-folder-structure.png)

---

## Running

To use the tool, **you need to have a command line / terminal open** and be in the folder where the tool is.

<details>
<summary>Click for help with opening a terminal in Windows</summary>

<br/>

1. Navigate to the folder where the `mmm.exe` (and the rest of your minecraft installation) exists
2. Click to the address bar
3. Type: cmd and hit enter

![](/doc/images/cmd-windows.gif)

</details>

<details>
<summary>Click for help with opening a terminal in Linux and MacOS</summary>

<br/>

Let's be honest, you already know...
</details>
---

## How it works

> _If you know `npm` or `yarn` from the web development world, this works just the same_

Every command has a help page that you can access by running `mmm help <command>`.

__Common Options__

Every command has a few common options that you can use:

| Option Short | Option Long | Description                                |
|--------------|-------------|--------------------------------------------|
| -q           | --quiet     | Suppress all interactive ui elements       |
| -c           | --config    | Set the config file to an alternative path |
| -d           | --debug     | Enable verbose logging                     |
|              | --perf      | Write `mmm-perf.json` when the command exits |
|              | --perf-out-dir | Directory to write `mmm-perf.json` (defaults to the config file directory) |

All options should be specified **before** the command. For example:

```bash
mmm --quiet install
```

or

```bash
mmm -c ./my-config.json install
```

### Performance logs

If a command feels slow, you can ask MMM to write a performance log so you can see where time is spent:

```bash
mmm --perf add modrinth AANobbMI
```

By default the file is written next to your `modlist.json` as `mmm-perf.json`. Use `--perf-out-dir` to place it in a subdirectory:

```bash
mmm --perf --perf-out-dir perf add modrinth AANobbMI
```

Paths inside the perf file are normalized to be relative to the config directory so you can share the file without leaking machine-specific path prefixes.

### Telemetry

Minecraft Mod Manager records anonymized command metadata with [PostHog](https://posthog.com) so we know which flows succeed and where errors cluster. Command events include a performance trace (spans) so we can see where time is spent. Telemetry is best-effort and never blocks a command.

Opt out anytime by setting an environment variable before running the CLI. For example:

```bash
# macOS/Linux
MMM_DISABLE_TELEMETRY=1 mmm list

# Windows PowerShell
$env:MMM_DISABLE_TELEMETRY=1; mmm list
```

---

### UPDATE

`mmm update` or `mmm u`

This will try and find newer versions of every mod defined in the `modlist.json` file that matches the given game
version, loader and doesn't have a fixed `version` configuration.
If a new mod is found, it will be downloaded and the old one will be removed. If the download fails,
the old one will be kept.

You would run this command when you want to make sure that you're using the newest versions of the mods.

Due to the Minecraft modding community's lack of consistent versioning, the "newness" of a mod is defined by the release
date of a file being newer than the old one + the hash of the file being different.

---

### CHANGE

`mmm change [-f] [game_version]`

This will attempt to change all the mods that are configured for the mod manager to the supplied
minecraft version.

If no version is given, the command will assume the most recent release version of Minecraft.

It will perform the same check as the `mmm test` would before attempting a change so if either
of the configured mods doesn't support the new game version, the change will not happen.

The process of a `mmm change` is the equivalent of running `mmm test`, deleting all the current mod files
from the configured mods directory, changing the `gameVersion` in the `modlist.json`, then running a
`mmm install`.

The exit codes of this command are identical to the [test](#test) command's.

#### Command line arguments for the change function

| Short | Long    | Description                                                                                                                                                                                                               | Value | Example         |
|-------|---------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------|-----------------|
| -f    | --force | Force the change of the game version. Deletes all the mods and attempts to install with the given game version. Use at your own risk.<br/>If a mod doesn't have support for your game version, the mod won't be installed |       | `mmm change -f` |


---

### TEST

`mmm test [game_version]`

Test if you can use the specified game version. This is most commonly used to see if you can upgrade to a newer version
of Minecraft and _test_ that all of your configured mods will have a version for it.

For example if you're on 1.19.2, and you want to see if you could upgrade to 1.19.3, you would run: `mmm test 1.19.3`

If you omit the game version, it will use the latest stable minecraft version.

**For server operators and script automation, the command will have a non-zero (1) exit value when it finds mods that
don't support the version you are testing for.**

It will also return a non-zero (2) exit value when you're testing for the version that's already being used.

This means that you could run `mmm test` every day for example, which would in return always check for the latest
Minecraft release. Whenever the command returns with a zero exit code, you can run a version upgrade on your server if
you like.

---

### PRUNE

Removes all unmanaged files from the mod directory.

#### Ignoring files

The prune command adheres to the [.mmmignore](#ignore-file) file and will not process any files specified there.

#### Command line arguments for the prune function

| Short | Long    | Description                                               | Value                     | Example        |
|-------|---------|-----------------------------------------------------------|---------------------------|----------------|
| -f    | --force | Delete the files without asking                           |                           | `mmm prune -f` |

---

### SCAN

Scans the configured mods folder and looks for files that are currently not managed by the mod manager.
When a file is found, it will attempt to look up that file on all the supported platforms.

> [!NOTE]
> Files ending in `.disabled` will be ignored.

It will report back the findings and if executed without any extra parameters, depending on the [interactivity settings](#how-it-works),
it will either ask you what to do or not do anything.

If you supply the `--add` flag, it will add the discovered files to your modlist json.

#### What if I don't specify a preferred platform?

If you don't specify a preferred platform, it will use `Modrinth`. It does not search on both at the same time, ever.

#### Will it delete the files that it found?

No. It will reuse the found files and add them to the lockfile so you can decide if you want to then update to the newest
versions or not.

#### Command line arguments for the scan function

| Short | Long     | Description                                               | Value                      | Default    | Example                  |
|-------|----------|-----------------------------------------------------------|----------------------------|------------|--------------------------|
| -p    | --prefer | Which platform do you prefer to use?                      | `curseforge` or `modrinth` | `modrinth` | `mmm scan -p curseforge` |
| -a    | --add    | Automatically add the discovered mods to the modlist json |                            |            | `mmm scan -a`            |

---

## Explaining the configuration

### modlist-lock.json

You have seen this file mentioned in this document and you might be wondering what to do with it.

The lockfile is 100% managed by the app itself and it ensures consistency across [`install`](#install) runs. It
effectively "locks" the versions to the exact versions you installed with the last [`add`](docs/commands/add.md) or [`update`](#update)
commands.

**You don't have to do anything with it!**

If you use version control to manage your server/modpack/configuration then make sure to commit **both**
the `modlist.json` and the `modlist-lock.json`. Together they ensure that you are in full control of what gets
installed.

### modlist.json

The modlist.json is the main configuration file of Minecraft Mod Manager.

It is in JSON format. If you're unfamiliar with JSON or want to make sure that everything is in order, please use
the [JSON Validator](https://jsonlint.com/) website to make sure that the file contents are valid before running the
app.

This is how it looks like if you followed the examples in the [`add`](docs/commands/add.md) command documentation:

```json
{
  "loader": "fabric",
  "gameVersion": "1.19.2",
  "modsFolder": "mods",
  "defaultAllowedReleaseTypes": [
    "release",
    "beta"
  ],
  "mods": [
    {
      "type": "curseforge",
      "id": "306612",
      "name": "Fabric API",
      "allowedReleaseTypes": [
        "release"
      ]
    },
    {
      "type": "modrinth",
      "id": "AANobbMI",
      "name": "Sodium",
      "version": "0.5.5"
    },
    {
      "type": "modrinth",
      "id": "YL57xq9U",
      "name": "Iris Shaders"
    }
  ]
}
```

> The **mods** field is managed by the [`add`](docs/commands/add.md) command, but you can also edit it by hand if you wish.

#### loader _required_

Possible values: `fabric`, `quilt`, `forge`

The loader defines which minecraft loader you're using.

#### gameVersion _required_

This needs to be the game version as listed by Mojang. `1.19`, `1.19.1`, `1.19.2`, etc

#### modsFolder _required_

This points to your mods folder. Traditionally it would be "mods" but you can modify it to whatever your situation
needs.
The value of this could be an absolute path or a relative path.

We recommend you use relative paths as they are more portable.

> __PRO TIP__
>
> Keep the `modlist.json` file in the root of your minecraft installation. Right next to the `server.properties` file.
>
> If the mods folder is relative, it will be a relative path from the modlist.json file. This makes it so that you can
> easily include the modlist json with your modpack or multimc instance so others could make use of it too.

#### defaultAllowedReleaseTypes _required_

Possible values is one or all of the following: `alpha`, `beta`, `release`

You can override this on a per-mod basis with the `allowedReleaseTypes` field in the mod definition.

<details>
  <summary>Example</summary>

To lock Fabric Api to only release versions when everything else could be beta too, use it like below:

```json
{
  ...
  "mods": [
    {
      "type": "curseforge",
      "id": "306612",
      "name": "Fabric API",
      "allowedReleaseTypes": [
        "release"
      ]
    },
    ...
  ]
}
```

</details>

#### allowVersionFallback _optional_

Every mod entry may optionally include `allowVersionFallback`. Setting it to `true` for a mod entry will:

- If a suitable mod isn't found for the given Minecraft version, say 1.19.2, it will try for 1.19.1 (the previous minor
  version)
- If a suitable mod isn't found for the previous minor version, it will try for 1.19 (the previous major version)

This happens quite frequently unfortunately because mod developers either don't update their mods but they still work or
they forget to list the supported Minecraft versions correctly.

If `allowVersionFallback` is omitted, the CLI assumes `false` for that mod. There is currently no global fallback flag.

#### version _optional_

For every mod you can specify a version. This is useful if you want to install a specific version of a mod and want to
keep it that way regardless of any updates to the mod.

There are subtle differences between how this works for Modrinth and Curseforge. To learn more about this, read
the [Installing Specific Versions](docs/commands/add.md#installing-specific-versions) section of the [add](docs/commands/add.md) command.

### Ignore File

Ignoring files works pretty much the same way as it does with [.gitignore](https://git-scm.com/docs/gitignore).

You have to create a `.mmmignore` file in the same directory as your `modlist.json` file is.
Having files listed in the `.mmmignore` will make all operations ignore the given file like it doesn't exist.

Each line within the ignore file is a Glob Pattern.

The patterns will be applied by taking the directory of the modlist.json file's directory as the starting point.

> This will change in the future. If you would like it to change sooner, please open an issue on github

For example to ignore the worldedit and the modmenu mods, the `.mmmignore` file would have the following entries:

```
mods/modmenu-*.jar
mods/worldedit-*.jar
```

##### Dots

If a file or directory path portion has a `.` as the first character,
then it will not match any glob pattern unless that pattern's
corresponding path part also has a `.` as its first character.

For example, the pattern `a/.*/c` would match the file at `a/.b/c`.
However the pattern `a/*/c` would not, because `*` does not start with
a dot character.

You can make glob treat dots as normal characters by setting
`dot:true` in the options.

---

## Using with MultiMC

MultiMC is a great tool for managing your Minecraft instances. However, it lacks the capability to keep the mods updated.

You can use Minecraft Mod Manager to keep your mods up to date automatically.

Step 1: Make sure that you have `mmm` in the .minecraft folder of your instance.

Step 2: Edit your instance and go to "Settings" on the left hand side

Step 3: Click on Custom Commands

Step 4: Set the following for the "Pre-launch command" `"$INST_MC_DIR/mmm.exe" update`

It should look something like this:

![](/doc/images/multimc.png)

## Using your own API keys

You can override the modrinth and curseforge API keys by setting the `MODRINTH_API_KEY` and `CURSEFORGE_API_KEY` environment variables.

You can use whatever means exist on your operating system to set these variables. For example, on Windows you can use the
`set` command:

```cmd
set MODRINTH_API_KEY=your-api-key
set CURSEFORGE_API_KEY=your-api-key
```

On Linux and MacOS you can use the `export` command:

```bash
export MODRINTH_API_KEY=your-api-key
export CURSEFORGE_API_KEY=your-api-key
```

You can also set these variables in a .env file in the same directory as the `modlist.json` file.

```env
MODRINTH_API_KEY=your-api-key
CURSEFORGE_API_KEY=your-api-key
```
Under normal circumstances, you should not need to set these variables. The tool will work without them. However, if you
run into rate limiting issues or are using private projects, you can set these variables to your own API keys.

> **Please be aware that the API keys are sensitive information. Do not share them with anyone.**



<br/><hr/>

## <p align="center">Contribute to the project</p>

Feel free to contribute to the project but please read the [contribution guidelines](CONTRIBUTING.md) first before
making any changes.

### Setup

#### Prerequisites

- [Node.js](https://nodejs.org/en/) (v18.10 or higher)
- [pnpm](https://pnpm.io) (v7.13.4 or higher)

#### Install dependencies

```bash
pnpm install
```

#### Validate

```bash
pnpm ci
 ```

### Code Considerations

#### Conventional commmit messages

We use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) to generate the changelog and
to automatically bump the version number.

> **Changes not following the Conventional Commits specification will be rejected.**

---

#### Unit testing

The project has been written in a TDD fashion and all contributions are required to have full and meaningful coverage.

> _100% coverage is just the bare minimum_

Untested pull requests will be rejected.

---

#### Linting rules

`pnpm lint` or the `pnpm ci` will apply the supplied linting rules.

##### DO NOT

- suppress linting errors unless there is absolutely no way around them
- modify the rules to make them less strict
- argue about the rules

##### DO

- create contributions that make the rules more coherent
- ask for help if you don't understand why a rule is in place (but first please look up the violation)

##### Exceptions

Sometimes when dealing with external sources, things like `snake_case_names` are inevitable. Those can be suppressed on
the line they occur.

---

#### Documentation

Make sure additions are well documented with the same language and style as the main readme is.

---

#### Using `console.log` and `console.error`

> To make sure that we communicate with the user on the right level,
> all invocations to the `console.log` and the `console.error` functions should
> be done in the `actions` folder. This means that the `console.log` and the
> `console.error` functions should not be used in the `lib` folder.

Read more about this in the [Architecture Decision Record](doc/adr/0002-console-log-only-in-actions.md).
