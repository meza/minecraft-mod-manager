# Agent Guidance

## Persona

### Only when asked to review code

If you're responding to address a review, please follow the instructions for the [other](#in-any-other-situation) situation below.

Do not process this section any further if you are not explicitly asked to review code.

If you are truly only reviewing code and not adding new code or functionality, then:

You must inhabit the role described in this file: https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/CodeReview.md
You must make all attempts to acquire it and incorporate it into your responses.

### In any other situation

You must inhabit the role described in this file: https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/Engineer.md
You must make all attempts to acquire it and incorporate it into your responses.

## Issue Tracking

Instructions for issue tracking [here](https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/Beads.md).
You MUST read and adhere to these instructions.

## Long Term Memory

Instructions for long term memory management [here](https://raw.githubusercontent.com/meza/agent-docs/refs/heads/main/LongTermMemory.md).
You MUST read and adhere to these instructions, and you MUST update `memory.md` during the session—not just at the end. Capture new insights, user preferences, and open questions immediately; deferring notes risks losing context if the session drops.

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
- We use makefiles for build automation. Refer to the existing `Makefile` for commands related to building, testing, and coverage enforcement, and **always invoke the documented `make` targets (e.g., `make test`, `make coverage`, `make build`) instead of calling toolchain binaries directly**. This ensures we honor repo-specific flags and hooks.

## Knowledge Material

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

## Development Workflow

1. **Write tests first**: Follow TDD principles where possible
2. **Implement changes**: Make minimal, focused changes
3. **Verify continuously**: Run the relevant tests frequently during development
4. **Final verification**: Follow the quality gates below before submitting
5. **Report to the team**: Notify the team of your changes for review. They will provide feedback and they will commit them when ready.

## User Facing Documentation Reminders

Write user-facing docs in a conversational, guide-like tone:

- Address the reader as 'you'; use active voice.
- Start with what the command does in 1–2 sentences, then explain the most common scenario and why you'd use options (briefly).
- Prefer short paragraphs over spec sections like 'Behaviour/Edge Cases'; only mention edge cases as 'If X happens, do Y'.
- Include at least one copy/paste example command near the top.
- Then include a simple flags table (flag, meaning, allowed values, example).
- Keep language non-technical; define any necessary terms in a short clause.

### User Facing Documentation Principles

- must always reflect the current state of the project without mentions of future plans, historical states or internal processes.
- must be written with empathy for the user's perspective, anticipating their needs and questions for the _current_ state of the project.

## Verification

- [ ] Review every line of code for adherence to coding standards
- [ ] Attempt to simplify or improve any code in the changeset for clarity and maintainability
- [ ] Ensure `go fmt ./...` passes
- [ ] Ensure tests pass (`./make test`)
- [ ] Ensure coverage is 100% (`./make coverage-enforce`)
- [ ] Ensure build (`./make build`)
- [ ] Documentation updated if needed
- [ ] The team/user has reviewed the changes and explicitly asked for completion

## IMPORTANT

- Refer to the existing `Makefile` for commands related to building, testing, and coverage enforcement, and **always invoke the documented `make` targets (e.g., `make test`, `make coverage`, `make build`) instead of calling toolchain binaries directly**. This ensures we honor repo-specific flags and hooks.
- Evaluate your methods and thinking against this document at all times. If you find yourself deviating from these guidelines, stop and reassess your approach.
- The Verification checklist MUST be completed before reporting to the team. It's not a suggestion, it's not a guideline—it's a HARD requirement.
- You may not call a task finished yourself. You MUST report to the team for review and they will determine when it is complete.
- Markdown files must always use ASCII and proper markdown syntax.
