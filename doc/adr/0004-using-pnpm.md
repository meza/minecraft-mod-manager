# 4. using pnpm

Date: 2022-10-02

## Status

Accepted

Causes [5. Using Renovate bot](0005-using-renovate-bot.md)

## Context

Yarn was getting slow at installing dependencies and we were looking for a faster alternative.
Unfortunately yarn2 or 3 isn't supported by dependabot.

## Decision

Yarn2+ doesn't seem to be all that popular to begin with so we decided to go with pnpm which seemingly is more
well liked.

## Consequences

- pnpm is seemingly a lot faster than yarn.
- dependabot doesn't support pnpm either
