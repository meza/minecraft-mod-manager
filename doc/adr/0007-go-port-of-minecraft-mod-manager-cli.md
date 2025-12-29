# 7. Go port of Minecraft Mod Manager CLI

Date: 2024-08-30

## Status

Accepted

## Context

The current TypeScript implementation has unacceptable startup latency for a CLI tool, taking around 3.5 seconds before it reaches our code. Minecraft Mod Manager is intended to be used frequently and in automated contexts, where that fixed cost is especially painful.

The current TypeScript implementation also cannot take advantage of multi-threading, which limits our ability to speed up work like downloading, checking, and verifying multiple mods in parallel.

Distribution is another constraint. Packaging Node.js applications into cross-platform executables has been brittle, and the tool we used for this (`pkg`) has been discontinued and does not support modern Node.js versions. Depending on users to manage their own Node.js installs (and matching versions across platforms) is a poor fit for the reliability expectations of a single-binary CLI.

Finally, the TypeScript version was a proof of concept that validated the product direction, but it is not an ideal base for sustained development given the performance and distribution constraints above.

## Decision

We will port Minecraft Mod Manager from TypeScript to Go. The Go implementation is the primary target for feature parity and future development.

## Consequences

The Go port addresses startup performance and enables concurrency for mod operations. It also provides a more reliable path to cross-platform distribution as a single executable.

This decision requires a full parity effort and a period where documentation and behavior may diverge between the old and new implementations. It also means that future feature work should target the Go codebase rather than the TypeScript version.
