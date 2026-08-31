# Minecraft Mod Manager

Minecraft Mod Manager is a Go command-line application for installing and updating Minecraft mods without a launcher.

## Load project guidance progressively

- Read the root `README.md` for product context and `CONTRIBUTING.md` for contributor requirements.
- Before working on an affected path, inspect its directory and every ancestor through the repository root for `README.md` and `CONTRIBUTING.md` files. Read every one found: localized files provide context and govern work within their directory tree.
- Do not recursively scan unrelated directory trees. Load other guidance when its scope or an applicable skill routes to it.
- Use applicable project skills from `.agents/skills` before relying on general knowledge.

## Manage project skills

`skills-lock.json` identifies third-party skills installed into this repository. Treat those skill directories and their bundled resources as managed dependencies: do not edit them directly. Update them only through the skills tool in a separately authorized dependency update.

First-party skills not listed in `skills-lock.json` may define repository-specific workflows. Keep detailed task policy in the canonical repository document named by the skill rather than duplicating it in the skill entry point.
