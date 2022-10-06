# 3. Rename the project to Minecraft Mod Manager

Date: 2022-09-30

## Status

Accepted

## Context

The original name of the project and the repo was Minecraft Mod Updater as the original scope was to just be able to
update mods.

However, during the development the similarity between npm/yarn has been made and since npm is "node package manager",
it would make sense to have a "minecraft package manager" which is what this project is.

## Decision

Since the packages in Minecraft are called mods, the project will be renamed to Minecraft Mod Manager.

## Consequences

- The project will be renamed to Minecraft Mod Manager and the repo will be renamed to `minecraft-mod-manager`.
- The executable will be renamed to `mmm` (Minecraft Mod Manager).
