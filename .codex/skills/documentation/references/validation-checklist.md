# Documentation Validation Checklist

Use this checklist before presenting documentation. Each category must pass for documentation to be considered complete.

## Pre-pass and DRY Checks

- [ ] Read all existing documentation in the affected scope
- [ ] Identified gaps and contradictions
- [ ] Planned updates that link rather than duplicate
- [ ] Preserved content that already works
- [ ] No redundant information introduced

## Audience Coverage

- [ ] Primary audience identified and stated
- [ ] User needs addressed: purpose, quick start, usage examples
- [ ] Developer needs addressed: architecture, dependencies, integration
- [ ] Maintainer needs addressed: trade-offs, pitfalls, testing expectations
- [ ] Mixed-audience documents use clear section separation

## Hierarchy and Placement

- [ ] Documentation lives at the correct level (root, module, package)
- [ ] Links upward to parent context rather than duplicating
- [ ] Siblings cross-reference where relevant
- [ ] No orphaned documentation without discoverable paths

## Structure and Headings

- [ ] Headings express conceptual hierarchy, not arbitrary breaks
- [ ] A reader can navigate from headings alone
- [ ] Prefer headings over lists for conceptual relationships
- [ ] Lists reserved for true sequences or parallel items
- [ ] Logical flow from purpose through details to action

## Tone and Clarity

- [ ] Friendly, confident, plainspoken voice
- [ ] Sounds like a competent peer, not a manual or marketing
- [ ] Short, declarative sentences
- [ ] Every paragraph answers a real reader question
- [ ] No hidden assumptions - prerequisites named and linked
- [ ] Technical terms defined or linked on first use

## Examples

- [ ] Examples are runnable and minimal
- [ ] Directly relevant to the documented feature
- [ ] Concrete scenarios preferred over hypothetical ones
- [ ] Quick start example appears early
- [ ] Complex scenarios documented with progression

## Navigation and Cross-linking

- [ ] Cross-links to related content included
- [ ] Outbound links include one sentence of context explaining relevance
- [ ] Horizontal links (sibling modules) present where useful
- [ ] Vertical links (abstraction levels) connect parent and child docs
- [ ] Sections end with clear next steps when appropriate

## Change Safety Applied

- [ ] Assessed necessity before modifying existing content
- [ ] Augmented rather than replaced where possible
- [ ] Preserved existing authorial voice
- [ ] Commit message explains why changes were made

## Formatting and Style

- [ ] ASCII-clean text only (no smart quotes, em dashes, special characters)
- [ ] Standard markdown syntax
- [ ] Consistent heading levels
- [ ] Code blocks specify language for syntax highlighting
- [ ] Tables used appropriately and render correctly

## Accessibility and Readability

- [ ] Renders correctly in narrow terminal (80 characters)
- [ ] Headings form scannable outline
- [ ] Important information not buried in paragraphs
- [ ] Alt text or descriptions for any images
- [ ] No reliance on color alone for meaning

## Expected Artifacts

- [ ] README.md present at appropriate levels
- [ ] CONTRIBUTING.md exists for projects accepting contributions
- [ ] ADRs created for significant architectural decisions
- [ ] Module-level documentation where complexity warrants

## Content Integrity

- [ ] No placeholder text remaining
- [ ] Version numbers and paths are accurate
- [ ] Command examples tested and working
- [ ] API references match current implementation

## Final Readiness

- [ ] Skim all headings - narrative makes sense
- [ ] Test rendering in narrow terminal
- [ ] Verify all internal and external links
- [ ] Spell check completed
- [ ] Review as if encountering for the first time
