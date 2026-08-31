---
name: curseforge-api
description: CurseForge API Gate - Use when implementing, updating, or debugging CurseForge API integration (endpoints, x-api-key auth header, rate limits, pagination, error handling) or when working with the CurseForge REST API docs.
---

# CurseForge API

## When to Use

Use this skill when a task involves any of the following:
- Adding or modifying CurseForge API calls (any HTTP to `api.curseforge.com`).
- Updating request/response parsing for CurseForge resources (mods, files, fingerprints, search).
- Investigating CurseForge API errors (auth failures, rate limits, unexpected response shapes).
- Needing up-to-date endpoint definitions, parameters, or schemas via `https://docs.curseforge.com/rest-api/`.

## Core Approach

1. Prefer authoritative sources: read the official CurseForge REST API docs for up-to-date truth.
2. Keep repo behavior consistent: align with `docs/platform-apis.md` before changing behavior.
3. Be conservative under uncertainty: label assumptions, verify in docs, and avoid inventing endpoints or fields.
4. Protect secrets: never log or persist API keys; treat `x-api-key` values as sensitive.

## Workflow

1. Read local expectations first (repo-specific)
   - Read `docs/platform-apis.md` and confirm the intended CurseForge behaviors (headers, rate limiting, pagination, endpoint usage).

2. Acquire the current CurseForge REST API docs
   - Docs: `https://docs.curseforge.com/rest-api/`
   - Use them to confirm: base URL, required headers, path/query params, request/response schemas, pagination mechanics, and error formats.

3. Implement or adjust the integration
   - Send `x-api-key` on all requests.
   - Handle pagination explicitly where required (for example, follow `index` and `pageSize` patterns until you have inspected the full result set).
   - Handle rate limiting as signaled by the API, consistent with the repo networking layer (for example, `X-Ratelimit-Remaining` and `X-Ratelimit-Reset`).
   - Fail with actionable errors when required fields are missing (for example, missing download URL or file hash).

4. Validate
   - Add or update tests where this repo has them.
   - If testing is not available for the change, do a focused smoke check (for example, validate JSON shape parsing against representative samples).
