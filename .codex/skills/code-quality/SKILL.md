---
name: code-quality
description: Framework defining what good code looks like across all dimensions. MUST USE when writing, reviewing, or reasoning about code quality. Consult this skill to evaluate code against quality standards, make trade-off decisions, or understand what qualities to prioritize. Covers readability, correctness, reliability, operability, efficiency, maintainability, safety, and consistency.
---

# Code Quality Framework

## Purpose

This describes the intrinsic qualities that distinguish good code. It provides a reasoning framework for recognizing these qualities when reading code and aspiring to them when writing code. These qualities apply across languages and platforms.

Use it when:

- Writing new code and need guidance on quality standards
- Reviewing code and need criteria for evaluation
- Making trade-off decisions between competing concerns
- Understanding what qualities to prioritize in a given context

## Quality Dimensions Overview

| Dimension | Core Qualities |
|-----------|----------------|
| **Readability** | Clear names, simple solutions, explicit dependencies and state, consistent and versioned APIs |
| **Correctness** | Predictable behavior, tested invariants, tested failures, real code exercised, deterministic and fast tests |
| **Reliability** | Explicit errors, boundary validation, proper resource management, timeouts, circuit breakers, idempotency, thread safety, data integrity |
| **Operability** | Structured telemetry, traceable requests, informative errors, validated configuration, graceful shutdown, health exposure |
| **Efficiency** | No needless waste, appropriate data structures, measured optimization |
| **Maintainability** | Small units, high cohesion, loose coupling, localized changes, visible debt, scoped refactoring |
| **Safety** | No secrets, sanitized input, minimal privilege, privacy respected, accessible and inclusive |
| **Consistency** | Follows project standards, automated enforcement |

These qualities reinforce each other. Clear code is easier to test. Tested code is safer to change. Well-structured code is easier to observe. Consistent code is easier to understand. Good code embodies all these qualities together.

## Using This Framework

When evaluating or writing code, consult the detailed quality definitions in [references/qualities.md](references/qualities.md).

Each dimension contains specific qualities with concrete guidance. Use them as a checklist when:

- Designing new components
- Reviewing pull requests
- Refactoring existing code
- Making architectural decisions

When trade-offs arise, use these qualities as the standard against which options are evaluated.
