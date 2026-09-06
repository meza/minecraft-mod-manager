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

The [project glossary](docs/GLOSSARY.md) defines the canonical product and architecture vocabulary for MMM's Go-port target.

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

It's purposefully made to have a very explicit modlist to avoid any "magic". This allows you to have full
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
  * [TEST](docs/commands/test.md)
  * [PRUNE](#prune)
  * [SCAN](#scan)
* [Modlist and lockfile](#modlist-and-lockfile)
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
|              | --unattended | Disable prompts; the Go-port target also selects plain, append-only pure CLI output |
| -q           | --quiet     | Suppress non-essential output (errors and required results still print) |
| -c           | --config    | Select an alternative modlist file |
| -d           | --debug     | Enable verbose logging                     |
|              | --perf      | Write `mmm-perf.json` when the command exits |
|              | --perf-out-dir | Directory to write `mmm-perf.json` (defaults to the configuration directory) |

All options should be specified **before** the command. For example:

```bash
mmm --quiet install
```

or

```bash
mmm -c ./my-config.json install
```

To avoid prompts in scripts, add `--unattended`:

```bash
mmm --unattended init -l fabric -g 1.21.1 -m ./mods
```

Supplying every argument can avoid a question, but it does not select unattended execution. The Go-port target defines five terminal profiles and requires `--unattended` output to remain plain even when attached to a terminal. Current builds can still render rich unattended output, do not provide complete ASCII presentation, and do not yet route plain line-oriented questions across command flows. See [execution modes and operator intent](docs/intent.md#execution-modes-and-operator-intent) for the target contract.

### Performance logs

If a command feels slow, you can ask MMM to write a performance log so you can see where time is spent:

```bash
mmm --perf add modrinth AANobbMI
```

By default the file is written next to your `modlist.json` as `mmm-perf.json`. Use `--perf-out-dir` to place it in a subdirectory:

```bash
mmm --perf --perf-out-dir perf add modrinth AANobbMI
```

Paths inside the performance recording are normalized to be relative to the configuration directory so you can share the file without leaking machine-specific path prefixes.

### Telemetry

Minecraft Mod Manager records operational command metadata with [PostHog](https://posthog.com) so we know which flows succeed and where errors cluster. Telemetry uses a stable machine identifier that is not PII to track long-term behavior. Session events include a `performance` payload (perf_summary_v1 schema: app version, OS, execution mode, ordered commands, command timings, modlist context, and request/download counts); raw perf span trees are only written to `mmm-perf.json` when you opt in with `--perf`. Telemetry is best-effort and never blocks a command.

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

For each unpinned mod config in the modlist, this looks up a newer artifact matching the Minecraft target and loader.
If a newer eligible artifact is found, it will be downloaded and the previous file will be removed. If the download fails,
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

The exit codes of this command are identical to the [test](docs/commands/test.md) command's.

#### Command line arguments for the change function

| Short | Long    | Description                                                                                                                                                                                                               | Value | Example         |
|-------|---------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------|-----------------|
| -f    | --force | Force the change of the game version. Deletes all the mods and attempts to install with the given game version. Use at your own risk.<br/>If a mod doesn't have support for your game version, the mod won't be installed |       | `mmm change -f` |


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
> Files ending in `.disabled` are excluded.

It will report back the findings and if executed without any extra parameters, depending on the [interactivity settings](#how-it-works),
it will either ask you what to do or not do anything.

If you supply the `--add` flag, it will adopt recognized artifacts by recording their mod configs in the modlist and their exact artifact information in the lockfile.

#### What if I don't specify a preferred platform?

If you don't specify a preferred platform, it will use `Modrinth`. It does not search on both at the same time, ever.

#### Will it delete the files that it found?

No. It will reuse the found files and record their artifacts in the lockfile so you can decide if you want to then update to the newest
versions or not.

#### Command line arguments for the scan function

| Short | Long     | Description                                               | Value                      | Default    | Example                  |
|-------|----------|-----------------------------------------------------------|----------------------------|------------|--------------------------|
| -p    | --prefer | Which platform do you prefer to use?                      | `curseforge` or `modrinth` | `modrinth` | `mmm scan -p curseforge` |
| -a    | --add    | Adopt recognized artifacts into the modlist and lockfile |                            |            | `mmm scan -a`            |

---

## Modlist and lockfile

### modlist-lock.json

You have seen this file mentioned in this document and you might be wondering what to do with it.

The lockfile is managed by MMM and records exact resolved artifacts for consistent [`install`](docs/commands/install.md)
runs. The [`add`](docs/commands/add.md) and [`update`](#update) commands record the artifacts they resolve in it.

**You don't have to do anything with it!**

If the lockfile contains artifacts for mods that are missing from your modlist, MMM will stop and ask how to reconcile them (add a mod config, delete from disk, ignore, or do nothing). In non-interactive or unattended runs, it defaults to add unless you pass a lockfile sync policy flag like `--lock-sync-ignore`. See [lockfile sync](docs/commands/lockfile-sync.md) for the full flow.

If you use version control to manage your server/modpack/configuration then make sure to commit **both**
the `modlist.json` and the `modlist-lock.json`. Together they ensure that you are in full control of what gets
installed.

### modlist.json

The modlist, stored by default in `modlist.json`, contains installation-wide settings and a mod config for each desired mod.

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

> Each entry in **mods** is a mod config. You can manage these entries with [`add`](docs/commands/add.md) or edit them by hand.

#### loader _required_

Possible values: `fabric`, `quilt`, `forge`

The loader identifies the Minecraft mod-loading software your game or server uses. MMM matches that identity against platform metadata during artifact lookup.

#### gameVersion _required_

This needs to be the game version as listed by Mojang. `1.19`, `1.19.1`, `1.19.2`, etc

#### modsFolder _required_

This points to your mods folder. Traditionally it would be "mods" but you can modify it to whatever your situation
needs.
The value of this could be an absolute path or a relative path.

We recommend you use relative paths as they are more portable.

Important: the modlist's `modsFolder` is trusted input. MMM writes and deletes files in the folder you point it at.
Only run MMM against modlist files you trust, especially when `modsFolder` is absolute or points outside the configuration directory.

> __PRO TIP__
>
> Keep the `modlist.json` file in the root of your minecraft installation. Right next to the `server.properties` file.
>
> If the mods folder is relative, it will be a relative path from the modlist.json file. This makes it so that you can
> easily include the modlist json with your modpack or multimc instance so others could make use of it too.

#### defaultAllowedReleaseTypes _required_

Possible values is one or all of the following: `alpha`, `beta`, `release`

You can override this on a per-mod basis with the `allowedReleaseTypes` field in the mod config.

<details>
  <summary>Example</summary>

To allow only release artifacts for Fabric API while the modlist's default release policy also permits beta artifacts, set its mod config like this:

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

Every mod config may optionally include `allowVersionFallback`. Setting it to `true` permits version fallback:

- If no eligible artifact is found for the Minecraft target, say 1.19.2, lookup tries 1.19.1, an earlier release in the same series.
- If no eligible artifact is found for 1.19.1, lookup tries 1.19, an earlier release in that same series.

This happens quite frequently unfortunately because mod developers either don't update their mods but they still work or
they forget to list the supported Minecraft versions correctly.

If `allowVersionFallback` is omitted, the CLI assumes `false` for that mod. There is currently no global fallback flag.

#### version _optional_

Set `version` in a mod config to pin that mod to an explicit platform-specific version. Ordinary updates preserve the pin.

There are subtle differences between how this works for Modrinth and Curseforge. To learn more about this, read
the [Installing Specific Versions](docs/commands/add.md#installing-specific-versions) section of the [add](docs/commands/add.md) command.

### Ignore File

Ignoring files works pretty much the same way as it does with [.gitignore](https://git-scm.com/docs/gitignore).

You have to create a `.mmmignore` file in the same directory as your `modlist.json` file is.
Having files listed in the `.mmmignore` will make all operations ignore the given file like it doesn't exist.

Each line within the ignore file is a Glob Pattern.

Patterns are evaluated relative to your mods folder. If you use subfolders under your mods directory, include them in the pattern.

For example to ignore the worldedit and the modmenu mods, the `.mmmignore` file would have the following entries:

```
modmenu-*.jar
worldedit-*.jar
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

Official `mmm` releases ship with built-in (embedded) API keys so they can work out of the box.

If you need to use your own keys (for example: you are hitting rate limits, you need access to private projects, or you want to route telemetry to your own PostHog project), you can override the embedded defaults at runtime via environment variables or a `.env` file.

Precedence is:

1. `.env` / environment variables you set at runtime (if you set an empty value, it overrides the embedded default)
2. Embedded defaults in the `mmm` binary

If you build `mmm` yourself, `make build` embeds token defaults from your shell environment and/or the repo-root `./.env` file, and fails if any are missing.

You can use whatever means exist on your operating system to set these variables. For example, on Windows you can use the
`set` command:

```cmd
set MODRINTH_API_KEY=your-api-key
set CURSEFORGE_API_KEY=your-api-key
set POSTHOG_API_KEY=your-posthog-key
```

On Linux and MacOS you can use the `export` command:

```bash
export MODRINTH_API_KEY=your-api-key
export CURSEFORGE_API_KEY=your-api-key
export POSTHOG_API_KEY=your-posthog-key
```

You can also set these variables in a `.env` file in the directory where you run `mmm`.

```env
MODRINTH_API_KEY=your-api-key
CURSEFORGE_API_KEY=your-api-key
POSTHOG_API_KEY=your-posthog-key
```
Under normal circumstances, you should not need to set these variables. The tool will work without them. However, if you
run into rate limiting issues or are using private projects, you can set these variables to your own API keys.

> **Please be aware that the API keys are sensitive information. Do not share them with anyone.**



<br/><hr/>

## Contribute to the project

Start with the [contribution guide](CONTRIBUTING.md) to choose the guidance and verification
for your work, including product design, code, tests, documentation, translations and releases.
