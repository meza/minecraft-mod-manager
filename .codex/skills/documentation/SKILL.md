---
name: documentation
description: >
  MUST USE when writing or reviewing documentation of any kind. Applies to README files,
  CONTRIBUTING guides, module docs, API documentation, and inline documentation. Covers
  audience analysis, documentation hierarchy, change safety, tone, and structure. Use when
  creating new documentation, updating existing docs, or evaluating documentation quality.
---

# Documentation Guidelines

Documentation transfers understanding so readers can act. You write to transfer understanding quickly and accurately, giving readers enough context to act (use, contribute, or maintain).

## Purpose of documentation

Your purpose is to help people understand why something exists, what it enables, and how to use or maintain it effectively, whether that "something" is an entire project, a submodule, or a small internal library.
You write documentation that grows in depth and clarity the closer it gets to the code.
You are not here to describe implementation. You are here to explain purpose, design, and practical use.

Documentation exists to help readers make decisions.
Your job is to guide real humans through complexity. You are not trying to impress experts.

## Audiences

You write for three audiences that overlap but differ in focus.
Identify the primary audience before writing. Mixed-audience documents should use clear sections.

### Users

People who want to know what it does and how to use it. Focus on clarity, examples, and successful outcomes. They do not need to know how it works internally; they need to succeed using it.

### Developers (if the project is a library, framework, or tool)

People who want to understand how it works or extends. Explain design intent and the reasoning behind structural choices. Developers care about constraints and trade-offs: the why behind the how.

### Maintainers

People who need to preserve and evolve it safely. They depend on predictability and clear documentation of invariants, assumptions, and dependencies. Help them avoid regression and drift.

Your writing scales naturally: it should be equally useful at the project root or deep within a feature directory. Your perspective changes, but your goal stays the same: clarify intent.

## Process

Each phase builds on the last. Start by understanding what exists, then move from outward clarity (users) to inward reasoning (developers) to long-term context (maintainers). This layered approach prevents you from jumping straight into internal explanations before the reader has a reason to care.

### Pre-pass - discovery and DRY check

Before writing or revising documentation, always perform a discovery pass.

1. Read all existing documentation in this scope, including `README.md`, `CONTRIBUTING.md`, ADRs, and other documentation materials.
2. Understand the intent, tone, and structure so your updates remain consistent.
3. Identify gaps, contradictions, or duplication.
4. Plan your update to add clarity where missing and link where repetition exists.

Never rewrite what is already clear. Improve, connect, and extend instead.

### Phase 1 - user guide

Write for someone discovering this project or module for the first time.
Begin with why: the problem it solves, its audience, and its value. Avoid parroting implementation details from the code.
Give yourself time to digest the purpose and value of the software; the big picture is essential before diving into usage.
Show how to use it: installation, setup, quick examples.
Keep it clear, friendly, and actionable.
Assume this document might be the first file someone opens.

Primary artifacts include:

* `README.md` at the project root (broad overview)
* `README.md` inside modules, packages, or feature directories (focused guides)
  Each README lives directly beside what it documents.

### Phase 2 - developer reference

Write for people reading or extending the code as a library, framework, or tool.
Explain how it works conceptually, not line by line.
Cover architecture, dependencies, and integration points.
Provide short, runnable examples and link to ADRs or API definitions.
Use diagrams or summaries to show how parts connect.

Primary artifacts include:

* Technical sections of each `README.md` following the "Usage" section
* Optional `docs/` or `architecture/` folders for detailed internal explanations

### Phase 3 - maintainer notes

Write for long-term stewards of the project.
Explain why it is designed this way: trade-offs, constraints, invariants.
Point to ADRs and decision history.
Note known pitfalls, testing expectations, and release or branching conventions.
Keep it candid and pragmatic.

Primary artifact:

* `CONTRIBUTING.md`, the single, authoritative place for contribution and maintenance guidance.
  Include setup, testing, branching, review, and release expectations.

## Documentation Hierarchy

Readers enter at unpredictable depths. Some start at the root, others open a module folder directly. Clear hierarchy ensures they can always navigate up or down without confusion.
Documentation follows project structure. Each level serves its scope.

| Level | Location | Purpose |
|-------|----------|---------|
| Top | Root README | Project purpose, audience, quick start |
| Mid | Module READMEs | Module intent and usage |
| Low | Package/class docs | Specific purpose and constraints |

When information overlaps, link upward rather than duplicate.
When higher-level context is missing, add just enough here for local understanding.

## Change safety rules

Documentation has continuity. Rewriting too much breaks readers' mental links, diffs, and version history. Favor surgical precision over stylistic overhaul.

### Assess necessity

If content is already clear and accurate, leave it untouched.
If information is outdated or missing, update only the affected parts.

### Prefer augmentation over replacement

Add context or links instead of rewriting entire sections.
Extend examples rather than reformatting them unless clarity demands it.

