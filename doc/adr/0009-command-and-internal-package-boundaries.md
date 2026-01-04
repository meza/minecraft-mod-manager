# 8. Command and internal package boundaries

Date: 2025-12-27

## Status

Accepted

## Context

We need a clear boundary between the frontend command layer and reusable domain packages to avoid unplanned structural changes, reduce churn, and keep code ownership obvious. Recent refactoring discussions showed that the absence of an explicit rule leads to ambiguity about when code can move out of command packages into internal packages.

Existing guidance already assumes command packages own their terminal interaction concerns (see docs/guide-to-working-with-the-terminal.md) and cmd/mmm/README.md positions cmd/mmm/<command> as the place where command behavior is implemented. We need to make the reuse boundary explicit so future refactors do not shift command-specific logic into internal packages unless it is truly shared.

## Decision

We will treat cmd/* packages as the frontend layer that owns command-specific logic, orchestration, and TUI models. Code that is unique to a single command stays in cmd/mmm/<command>.

We will use internal/* only for packages that are reused by multiple commands or shared across internal packages. New internal packages must have at least two consumers (or a clearly documented near-term second consumer) and must represent a reusable domain capability, not a thin wrapper around a single command.

When refactoring for maintainability, we will prefer extracting shared helpers into internal/* only when duplication exists across multiple commands. Otherwise, we keep the logic within cmd/* and reduce size by splitting files within the same command package.

## Consequences

Command packages remain the canonical home for command-specific logic and TUI flows, improving discoverability and preventing accidental coupling. Reusable domain logic becomes more explicit and easier to test when it truly belongs in internal/*.

This decision allows some duplication when a helper is used by only one command, which may slow deduplication until a second consumer appears. Refactors that create new internal packages now require justification and at least two consumers, which can add a small amount of upfront design work.
