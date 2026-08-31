---
name: modrinth-api
description: Modrinth API integration gate. Use when implementing, updating, debugging, or reviewing Modrinth endpoints, authentication, rate limiting, errors, request or response models, or official API and OpenAPI documentation.
---

# Modrinth API

## Establish the contracts

Read these repository documents before assessing or changing the integration:

1. `internal/platform/README.md` for the shared provider boundary, normalized result, selection behavior, clients, and expected errors.
2. `internal/modrinth/README.md` for Modrinth package ownership, public APIs, headers, selection, and provider-specific errors.

Consult the current official Modrinth API documentation at `https://docs.modrinth.com/api/` and OpenAPI specification at `https://docs.modrinth.com/openapi.yaml` for the external wire contract. Confirm the base URL, authentication, required user agent, endpoint, parameter encoding, response schema, error shape, and published rate-limit behavior relevant to the task.

Local documentation owns project architecture and behavior. Official documentation owns the current external API contract. If they conflict, report the conflict and establish which project behavior must change rather than silently choosing one.

## Work within the provider boundary

- Keep shared orchestration and normalized results in `internal/platform`.
- Keep Modrinth transport, models, selection, and provider-specific error types in `internal/modrinth`. Preserve shared project-level error types in `internal/globalerrors` as documented by the provider package.
- Reuse the injected HTTP `Doer` and shared rate limiter.
- Preserve the authorization and user-agent handling owned by the Modrinth client. Never log, persist, or expose credentials.
- Map expected failures consistently with the documented platform and provider contracts.
- Do not add endpoints, fields, headers, retry rules, or rate-limit semantics that are not supported by authoritative evidence.

For a review, trace each changed request and response through the provider package and every applicable consumer. Trace normalized fetch behavior through `internal/platform`; review provider-only APIs at their documented public or package boundary. Report findings without modifying files unless changes are authorized.

## Validate

Use local `httptest` coverage for request construction, headers, encoding, decoding, selection, and error mapping as applicable. Follow `CONTRIBUTING.md` and use its required `make` targets rather than calling Go test or build commands directly.

## Self-verification

Before completing the task, verify that:

- both local architecture documents were read;
- external claims were checked against current official Modrinth documentation;
- provider-specific behavior remains inside `internal/modrinth`;
- the shared `internal/platform` contract remains coherent;
- credentials cannot appear in output or persisted artifacts; and
- validation covers the changed or reviewed request, response, and failure paths.
