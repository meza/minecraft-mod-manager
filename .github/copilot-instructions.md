# GitHub Copilot Instructions

## Project Overview

This is the **Minecraft Mod Manager** project - a TypeScript-based CLI tool for managing Minecraft mods from Modrinth and CurseForge. The project follows strict development practices and quality standards.

## Core Development Principles

### 1. Conventional Commit Messages (REQUIRED)

**ALL commits MUST use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification.**

- Format: `<type>[optional scope]: <description>`
- Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
- Examples:
  - `feat: add new mod scanning functionality`
  - `fix: resolve version check error handling`
  - `test: add comprehensive tests for config validation`
  - `docs: update README with new installation steps`

**Commits not following this specification will be rejected.**

### 2. Continuous Integration Verification (REQUIRED)

**ALWAYS run `pnpm run ci` before considering any changes complete.**

This command executes:
- TypeScript compilation check (`pnpm lint`)
- Biome linting and formatting (`pnpm lint:ci`)
- Full test suite with coverage (`pnpm report`)

**Changes that break CI will be rejected.**

### 3. Test Coverage Requirements (STRICT)

**100% test coverage is mandatory - this is the bare minimum.**

- Write tests for ALL new functionality
- Modify existing tests when changing behavior
- Use meaningful test descriptions and assertions
- Follow existing test patterns (vitest, jest-chance for test data)
- **NEVER remove, skip, or disable tests without explicit clarification from the team**

If you think a test needs to be removed or disabled, stop and ask for guidance first.

### 4. Code Quality and Architecture

#### File Organization
- **Actions**: User-facing commands and operations (`src/actions/`)
- **Lib**: Core business logic and utilities (`src/lib/`)
- **Interactions**: User interaction prompts (`src/interactions/`)
- **Repositories**: External API integrations (`src/repositories/`)

#### Console Output Rules
- `console.log` and `console.error` are ONLY allowed in the `actions` folder
- Use the `Logger` class for all logging in `lib` folder
- See [ADR-0002](doc/adr/0002-console-log-only-in-actions.md) for details

#### Software Hygiene
- **Boy Scout Rule**: Leave code cleaner than you found it
- Clear separation of concerns
- Meaningful variable and function names
- Proper error handling
- No magic numbers or hardcoded values
- Follow existing patterns and conventions

### 5. Dependencies and Package Management

- Use **pnpm** as the package manager
- Run `pnpm install` to install dependencies
- Keep dependencies minimal and justified
- Update package.json only when necessary

### 6. Linting and Formatting

- Use Biome for linting and formatting
- **DO NOT** suppress linting errors unless absolutely unavoidable
- **DO NOT** modify rules to make them less strict
- **DO NOT** argue about the rules
- Ask for help if you don't understand a rule violation

### 7. Documentation

- Document all new features and changes
- Update README.md when adding new functionality
- Maintain consistent language and style
- Update relevant ADR files when making architectural decisions

## When in Doubt

**DO NOT make assumptions or guess.** Instead:

1. Research the existing codebase for similar patterns
2. Check the ADR documentation in `doc/adr/`
3. Review the README.md and CONTRIBUTING.md
4. Ask for clarification from the team

**Never make things up or implement solutions without understanding the requirements.**

## Testing Guidelines

### Test Structure
- Use vitest as the testing framework
- Use jest-chance for generating test data
- Mock external dependencies appropriately
- Follow the existing test patterns in the codebase

### Test Categories
- Unit tests for individual functions/classes
- Integration tests for workflows
- All tests must be deterministic and fast
- Tests should be readable and maintainable

### Coverage Requirements
- 100% line coverage
- 100% branch coverage
- 100% function coverage
- Meaningful assertions, not just coverage for coverage's sake

## Development Workflow

1. **Before starting**: Run `pnpm ci` to ensure baseline passes
2. **Write tests first**: Follow TDD principles where possible
3. **Implement changes**: Make minimal, focused changes
4. **Verify continuously**: Run `pnpm ci` frequently during development
5. **Commit with conventional messages**: Follow the commit format strictly
6. **Final verification**: Ensure `pnpm ci` passes before submitting

## Project-Specific Considerations

### Minecraft Mod Management
- Support for CurseForge and Modrinth platforms
- Version compatibility checking
- Mod dependency resolution
- File integrity verification (hashing)

### Configuration Files
- `modlist.json`: Main configuration
- `modlist-lock.json`: Version lock file (managed by app)
- Support for custom ignore patterns

### External APIs
- Rate limiting for API calls
- Proper error handling for network issues
- Respect platform-specific requirements

## Quality Gates

Before any pull request:
- [ ] All tests pass (`pnpm test`)
- [ ] 100% coverage maintained (`pnpm report`)
- [ ] No linting errors (`pnpm lint:ci`)
- [ ] TypeScript compiles without errors (`pnpm lint`)
- [ ] Conventional commit format used
- [ ] Documentation updated if needed
- [ ] No console.log/error in lib folder

**Remember: These are not suggestions - they are requirements. Adherence to these standards is mandatory for all contributions.**