### Preserve authorial tone

Match the existing writing style and structure.
When major stylistic changes are required, justify them with consistency or clarity concerns.

### Validate linkage

Ensure all references, anchors, and file paths remain valid.
If removing text, verify that nothing else depends on it.

### Annotate intent

When you revise, leave clear commit messages or inline comments explaining why the change was made, not just what changed.

These rules exist to protect both clarity and authorship.
Your edits should make documentation more consistent, not more "yours".

## General Correction Principles

When improving existing documentation:

- Replace mechanical lists with purpose-driven sections
- Convert "setup steps" into "capabilities" - focus on what becomes possible
- Write from the user's seat - address their questions, not your knowledge
- End with action - every section should leave the reader knowing what to do next

## Response principles

### Tone

Be friendly, confident, and plainspoken.
Your voice should sound like a competent peer explaining a tool they know well, not a manual or marketing copy.
Avoid exaggerated enthusiasm or empty phrases.

### Structure

Use Markdown headings to express relationships and meaning, not just to break up text visually.
The document should be navigable from its table of contents alone; a reader should see the structure and understand the logical flow without reading every word.
Headings form a semantic map that aids accessibility, screen readers, and static site generators.
Depth of structure is better than a wall of paragraphs or endless bullet points.
Every new idea that answers a "what", "why", or "how" question deserves its own heading.

### Hierarchical formatting

Use lists only when a true sequence or unordered set is being conveyed, for example, installation steps or enumerated configuration options.
When a list actually represents a conceptual hierarchy, replace it with nested headings and short explanations.
Lists compress thought, while headings reveal relationships.

### Examples

Examples are your proof that the text is not theoretical.
Examples should be runnable, minimal, and directly relevant to the section they sit in.
Avoid giant code blocks; show the smallest possible illustration that still teaches.
If an example introduces new variables or context, explain them immediately afterward.

### Clarity

Every paragraph must answer an actual reader question.
Favor short, declarative sentences that remove ambiguity.
If a concept requires background knowledge, name that dependency explicitly and provide a link.
Do not assume shared context; readers may arrive from search results or error messages.

### Navigation

Readers rarely consume documentation linearly. They skim, search, and jump, so help them do that gracefully.
Start sections with orienting context and end them with clear next steps or closure ("You can stop here if...").
Include cross-links between related parts of the same repo so navigation feels continuous rather than fragmented.
Where possible, match the reader's current scope: if they are in a module README, link to other nearby modules before sending them up to higher-level docs.

### Cross-linking

Use internal links to connect ideas horizontally (between sibling modules) and vertically (between levels of abstraction).
Never duplicate large sections of text when a link would suffice.
When linking outward, provide one sentence of context explaining why the destination matters ("Why should I click this?").

### Style

Use standard Markdown syntax only.
Text must be ASCII-clean: never use curly quotes, smart quotes, em dashes, typographic ellipses, or non-breaking spaces.
Plain text should render correctly in any environment, from terminals to static sites.
Avoid emojis or decorative symbols; visual tone comes from structure, not ornament.
Use code fences for commands and configuration snippets, and backticks for inline literals.
Prefer consistent link formatting and headings that read as natural language.
Keep sentences short enough to scan comfortably in monospaced fonts.
The goal is legibility across contexts, not aesthetic flourish.

## Scope

Documentation should be broad enough to teach and narrow enough to stay truthful. What you cover must remain accurate as the code evolves. Avoid speculative or forward-looking statements unless explicitly noted.

### Local

Each `README.md` lives next to its code and explains it in context.

### Layered

Each level (root, module, utility) builds on the one above it.

### Complete

Root README introduces; sub-READMEs explain; `CONTRIBUTING.md` sustains.

### DRY

Never repeat what is already written.
Always read before you write.

### Safe

Never overwrite clarity.
Modify only when improvement is clear and necessary.

### Focused

Avoid speculation or redundancy.
Link across instead.

You always move through four steps: Discover -> User -> Developer -> Maintainer, producing or updating the relevant artifacts.
Your goal is that every reader, from the casual user to the deep maintainer, can say:

"I understand what this does, why it exists, how to use it, and how to keep it healthy."

## Configuration documentation

Configuration is where users succeed or give up. Document it as a first-class topic, not an appendix.

See [references/configuration-docs.md](references/configuration-docs.md) for the full guide.

## Validation checklist

Before presenting documentation, verify completeness against the validation checklist.
See [references/validation-checklist.md](references/validation-checklist.md) for the full checklist.

## References

| Reference | Purpose | When to Use |
|-----------|---------|-------------|
| [validation-checklist.md](references/validation-checklist.md) | Pre-delivery quality verification | Before presenting any documentation |
| [configuration-docs.md](references/configuration-docs.md) | Deep guidance on configuration documentation | When documenting configurable systems |
