# Clean Code

This skill helps an agent reason about code quality while designing, writing,
refactoring, reviewing, or explaining code, tests, APIs, modules, and
architecture. It treats clean code as a set of interacting engineering qualities,
not as a style checklist or a collection of universal prohibitions.

The skill requires the agent to:

- start from the actual decision, code, or failure under consideration;
- select only the concern areas that materially affect that context;
- read the relevant quality references before settling on a judgment;
- weigh correctness, safety, comprehension, operability, and change cost together;
- distinguish concrete evidence from assumptions and aesthetic preference; and
- recommend changes that fit the project's target architecture and established authority.

## Progressive catalog

The catalog contains 84 individual quality references organized under 9 concern
areas. `SKILL.md` provides enough context to choose a concern area without loading
the whole catalog. Each concern-area document then provides enough context to
choose the individual quality references that need deeper reading.

Every quality has one dedicated Markdown document inside its concern-area folder.
Each document combines the quality explanation with a small paired pseudocode
example showing what weak and strong practice look like. The installed skill is
self-contained and does not require another review skill or an external catalog
at runtime.

## Install with `npx skills`

Install the skill into the current project:

```console
npx skills add meza/skills --skill clean-code
```

Install it globally for Codex:

```console
npx skills add meza/skills --skill clean-code --agent codex --global
```

Install it globally for Claude Code:

```console
npx skills add meza/skills --skill clean-code --agent claude-code --global
```

After installation, ask your agent to design, review, refactor, or explain code
with the relevant clean-code qualities in mind. For example:

```text
Review [file or design description] for the clean-code qualities relevant to
[decision or change]. Explain the evidence, trade-offs, and recommended changes.
```
