# Roadmap to 3.0.0

This document defines the work required to ship version 3.0.0 of minecraft-mod-manager, the Go rewrite.

## Release Definition

Version 3.0.0 represents the complete Go port of the CLI, replacing the Node.js implementation. The release is defined by the GitHub 3.0.0 milestone, which contains four items:

1. **#629 - Porting to Go** - The umbrella issue for the rewrite
2. **#452 - More sensible errors** - Improve error message clarity
3. **#630 - Cache the minecraft manifest file** - Offline resilience
4. **#1030 - Network proxy function** - Proxy support for restricted networks

## Current State

The Go port is substantially complete. Most commands are implemented; remaining work is tracked under the feature parity epic (mmm-1).

The codebase underwent an independent audit on 2025-12-19. All audit findings are tracked under epic mmm-63, which must be closed before release.

## Release Gates

The following conditions must be met before 3.0.0 can ship:

### Gate 1: Audit Resolution (mmm-63)

The 2025-12-19 audit epic (mmm-63) must be closed before release. This includes all findings resolved or explicitly accepted.

### Gate 2: Feature Parity (mmm-1)

All commands documented in the Node CLI must function equivalently in Go. The feature parity epic (mmm-1) being closed is the gate. This includes all command implementations and telemetry/update notification wiring.

### Gate 3: Release Automation Functional

The release pipeline (mmm-62) must be capable of producing and publishing:

- dist/*-windows.zip (contains mmm.exe)
- dist/*-macos.zip (contains mmm)
- dist/*-linux.zip (contains mmm)

This requires:
- CI workflow running on PRs and pushes
- Release workflow running semantic-release on release branches
- Secrets properly injected without log leakage

### Gate 4: Milestone Features

The three non-port issues on the 3.0.0 milestone require explicit disposition:

#### #452 - More sensible errors

**Requirement**: When a mod cannot be found because the game version is not yet supported, the error must explicitly state that the version is not available rather than showing a confusing or generic error.

**Success criteria**:
- Error messages for version incompatibility explicitly mention the version constraint
- No generic "not found" or timeout errors when the root cause is version mismatch
- Related error (#349 connect timeout) is also addressed if still present

**Disposition**: Must be addressed in 3.0.0 scope.

#### #630 - Cache the minecraft manifest file

**Requirement**: Cache the Minecraft version manifest locally so the CLI can function when the manifest cannot be downloaded.

**Success criteria**:
- Manifest is cached in a system-appropriate location (os.TempDir or equivalent)
- When manifest download fails, cached version is used if available
- Cache has appropriate TTL or staleness handling
- Failure mode when no cache exists is clear to the user

**Disposition**: Must be addressed in 3.0.0 scope. This aligns with audit finding mmm-63.20 (Minecraft manifest cache is global, has no TTL, and is not concurrency-safe).

#### #1030 - Network proxy function

**Requirement**: Support network proxy configuration for environments that require proxy access.

**Success criteria**:
- Standard proxy environment variables (HTTP_PROXY, HTTPS_PROXY, NO_PROXY) are respected
- Proxy configuration works for all outbound HTTP requests (API calls, downloads)
- Documentation explains proxy configuration

**Disposition**: Requires design decision. Either:
- (A) Include in 3.0.0 scope, or
- (B) Defer to 3.1.0 with explicit milestone reassignment

This is a new feature request (November 2025) and may represent scope creep for an already substantial release.

## Work Phases

### Phase 1: Resolve Audit

**Objective**: Close the audit epic (mmm-63).

All findings under mmm-63 must be resolved or explicitly accepted.

**Exit criteria**: Epic mmm-63 is closed.

### Phase 2: Complete Feature Parity

**Objective**: Close the feature parity epic (mmm-1).

All remaining work items under mmm-1 must be completed. This includes command implementations (change, prune) and supporting infrastructure (telemetry, update notifications).

**Exit criteria**: Epic mmm-1 is closed.

### Phase 3: Address Milestone Features

**Objective**: Implement or explicitly defer the three non-port milestone issues.

| Work Item | Priority | Description |
|-----------|----------|-------------|
| #452 | TBD | Error message improvements |
| #630 | TBD | Manifest caching |
| #1030 | TBD | Proxy support (requires disposition decision) |

**Exit criteria**: Each issue is either closed or explicitly moved to a future milestone with documented rationale.

### Phase 4: Release Automation

**Objective**: Release pipeline can ship artifacts.

| Work Item | Priority | Description |
|-----------|----------|-------------|
| mmm-62 | P1 | GitHub Actions release workflow |

**Exit criteria**: Dry-run release produces expected artifacts; secrets are properly handled.

### Phase 5: Release Validation

**Objective**: Validate release readiness in the actual release context.

With the audit epic (mmm-63) closed and release automation functional, perform final validation that all gates are satisfied and the release pipeline produces correct artifacts.

**Exit criteria**: Dry-run release succeeds; all gates verified as met.

## Sequencing

```
Phase 1 (Audit Resolution)
    |
    v
Phase 2 (Feature Parity) -----> Phase 4 (Release Automation)
    |                                      |
    v                                      v
Phase 3 (Milestone Features) -----> Phase 5 (Validation)
                                           |
                                           v
                                      3.0.0 Release
```

Phase 1 must complete before Phase 2. Phase 4 can proceed in parallel. Phase 5 depends on all prior phases completing.

## Open Questions

The following require stakeholder input before Phase 3 can complete:

1. **Proxy support scope**: Should #1030 be included in 3.0.0 or deferred?
   - Including it expands scope but addresses a user-reported need
   - Deferring it keeps release focused but leaves a feature request unaddressed

2. **Error message scope (#452)**: What specific error scenarios must be addressed?
   - The issue mentions version incompatibility and connect timeout
   - A comprehensive error audit may reveal additional cases

## Tracking

This roadmap tracks the following sources:

- **GitHub milestone**: https://github.com/meza/minecraft-mod-manager/issues?q=milestone%3A3.0.0
- **Feature parity epic**: mmm-1 (Go CLI reaches Node feature parity)
- **Audit epic**: mmm-63 (Audit: Independent report 2025-12-19)
- **Release automation**: mmm-62 (Set up GitHub Actions release workflow)

Work items are tracked in the beads issue tracker. Use `bd list --status open` to see current state.

## Summary

3.0.0 requires:

1. Audit epic closed (mmm-63)
2. Feature parity epic closed (mmm-1)
3. Release automation functional (mmm-62)
4. Milestone features addressed (#452, #630, #1030 with proxy disposition decision)

The release represents a major rewrite. The audit identified substantial technical debt that will require ongoing attention beyond 3.0.0. The remaining work is tractable once the epic-level blockers are resolved.
