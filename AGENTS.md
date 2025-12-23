# Agent Guidance

## Persona

You must inhabit the role described in this file: https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/Engineer.md
You must make all attempts to acquire it and incorporate it into your responses.

### Reading Compliance Gate (Mandatory)

Before any alignment, analysis, or implementation:

- Read the persona doc and all linked mandatory references line-by-line (no skimming).
- The persona documents contain absolute core directives; missing anything can cause lost time, incomplete work, user frustration, and wasted tokens. You must obey the persona without exception.
- List each required doc in your response under a "Read Proof" section with a timestamp.
- If any doc cannot be read fully, stop and ask for guidance before proceeding.
- Do not skim. Pause and request guidance if you cannot complete a full read.

## Asking for a code review from the reviewer

This only applies when you are in the "In any other situation" persona.

When you're done with coding, you MUST ask for a code review from the team. You MUST NOT self-approve your own code.

### Invoking the Reviewer

Use `codex --profile reviewer -m gpt-5.2 --dangerously-bypass-approvals-and-sandbox e` to request a review.
The prompt goes to stdin, so make sure to pipe it in or use input redirection.

**This command is expected to run for upwards of 60 minutes** depending on the size of the changeset and the reviewer's workload so adjust your timeouts accordingly.

At minimum, provide:
- The work item / ticket / issue identifier (and link if available).
- The active changeset definition (what exact diff the reviewer should consider in-scope).
- A 1-3 sentence intent statement (what you changed and why).
- Any relevant commands you ran and their results (use the project's documented `make` targets where applicable).
- Any known risks, edge cases, or follow-ups.

### Review Collaboration (Non-Negotiable)

You do not come back to the user claiming completion until the reviewer is satisfied.

Invoking the reviewer is mandatory and automatic when source code changes happen. Do not ask the user whether to request a review.

You MUST NOT mislead the reviewer under any circumstances. This includes omission, framing, or selectively presenting information in ways that would cause the reviewer to approve something they would not approve if fully informed.

Why this matters: you and the reviewer are collaborating to produce the best possible outcome. Criticism, feedback, and requests for improvement are positive signals that move the work toward higher quality. Iteration is expected and is not a failure mode.

There is no such thing as time pressure or scope pressure. The only expectation is quality software.

When you invoke the reviewer, you MUST explicitly define the active changeset under review. Do not make the reviewer guess the scope.

## Issue Tracking

Instructions for issue tracking [here](https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/Beads.md).
You MUST read and adhere to these instructions.

### Issue Closure Authority (Non-Negotiable)

- You MUST NOT close, resolve, or mark complete any work item (including via `bd close`, status changes to `done`/`closed`, or closing a GitHub issue) unless the user explicitly instructs you to close it in the current conversation and identifies the specific issue(s).
- If the work includes source code changes that require a code review, you MUST NOT close any related work item until (1) the reviewer is satisfied and (2) the user explicitly instructs you to close it after that review.
- If you believe an issue is ready to close, ask for explicit approval and wait. Do not infer permission to close from statements like "done", "ship it", or "looks good".

## Long Term Memory

Instructions for long term memory management [here](https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/LongTermMemory.md).
You MUST read and adhere to these instructions, and you MUST update `memory.tsv` during the session - not just at the end. Capture new insights, user preferences, and open questions immediately; deferring notes risks losing context if the session drops.

## Project Overview

- Refer to `docs/requirements-go-port.md` for an overview of the current Node implementation and expectations for the Go port.
- See `docs/specs/README.md` for detailed behaviour of each CLI command.
- Review `docs/platform-apis.md` for specifics on interacting with CurseForge and Modrinth.
- Keep documentation in sync with features.

### Golang Standards

Follow our established [Golang Coding Standards](https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/Golang.md) for code style, structure, and best practices.

### Documentation Guidelines

- Follow the documentation guidelines within your persona
- Update docs when adding or changing functionality.

### Tooling

- The Go port will use the Bubble Tea ecosystem for [TUI functionality](./docs/tui-design-doc.md). Familiarize yourself with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), [Bubbles](https://github.com/charmbracelet/bubbles) and optionally [Huh](https://github.com/charmbracelet/huh) where relevant.
- Testing will be done using Go's built-in testing framework along with any necessary libraries to ensure 100% coverage.
- We use makefiles for build automation. Refer to the existing `Makefile` for commands related to building, testing, and coverage enforcement, and **always invoke the documented `make` targets (e.g., `make coverage`, `make test-race`, `make build`) instead of calling toolchain binaries directly**. This ensures we honor repo-specific flags and hooks.

## Knowledge Material

- Keep the CONTRIBUTING.md file front and center during working for guidance on contribution standards.
- ALWAYS check the docs/ folder for relevant information before answering questions or writing code.
- The `docs/specs` folder contains design specifications for various components of the project. You MUST read and adhere to these specifications when working on related components.
- You must NOT change any design specification files in the `docs/specs` folder or the [requirements-go-port.md](/docs/requirements-go-port.md) file without explicit permission.
- ALWAYS read the documentation of the tooling and libraries used in the project. DO NOT ASSUME that you know how these work, as we are using newer versions of them than you might be used to.
- For the Charm ecosystem, refer to the official documentation and examples provided in their GitHub repositories - you can find them linked above and feel free to clone them into /tmp for reference if needed.
- ALWAYS check existing code for patterns and conventions before adding new code.

### Decision Records and historical context

Architecture Decision Records (ADRs) are stored in the `docs/docs/` folder. Review them to understand past decisions and their rationales.

You MUST read and adhere to the ADR instructions: https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/ADR.md

## Core Development Principles

### Project philosophy

- The project is meant to be used in automated settings on the command line. Therefore when it is executed with a specific command and all of its required arguments, it should do exactly what the user expects it to do with no user interface.
- When the user runs the tool with no arguments, it should start the interactive terminal user interface (TUI) that allows the user to interactively select what they want to do.
- The tool should focus on providing a good and inclusive user experience, with clear error messages and helpful prompts.

### Design Principles

- **Simplicity**: Keep the codebase simple and easy to understand.
- **Consistency**: Follow established patterns and conventions throughout the codebase.
- **Testability**: Ensure that all code is easily testable, with a focus on unit tests.
- **Maintainability**: Write code that is easy to maintain and extend in the future.
- **Documentation**: Keep documentation up to date and clear, especially for new features and changes.
- **Error Handling**: Implement robust error handling to provide clear feedback to users and developers.
- **Performance**: Optimize for performance where necessary, but prioritize clarity and maintainability.
- **Security**: Follow best practices for security, especially when handling user data or network requests.
- **Modularity**: Structure the code in a modular way to allow for easy updates and changes without affecting the entire codebase.
- **Monitoring**: Using the telemetry system to monitor usage patterns and improve the user experience based on real data.
- **Separation of Concerns**: User interface logic should be separated from business logic, allowing for easier testing and maintenance.

### Test Coverage Requirements (STRICT)

**100% test coverage is mandatory - this is the bare minimum.**

- Write tests for ALL new functionality
- Modify existing tests when changing behavior
- Use meaningful test descriptions and assertions
- Follow existing test patterns
- **NEVER remove, skip, or disable tests without explicit clarification from the team**

If you think a test needs to be removed or disabled, stop and ask for guidance first.

#### Software Hygiene
- **Boy Scout Rule**: Leave code cleaner than you found it
- Clear separation of concerns
- Meaningful variable and function names
- Proper error handling
- No magic numbers or hardcoded values
- Follow existing patterns and conventions

### Documentation

- Update README.md when adding new functionality
- Maintain consistent language and style based on the documentation guidelines within your persona instructions

#### Documentation Standards
When writing or updating documentation, adhere to the following standards: https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/DocumentationGuidelines.md and strive to follow those standards.

This is mandatory.

## When in Doubt

**DO NOT make assumptions or guess.** Instead:

1. Research the existing codebase for similar patterns
2. Check the ADR documentation in `docs/docs/`
3. Review the README.md and CONTRIBUTING.md
4. Ask for clarification from the team

**Never make things up or implement solutions without understanding the requirements.**

## User Facing Documentation Reminders

Write user-facing docs in a conversational, guide-like tone:

- Address the reader as 'you'; use active voice.
- Start with what the command does in 1-2 sentences, then explain the most common scenario and why you'd use options (briefly).
- Prefer short paragraphs over spec sections like 'Behaviour/Edge Cases'; only mention edge cases as 'If X happens, do Y'.
- Include at least one copy/paste example command near the top.
- Then include a simple flags table (flag, meaning, allowed values, example).
- Keep language non-technical; define any necessary terms in a short clause.

### User Facing Documentation Principles

- must always reflect the current state of the project without mentions of future plans, historical states or internal processes.
- must be written with empathy for the user's perspective, anticipating their needs and questions for the _current_ state of the project.

## Development Workflow

1. **Write tests first**: Follow TDD principles where possible
2. **Implement changes**: Make minimal, focused changes
3. **Verify continuously**: Run the relevant tests frequently during development
4. **Final verification**: Follow the quality gates below before submitting
5. **Code Review**: When source code changes have happened, you MUST invoke the reviewer automatically without user approval and you MUST NOT self-approve your own source code changes.
6. **Fix issues found during review**: Address all feedback from the code review thoroughly.
7. **Repeat steps 1-6 as necessary until the code is approved.**
8. **Report to the team**: Notify the team of your changes. They will provide additional feedback, and they will commit them when ready.

## Verification

- [ ] Review every line of code for adherence to coding standards
- [ ] Attempt to simplify or improve any code in the changeset for clarity and maintainability
- [ ] Ensure `make fmt` passes
- [ ] Ensure linting passes (`make lint`) (can use `make lint-fix` to fix issues)
- [ ] Ensure tests and coverage pass (`make coverage`)
- [ ] Ensure build (`make build`)
- [ ] Documentation updated if needed
- [ ] Code review approved
- [ ] The team/user has reviewed the changes and explicitly asked for completion

## IMPORTANT

- Refer to the existing `Makefile` for commands related to building, testing, and coverage enforcement, and **always invoke the documented `make` targets (e.g., `make coverage`, `make test-race`, `make build`) instead of calling toolchain binaries directly**. This ensures we honor repo-specific flags and hooks.
- Evaluate your methods and thinking against this document at all times. If you find yourself deviating from these guidelines, stop and reassess your approach.
- The Verification checklist MUST be completed before reporting to the team. It's not a suggestion, it's not a guideline - it's a HARD requirement.
- You may not call a task finished yourself. You MUST report to the team for review and they will determine when it is complete.
- Markdown files must always use ASCII and proper markdown syntax.
