# Modrinth

Modrinth is a Minecraft project repository for mods, datapacks, resource packs, shaders and modpacks.

## What do we do with Modrinth?

`mmm` uses Modrinth as one of the sources for managed files

## API Docs

The currently up-to-date API docs can be found at https://docs.modrinth.com/

## How we use the api?

### User Agent

We use the following user agent: `github_com/meza/minecraft-mod-manager/${version}` where the `${version}` is
replaced at build time by the automation via the `environment.go` file.
