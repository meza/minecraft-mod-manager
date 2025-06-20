# TUI Design Plan

This document outlines the planned structure for the Bubbletea based interface.

## Goals
- Provide a fullscreen terminal UI for managing all commands.
- Display a header with the application name and version.
- Navigate using the keyboard with clear key hints.
- Mirror the behaviour of the existing CLI commands.

## Layout
1. **Header** – shows "Minecraft Mod Manager" and the current version.
2. **Main Menu** – list of operations:
   - init
   - add
   - install
   - update
   - list
   - change
   - test
   - prune
   - scan
   - remove
3. **Footer** – help text for key bindings (`↑/↓` to move, `enter` to select, `q` to quit).

Each menu entry opens a dedicated Bubbletea model that implements the logic described in the docs under `docs/commands/`.

## Implementation Notes
- Models must remain testable and free from side effects.
- Views should rely on the styling helpers under `internal/tui`.
- The `main` command starts the TUI when run without sub‑commands.
