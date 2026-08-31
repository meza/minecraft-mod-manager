---
name: modrinth-api
description: Modrinth API Gate - Use when implementing, updating, or debugging Modrinth API integration (endpoints, auth headers, rate limits, pagination, error handling) or when working with Modrinth OpenAPI docs/spec.
---

# Modrinth API

## When to Use

Use this skill when a task involves any of the following:
- Adding or modifying Modrinth API calls (any HTTP to `api.modrinth.com`).
- Updating request/response parsing for Modrinth resources (projects, versions, version files, search).
- Investigating Modrinth API errors (auth failures, rate limits, unexpected response shapes).
- Needing up-to-date endpoint definitions, parameters, or schemas via Modrinth docs or OpenAPI.

## Core Approach

1. Prefer authoritative sources: read the official docs and OpenAPI spec for up-to-date truth.
2. Keep repo behavior consistent: align with `docs/platform-apis.md` before changing behavior.
3. Be conservative under uncertainty: label assumptions, verify in docs/spec, and avoid inventing endpoints or fields.
4. Protect secrets: never log or persist API keys; treat `Authorization` values as sensitive.

## Workflow

1. Read local expectations first (repo-specific)
   - Read `docs/platform-apis.md` and confirm the intended Modrinth behaviors (headers, rate limiting, endpoint usage).

2. Acquire the current Modrinth API docs and OpenAPI spec
   - Docs: `https://docs.modrinth.com/api/`
   - OpenAPI: `https://docs.modrinth.com/openapi.yaml`
   - Use them to confirm: base URLs, auth scheme, required headers, path/query params, request/response schemas, and error formats.

3. Implement or adjust the integration
   - Match the spec's parameter encoding and response shape.
   - Handle rate limiting and retries in a way consistent with the repo's networking layer.
   - Fail with actionable errors when required fields are missing.

4. Validate
   - Add or update tests where this repo has them.
   - If testing is not available for the change, do a focused smoke check (for example, validate JSON shape parsing against representative samples).
