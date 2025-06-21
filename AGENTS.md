# Agent Guidance

- Refer to `docs/requirements-go-port.md` for an overview of the current Node implementation and expectations for the Go port.
- See `docs/commands/README.md` for detailed behaviour of each CLI command.
- Review `docs/platform-apis.md` for specifics on interacting with CurseForge and Modrinth.
- Keep documentation in sync with features.

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

### Commit Guidelines

When constructing commit messages, please adhere to the following guidelines:

**ALL commits MUST use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification.**

- Format: `<type>[optional scope]: <description>`
- Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
- Examples:
  - `feat: add new mod scanning functionality`
  - `fix: resolve version check error handling`
  - `test: add comprehensive tests for config validation`
  - `docs: update README with new installation steps`
  - - `refactor: improve code structure without changing behavior`
  - `chore: anything that does not belong in the other categories`

**Commits not following this specification will be rejected.**

Ensure that commit types are chosen carefully, as they directly impact the software
version; only use `fix` or `feat` for changes that affect user-facing behavior,
and all commits must strictly follow the conventional commit format.

**Commit Message contents:**

The commit message should describe the value of the change, not the implementation details.
- **Good**: "fix: made the resource handling of the backup process hog the system less"
- **Bad**: "fix: added ionice and nice to the duply config"
- **Good**: "feat: added the option to schedule backups for the client containers"
- **Bad**: "feat: moved cron jobs to a script"


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
- Maintain consistent language and style


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
4. **Commit with conventional messages**: Follow the commit format strictly
5. **Final verification**: Follow the quality gates below before submitting

## Quality Gates

Before any pull request:
- [ ] Ensure tests pass (`./make test`)
- [ ] Ensure coverage is 100% (`./make coverage-enforce`)
- [ ] Ensure build (`./make build`)
- [ ] Conventional commit format used
- [ ] Documentation updated if needed

**Remember: These are not suggestions - they are requirements. Adherence to these standards is mandatory for all contributions.**
