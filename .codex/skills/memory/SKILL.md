---
name: memory
description: Maintain persistent project memory across work sessions for continuity when work is interrupted or contexts shift. Use when capturing insights about project architecture, codebase understanding, team patterns, unresolved questions, or assumptions needing validation. Also use at session start to review recent entries and at session end to capture what changed.
---

## Access Control

**User-facing agents**: Full READ and WRITE access to long-term memory.

**All other agents** (subagents, delegated agents, spawned tasks): READ access only. Do not write, modify, or delete memory entries.

This distinction exists because memory entries require human-facing context to be meaningful. Subagents lack the session continuity needed to make appropriate memory decisions.

## Purpose

Long-term memory enables continuity across interrupted work sessions. It captures insights that would otherwise be lost when context resets.

## What to Document

- Insights learned about the project, its architecture, and its dependencies
- Personal understanding of codebases and team patterns
- Unresolved questions or assumptions needing validation

## What to Avoid

- Content belonging in issue trackers or formal documentation
- Architectural decisions (use ADR skill instead)
- Information already captured elsewhere in the project

## File Location

Store all entries in `memory.tsv` at the repository root.

## Reading the Memory File

**CRITICAL: The memory file cannot be read in one operation.**

The memory file grows over time and will exceed single-read limits. You MUST read it in chunks.

**Reading strategy:**

1. Start by reading the END of the file first (most recent entries are at the bottom)
2. Use offset and limit parameters to read manageable chunks (500-1000 lines)
3. Work backwards from the end to find recent context
4. For session start, the last 500-1000 lines typically contain sufficient recent context

**Example approach:**

- First read: offset from end, limit 500 lines (gets most recent entries)
- If more context needed: read the preceding chunk
- Continue as necessary based on the task at hand

**Never assume the file is small enough for a single read.** Always use chunked reading.

## Entry Format

Use the actual system timestamp when creating entries.
Keep entries to a single TSV row. Do not add blank lines.
Do not include tab characters in the entry column (replace with spaces).

## Session Practices

**At session start**: Review recent entries to reestablish context.

**During work**: Log insights immediately as they emerge. Do not postpone.

**At session end**: Capture what changed, assumptions to validate, and next steps.

**For resolved items**: Prefix the entry text with [RESOLVED] rather than deleting.

**Periodically**: Consolidate completed work and expunge fully resolved items to keep open questions scannable.

## Tone

Keep entries factual and concise. This is internal continuity documentation, not external communication.
