---
name: ticketing
description: Create, draft, read, or refine actionable project tickets and issues. Use when defining work, acceptance criteria, constraints, dependencies, verification, or change safety, and when project context must be obtained from Linear.
---

# Ticketing

A ticket must let an implementer deliver and a reviewer verify one coherent outcome without inventing product decisions.

## Get the context

Use the enabled Linear connector directly when a ticket is identified or the user asks to read, create, or update one. Do not discover credentials, read secret files, or route through another skill.

Distinguish drafting from external mutation. Draft in the response or requested file unless the user explicitly asks to create or update the Linear ticket.

Read applicable product, architecture, specification, and contributor documentation. Treat links as evidence only for the contracts their sources own. State whether each included link is required or optional and summarize any requirement the ticket depends on.

## Write the ticket

Use only the sections needed for the work, preserving these decisions:

- Intent: the problem, affected audience, and desired outcome.
- Impact: why the change matters now.
- Observable outcomes: acceptance criteria a reviewer can verify.
- Constraints and non-goals: boundaries the implementation must respect.
- Expected change: the affected capability and architectural direction without prescribing incidental implementation details.
- Assumptions and open questions: known uncertainty and who must resolve it.
- Dependencies and risks: prerequisites, coordination, and credible failure concerns.
- Verification: commands, scenarios, or observable evidence that prove completion.
- Change safety: rollout, blast-radius, recovery, or rollback intent when the change can affect users or shared infrastructure.

Keep the ticket focused on one coherent change. Split unrelated goals rather than hiding broad cleanup inside the work.

For work with cross-cutting, user-facing, security, operational, or rollout implications, read [the optional concern scan](references/concern-scan.md) and include only the concerns that materially apply. Do not copy the scan into every ticket.

## Refine an existing ticket

- Replace vague intentions with observable outcomes.
- Expose decisions an implementer would otherwise have to invent.
- Remove requirements unsupported by product or architecture evidence.
- Reconcile scope, assumptions, dependencies, and verification with current repository truth.
- Preserve useful existing context and avoid rewriting for style alone.

## Self-verification

Before delivering or mutating a ticket, verify that:

- the outcome, audience, impact, and acceptance evidence are clear;
- scope and non-goals form one coherent unit of work;
- architecture claims agree with their canonical project sources;
- links identify their purpose and do not hide requirements;
- applicable risks, dependencies, and change-safety needs are visible;
- unresolved product decisions remain explicit rather than guessed; and
- Linear was changed only when the user authorized that external mutation.
