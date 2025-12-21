---
name: documentation
description: >
  MUST USE when writing or reviewing documentation of any kind. Applies to README files,
  CONTRIBUTING guides, module docs, API documentation, and inline documentation. Covers
  audience analysis, documentation hierarchy, change safety, tone, and structure. Use when
  creating new documentation, updating existing docs, or evaluating documentation quality.
---

# Documentation Guidelines

Documentation transfers understanding so readers can act. Every piece serves one of three audiences.

## Audiences

| Audience | Needs | Focus | Priority |
|----------|-------|-------|----------|
| **Users** | What it does, how to use it | Clarity, examples, outcomes | Immediate utility |
| **Developers** | How it works, how to extend | Design intent, structural reasoning, trade-offs | Understanding why |
| **Maintainers** | How to preserve and evolve safely | Invariants, assumptions, predictability | Long-term stability |

Identify the primary audience before writing. Mixed-audience documents should use clear sections.

## Workflow

### Pre-pass (Discovery)

Before writing, read existing documentation to understand what exists.

- Identify gaps and contradictions
- Plan updates that link rather than duplicate
- Preserve content that already works

### Phase 1: User Documentation

Why it exists, installation, setup, quick start, usage examples.

**Artifacts:** README.md files with purpose, audience identification, and getting-started instructions.

### Phase 2: Developer Reference

Architecture, dependencies, integration points.

**Artifacts:** Technical sections in READMEs, dedicated docs/ folders for complex systems.

### Phase 3: Maintainer Notes

Trade-offs, known pitfalls, architectural decisions, testing expectations, release conventions.

**Artifacts:** CONTRIBUTING.md, ADRs for significant decisions.

## Documentation Hierarchy

Documentation follows project structure. Each level serves its scope.

| Level | Location | Purpose |
|-------|----------|---------|
| Top | Root README | Project purpose, audience, quick start |
| Mid | Module READMEs | Module intent and usage |
| Low | Package/class docs | Specific purpose and constraints |

**Link upward, do not duplicate.** Child documentation references parent context rather than repeating it.

## Change Safety

When modifying existing documentation:

1. **Assess necessity** - Leave clear content untouched
2. **Prefer augmentation** - Add rather than replace when possible
3. **Preserve tone** - Match the existing authorial voice
4. **Validate links** - Ensure cross-references remain valid
5. **Annotate intent** - Commit messages explain why changes were made

## Writing Principles

**Tone:** Friendly, confident, plainspoken. Write as a competent peer, not a manual or marketing copy.

**Structure:** Markdown headings express conceptual hierarchy. A reader should navigate from headings alone. Prefer headings over lists for conceptual relationships; reserve lists for true sequences or parallel items.

**Clarity:** Every paragraph answers an actual reader question. No hidden assumptions. Prerequisites named and linked on first use.

**Sentences:** Short and declarative. Answer the questions readers actually have.

**Examples:** Runnable, minimal, directly relevant. No hypothetical scenarios when concrete ones exist.

**Navigation:** Include cross-links and orienting context. Outbound links include one sentence of context explaining why they matter. End sections with clear next steps when appropriate.

**Formatting:** ASCII-clean text only. No smart quotes, em dashes, or non-standard markdown.

## General Correction Principles

When improving existing documentation:

- Replace mechanical lists with purpose-driven sections
- Convert "setup steps" into "capabilities" - focus on what becomes possible
- Write from the user's seat - address their questions, not your knowledge
- End with action - every section should leave the reader knowing what to do next

## Configuration Documentation

Configuration is a first-class documentation topic, not an appendix. When documenting configurable systems, include:

- Purpose and placement of configuration
- Quick start with minimal viable config
- Common scenarios with examples
- Option catalog with consistent fields

For comprehensive configuration documentation guidance, see [references/configuration-docs.md](references/configuration-docs.md).

## Validation

Before presenting documentation, verify completeness against the validation checklist. This ensures coverage of all audiences, proper structure, and adherence to quality standards.

See [references/validation-checklist.md](references/validation-checklist.md) for the complete checklist.

## References

| Reference | Purpose | When to Use |
|-----------|---------|-------------|
| [validation-checklist.md](references/validation-checklist.md) | Pre-delivery quality verification | Before presenting any documentation |
| [configuration-docs.md](references/configuration-docs.md) | Deep guidance on configuration documentation | When documenting configurable systems |
