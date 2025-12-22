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

Write as a conversation with a future developer. Keep each ADR to one or two pages. Large documents go unread; modular, bite-sized documents stay current.

## Location and Numbering

Store ADRs in `doc/arch/adr-NNN.md` with sequential numbering. Numbers are never reused. Superseded ADRs remain in place with updated status.

## ADR Structure

| Section          | Content                                                                                                                       |
|------------------|-------------------------------------------------------------------------------------------------------------------------------|
| **Title**        | Short noun phrase: `ADR NNN: [Decision Subject]`                                                                              |
| **Context**      | Value-neutral description of forces at play. Describe tensions, constraints, and factors without advocating for any solution. |
| **Decision**     | Active voice statement beginning with "We will..."                                                                            |
| **Status**       | `Proposed`, `Accepted`, `Deprecated`, or `Superseded` (with link to replacement)                                              |
| **Consequences** | All outcomes: positive, negative, and neutral effects                                                                         |

## Workflow

Use `@meza/adr-tools` via npx:

```bash
npx -y -p @meza/adr-tools -- adr list                           # List all ADRs
npx -y -p @meza/adr-tools -- adr new "Description"              # Create new ADR
npx -y -p @meza/adr-tools -- adr supersede [N] "Description"    # Supersede ADR N
npx -y -p @meza/adr-tools -- adr new -l "[N]:Type:Inverse" "X"  # Create linked ADR
npx -y -p @meza/adr-tools -- adr help                           # Show help
```

## Critical Rules

- **Decision changed?** Use `adr supersede`, never edit the original
- **Typo or clarification?** Edit is acceptable
- **Consequences become context** - Later ADRs reference earlier ones
- **Preserve the chain** - Superseded ADRs stay in place, linked to replacements
