---
name: adr
description: >
  MUST USE when working with architectural decision records (ADRs).
  Triggers: creating new ADRs, superseding existing ADRs, listing/reviewing ADRs,
  linking related ADRs, or discussing architecture decisions affecting structure,
  dependencies, interfaces, or construction techniques.
---

# Architecture Decision Records

ADRs document architecturally significant decisions: those affecting structure, non-functional characteristics, dependencies, interfaces, or construction techniques.

## Philosophy

Agile projects still need documentation; they just avoid valueless, unmaintained documentation. Large documents go unread and stale quickly. Small, modular documents can stay current.

Write each ADR as a conversation with a future developer. Keep each ADR to one or two pages.

The goal is to preserve decision motivation so future developers do not:

1. Blindly accept a past decision without understanding whether it is still valid, or
2. Blindly change a past decision without understanding what it protected (including non-functional requirements).

Each ADR balances forces (similar in spirit to an Alexandrian pattern). Forces can appear in multiple ADRs; the single decision is the central piece.

One ADR describes one significant decision for a specific project. ADRs make motivations and trade-offs visible to current and future team members and stakeholders.

## Location and Numbering

Store ADRs in `doc/arch/adr-NNN.md` with sequential, monotonic numbering. Numbers are never reused.

If a decision is reversed, keep the old ADR but mark it as superseded (with a link to its replacement).

Write ADRs as plain text using a lightweight markup format (Markdown or Textile; Markdown preferred).

## ADR Structure

| Section          | Content                                                                                                                       |
|------------------|-------------------------------------------------------------------------------------------------------------------------------|
| **Title**        | Short noun phrase: `ADR NNN: [Decision Subject]` (e.g., `ADR 001: Deployment on Ruby on Rails 3.0.10`, `ADR 009: LDAP for Multitenant Integration`) |
| **Context**      | Value-neutral description of forces at play (technological, political, social, project-local). Call out tensions and constraints without advocating for any solution. |
| **Decision**     | Active voice statement beginning with "We will..."                                                                            |
| **Status**       | `Proposed`, `Accepted`, `Deprecated`, or `Superseded` (with link to replacement)                                              |
| **Consequences** | All outcomes: positive, negative, and neutral effects                                                                         |

## Writing Style

- Use full sentences organized into paragraphs.
- Use bullets only for visual grouping, not as an excuse for sentence fragments.
- Avoid bullet-only ADRs. (Bullets kill people.)

## Workflow

The project uses `@meza/adr-tools` to manage ADRs. Read the adr-tools documentation before making structural changes to the ADR set: `https://github.com/meza/adr-tools`.

Always run adr-tools via npx; do not install it globally:

```bash
npx -y -p @meza/adr-tools -- adr list                           # List all ADRs
npx -y -p @meza/adr-tools -- adr new "Description"              # Create new ADR
npx -y -p @meza/adr-tools -- adr supersede [N] "Description"    # Supersede ADR N
npx -y -p @meza/adr-tools -- adr new -l "[N]:Forward:Inverse" "X"  # Create linked ADR
npx -y -p @meza/adr-tools -- adr help                           # Show help
```

Example supersede:

```bash
npx -y -p @meza/adr-tools -- adr supersede 3 "The new decision that supersedes ADR 003"
```

### Linking ADRs

Use linking when a new decision relates to a previous ADR but does not supersede it.

Link format: `"[N]:<ForwardLabel>:<InverseLabel>"` where:

- `N` is the existing ADR number being linked to.
- `ForwardLabel` is the label that will appear on the new ADR pointing to the old ADR.
- `InverseLabel` is the label that will appear on the old ADR pointing to the new ADR.

The tool sets numbering and navigational links on both ADRs.

Example:

```bash
npx -y -p @meza/adr-tools -- adr new -l "3:Amends:Amended by" "Use jest only for pact testing"
```

## Critical Rules

- **Decision changed?** Use `adr supersede`, never edit the original
- **Typo or clarification?** Edit is acceptable
- **Consequences become context** - Later ADRs reference earlier ones
- **Preserve the chain** - Superseded ADRs stay in place, linked to replacements
