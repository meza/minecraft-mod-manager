# 5. Using Renovate bot

Date: 2022-10-02

## Status

Accepted

Caused by [4. Using pnpm](0004-using-pnpm.md)

## Context

We used Dependabot previously, but it's severely lacking in regard to modern package managers like pnpm and yarn2+.

## Decision

We decided to use Renovate bot instead.

## Consequences

- Renovate bot is a lot more configurable than Dependabot.
- Renovate bot supports pnpm and yarn2+.
