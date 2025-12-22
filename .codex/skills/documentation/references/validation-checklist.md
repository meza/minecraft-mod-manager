# Validation checklist

Use this checklist before presenting documentation. Mark each item Pass or Fail. If any Fail remains, revise and re-run the checklist.

## Pre-pass: discovery and DRY

* Read existing docs in scope: README, CONTRIBUTING, ADRs, docs folder.
* Identify gaps, contradictions, and duplication.
* Plan updates that link to existing content instead of repeating it.
* Confirm that no section repeats information available elsewhere. If overlap exists, replace with a link and one-sentence summary.

## Audience coverage and modes

* User pass done: explains why, quick start, minimal working example.
* Developer pass done: concepts, architecture, integration points, links to ADRs.
* Maintainer pass done: trade-offs, constraints, testing and release expectations located in CONTRIBUTING.md.

## Hierarchy and placement

* Top-level README introduces purpose, audience, and quick start.
* Module-level README exists next to code it documents, explains local intent and usage.
* Low-level notes exist where needed to clarify intent and constraints; otherwise omitted.
* If content belongs higher or lower, move it or link to the correct level.

## Structure and headings

* Headings reflect logical questions: what, why, how, next.
* Heading depth used where needed (#, ##, ###, ####) to show hierarchy.
* Table of contents or heading outline is navigable on its own.
* Sections start with orienting context and end with a clear next step or closure.

## Tone and clarity

* Voice is friendly, confident, plainspoken. No marketing language.
* Every paragraph answers a real reader question.
* Short, declarative sentences; no hidden assumptions. Prerequisites are named and linked.

## Examples

* Each example is minimal, runnable, and directly supports the nearby text.
* New variables or placeholders are explained immediately.
* Overlong blocks are avoided; examples are split or simplified if needed.

## Navigation and cross-linking

* Internal links connect sibling modules and parent docs.
* Outbound links include one sentence of context explaining why they matter.
* Links are stable and verified. Anchors and paths are correct.

## Change safety rules applied

* Necessity assessed: only unclear, outdated, or missing parts were changed.
* Augmentation preferred to replacement; tone and structure preserved.
* Removed content is not referenced elsewhere.
* Commit message explains why the change was made, not just what changed.

## Formatting and style

* Avoid non-ASCII typography in prose (smart quotes, en-dashes), but don't "sanitize" UI output examples.
* Keep UX glyphs in code blocks / snapshot-like examples exactly as the app shows them (including arrows, etc.).
* Markdown only. Headings over bulleted prose where hierarchy exists.
* Lists used only for true sequences or unordered sets.
* Code fences for commands and config; backticks for inline literals.
* Consistent link syntax; readable in plain text and terminals.

## Accessibility and readability

* Headings convey structure for screen readers.
* Link text is descriptive, not "here."
* No giant images or code blocks that hinder scanning.

## Artifact expectations

* Root README present and useful for immediate start.
* Module READMEs present where needed, living next to their code.
* CONTRIBUTING.md present and current for setup, testing, branching, review, release.

## Content integrity

* Commands and snippets tested or clearly marked as illustrative if not.
* Version references are current or pinned with rationale.
* Terminology is consistent across files.

## Final readiness

* Skim the headings only. Does the story make sense end to end?
* Open the file on a narrow terminal. Is it still readable?
* Run a quick link check and ASCII check.
* Confirm that the reader can stop at natural points without fear of missing steps.

## General correction principles

### Replace mechanical lists with purpose-driven sections

If the content looks like a task list, reframe it as a story of intent and outcome.
Use subheadings to encode relationships and provide anchors for reference.

### Convert "setup steps" into "capabilities"

Explain what a feature enables, not what code executes.

### Write from the user's seat

Imagine you're the person opening this file because something isn't working.
What do they need to understand to solve their problem quickly?

### End with action

Each section should close with either:

- something the reader can do next, or
- confirmation that they're done.
