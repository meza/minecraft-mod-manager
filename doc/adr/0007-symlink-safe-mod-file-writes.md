# 7. Symlink-safe mod file writes

Date: 2025-12-20

## Status

Accepted

## Context

MMM downloads and replaces jar files in a user-controlled mods folder. Server operators often use symlinks to manage their layout, but the downloader previously wrote directly to the destination path and could follow symlinks to locations outside the mods folder. This creates a risk of overwriting arbitrary files while still needing to support intentional symlink workflows.

We need a strategy that preserves common symlink usage while preventing writes outside the mods folder.

## Decision

We will allow symlinks in user layouts while enforcing a strict write boundary:

- The mods folder may be a symlink. We resolve it to a canonical root for safety checks.
- If a mod file path does not exist, we write via a temp file in the destination directory, verify the hash, then rename into place.
- If a mod file path exists and is a symlink, we resolve its target:
  - If the resolved target is inside the resolved mods folder, we write to the target file (temp + verify + rename) and preserve the symlink.
  - If the resolved target is outside the resolved mods folder, we refuse and return a clear error explaining why.

This applies to add, install, and update downloads so behavior is consistent across commands.

## Consequences

- Users can keep symlink-based layouts inside the mods folder without MMM replacing their symlinks.
- MMM will refuse to write to symlink targets outside the mods folder, preventing accidental or malicious overwrites.
- Download flows now perform path resolution and temp writes, adding small overhead and new error cases that must be surfaced clearly.
