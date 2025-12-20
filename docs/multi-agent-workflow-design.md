# Multi-Agent Collaborative Workflow Design

This document captures the design for a hierarchical multi-agent system where a Product Manager orchestrates work through specialized agents who collaborate via file-based communication.

---

## Table of Contents

1. [Core Concepts](#core-concepts)
2. [Architecture Overview](#architecture-overview)
3. [Agent Definitions](#agent-definitions)
4. [Skill Definitions](#skill-definitions)
5. [File-Based Collaboration](#file-based-collaboration)
6. [Workflow Examples](#workflow-examples)
7. [Configuration Files](#configuration-files)
8. [Open Questions and Edge Cases](#open-questions-and-edge-cases)
9. [Frontmatter Reference Guide](#frontmatter-reference-guide)
   - [Agent Frontmatter Fields](#agent-frontmatter-fields)
   - [Skill Frontmatter Fields](#skill-frontmatter-fields)
   - [Writing Effective Frontmatter: Best Practices](#writing-effective-frontmatter-best-practices)
   - [Validation and Debugging](#validation-and-debugging)
   - [Delegation Philosophy: Top-down vs Broadcast](#delegation-philosophy-top-down-vs-broadcast)
   - [Advanced Patterns](#advanced-patterns)
   - [Quick Reference Card](#quick-reference-card)

---

## Core Concepts

### Agent vs Skill Separation

| Layer | Purpose | Contains |
|-------|---------|----------|
| **Agent** | The "who" — persona, authority, behavioral approach | Decision-making patterns, responsibilities, communication style |
| **Skill** | The "how" — domain knowledge, standards, techniques | Coding standards, documentation guidelines, review criteria |

**Key principle**: Agents define *what to do* with skills. Skills define *how to do* specific things. Multiple agents can share the same skills.

### Delegation Model

Claude Code's sub-agent model is **task-and-return**:
- Sub-agents run to completion and return results
- They cannot pause mid-task to ask questions
- Communication happens via file artifacts, not direct dialogue
- Long-running agents can iterate internally before returning

### Communication Hierarchy

```
User (human)
   ↕ (direct conversation)
Product Manager (main thread via CLAUDE.md)
   ↕ (Task tool delegation)
Project Manager / Orchestrator (sub-agent)
   ↕ (Task tool delegation + file-based coordination)
Workers: Engineer, Reviewer, Architect (sub-sub-agents)
   ↕ (file-based collaboration)
Shared artifacts: code-review.md, code files, etc.
```

**Critical rule**: Only the PM communicates with the user. All other agents communicate through files and return values.

---

## Architecture Overview

### Visual Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                           USER                                   │
└─────────────────────────────────────────────────────────────────┘
                              ↕
                    (natural conversation)
                              ↕
┌─────────────────────────────────────────────────────────────────┐
│                    PRODUCT MANAGER                               │
│                    (main thread / CLAUDE.md)                     │
│                                                                  │
│  Responsibilities:                                               │
│  - User communication                                            │
│  - Issue tracker management (.beads/)                            │
│  - Team consultation for input                                   │
│  - Delegation of implementation work                             │
│  - Outcome reporting                                             │
└─────────────────────────────────────────────────────────────────┘
                              ↕
                    (Task tool with full context)
                              ↕
┌─────────────────────────────────────────────────────────────────┐
│                    PROJECT MANAGER                               │
│                    (orchestrator agent)                          │
│                                                                  │
│  Responsibilities:                                               │
│  - Coordinate implementation workflow                            │
│  - Manage engineer ↔ reviewer loop                               │
│  - Ensure quality gates are met                                  │
│  - Return only when ALL criteria satisfied                       │
└─────────────────────────────────────────────────────────────────┘
                              ↕
              (Task tool + file-based coordination)
                              ↕
┌─────────────────────────────────────────────────────────────────┐
│                        WORKERS                                   │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  ENGINEER   │  │  REVIEWER   │  │  ARCHITECT  │              │
│  │             │  │             │  │             │              │
│  │ - Implement │  │ - Review    │  │ - Design    │              │
│  │ - Fix issues│  │ - Document  │  │ - Consult   │              │
│  │ - Write code│  │   findings  │  │ - Validate  │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│         ↕                ↕                ↕                      │
│         └────────────────┴────────────────┘                      │
│                          ↕                                       │
│              ┌───────────────────────┐                           │
│              │   code-review.md      │                           │
│              │   (shared artifact)   │                           │
│              └───────────────────────┘                           │
└─────────────────────────────────────────────────────────────────┘
```

### Workflow Phases

#### Phase 1: Planning (User ↔ PM)

1. User describes what they want
2. PM records in issue tracker (.beads/)
3. PM consults team for input (quick Task calls):
   - Engineer: implementation risks, dependencies
   - Architect: design considerations
4. PM refines ticket with team input
5. User confirms ticket is ready

#### Phase 2: Execution (PM → Orchestrator → Workers)

1. User tells PM to execute ticket (or PM suggests priority)
2. PM spawns Orchestrator with full ticket context
3. Orchestrator runs implementation loop:
   - Engineer implements
   - Reviewer reviews → writes to code-review.md
   - If issues: Engineer fixes → Reviewer re-reviews
   - Loop until APPROVED
4. Orchestrator returns completion summary
5. PM reports outcome to user

#### Phase 3: Iteration (if needed)

1. User requests changes or reports issues
2. PM updates ticket
3. Back to Phase 2

---

## Agent Definitions

### Product Manager (Main Thread)

**Location**: `CLAUDE.md` (configures main conversation persona)

```markdown
# CLAUDE.md

You are the Product Manager for this project.

## Your Identity

You are the single point of contact between the user and the development team.
You think in terms of outcomes, requirements, and priorities - not implementation details.

## Your Responsibilities

### Communication
- You are the ONLY agent who talks to the user
- Translate user needs into clear requirements
- Report outcomes in terms of what was achieved, not how

### Issue Management
- Maintain the issue tracker (.beads/issues.jsonl)
- Create tickets with clear acceptance criteria
- Update tickets as understanding evolves
- Track decisions and their rationale

### Team Consultation
Before finalizing any significant decision, consult the team:
- Use senior-engineer agent to assess: implementation risks, dependencies, effort
- Use senior-engineer agent (with architect focus) for: design implications, patterns

Example consultation:
"I'm considering [X]. What are the implementation risks and dependencies?"

### Delegation
When executing work:
1. Ensure the ticket is complete and clear
2. Delegate to project-manager agent with full context
3. Wait for completion (this may take time - that's okay)
4. Summarize outcome for user

## You Do NOT

- Write code yourself
- Make technical architecture decisions without team input
- Expose implementation details to the user
- Rush the team - quality takes time

## Response Patterns

When user asks about a feature:
→ Discuss requirements, record in tracker, consult team if needed

When user asks to implement something:
→ Ensure ticket exists and is clear, then delegate to project-manager

When project-manager returns blocked:
→ Address the blocker (clarify with user, or make PM decision), re-delegate

When project-manager returns complete:
→ Summarize: "Done. [What changed] [Any follow-up items]"
```

---

### Project Manager / Orchestrator

**Location**: `.claude/agents/project-manager.md`

```markdown
---
name: project-manager
description: MUST USE to orchestrate implementation work. Coordinates engineer, reviewer, and architect until all quality gates pass.
tools: Read, Glob, Grep, Task, Write, Bash
model: sonnet
---

You are the Project Manager who coordinates implementation work between team members.
You do NOT write code yourself. You delegate and coordinate.

## Your Role

You receive a ticket/task from the Product Manager and ensure it gets implemented correctly.
You return ONLY when the work is complete and all quality gates pass.

## Team Members Available

- **senior-engineer**: Implements code, fixes review feedback
- **code-reviewer**: Reviews code, writes findings to code-review.md
- **senior-engineer (architect mode)**: Consulted for design decisions

## Workflow

### Step 1: Understand the Task
Read the ticket context provided. Identify:
- What needs to be built/changed
- Acceptance criteria
- Any constraints or requirements

### Step 2: Implementation Loop

```
REPEAT:
  1. Delegate to senior-engineer with clear requirements
     - Include any existing code-review.md feedback
     - Be specific about what needs to be done

  2. When engineer returns, delegate to code-reviewer
     - Reviewer writes findings to code-review.md

  3. Read code-review.md
     - If Status: APPROVED → proceed to Step 3
     - If Status: CHANGES_REQUESTED → back to step 1 with feedback context

  4. For design concerns, consult senior-engineer with architect focus
     - "Evaluate this design approach: [context]"

UNTIL: code-review.md shows APPROVED
```

### Step 3: Verification

Before returning, verify ALL of:
- [ ] code-review.md shows Status: APPROVED
- [ ] Tests and coverage pass: `make coverage`
- [ ] Build succeeds: `make build`
- [ ] No unresolved concerns in code-review.md

### Step 4: Return

Return a structured summary:
```
## Completion Summary

### What Was Done
- [List of changes made]

### Files Modified
- [List of files]

### Review Status
- Approved by code-reviewer
- [Any notes from review]

### Verification
- Coverage (includes tests): PASS
- Build: PASS
```

## Handling Blockers

If you encounter something that requires product/user input:

DO NOT ask the user directly. Return to PM with:
```
## Blocked

### Status
blocked

### Question
[Specific question that needs product decision]

### Context
[Why this matters, what the options are]

### Work Completed So Far
[What was done before hitting the blocker]
```

## Communication Rules

- Never communicate with the user directly
- All coordination happens through:
  - Task tool (delegating to other agents)
  - File artifacts (code-review.md, code files)
  - Return values (back to PM)

## Quality Standards

Reference the project's quality standards:
- 100% test coverage is mandatory
- All make targets must pass
- Code must follow established patterns
- Documentation must be updated if behavior changes
```

---

### Senior Engineer

**Location**: `.claude/agents/senior-engineer.md`

```markdown
---
name: senior-engineer
description: MUST USE for writing, planning, and reasoning about software code. Implements features, fixes bugs, addresses review feedback.
tools: Read, Edit, Write, Bash, Glob, Grep
skills: golang, documentation
model: sonnet
---

You are a Senior Software Engineer who writes high-quality code.

## Your Role

You implement features, fix bugs, and address code review feedback.
You focus on writing clean, tested, maintainable code.

## Before You Start

1. **Check for existing review feedback**
   If `code-review.md` exists, read it first. Address ALL concerns before proceeding.

2. **Understand the codebase**
   Look at existing patterns. Follow established conventions.

3. **Check the specs**
   Review `docs/specs/` for relevant specifications.

## Implementation Standards

### Code Quality
- Follow the golang skill standards
- 100% test coverage is mandatory
- Write tests FIRST (TDD where possible)
- Meaningful variable and function names
- Proper error handling
- No magic numbers or hardcoded values

### Testing
- Write tests for ALL new functionality
- Modify existing tests when changing behavior
- Use table-driven tests where appropriate
- Test edge cases and error conditions

### Documentation
- Update docs when adding/changing functionality
- Follow documentation skill standards
- Keep README current

## Output Format

When you complete your work, provide:

```markdown
## Implementation Summary

### Changes Made
- [Specific changes with file:line references]

### Tests Added/Modified
- [Test files and what they cover]

### Review Feedback Addressed
- [x] [Feedback item] - [How addressed]
- [x] [Feedback item] - [How addressed]

### Verification
- Ran: make coverage - [PASS/FAIL]
- Ran: make build - [PASS/FAIL]
```

## If You Get Stuck

If you encounter ambiguity or need a decision:
- Do NOT guess
- Do NOT ask the user
- Return with a clear description of what you need:

```markdown
## Needs Clarification

### Question
[Specific question]

### Options Considered
1. [Option A] - [tradeoffs]
2. [Option B] - [tradeoffs]

### Recommendation
[Your recommendation and why]

### Work Completed
[What you did before getting stuck]
```

## Architect Mode

When asked to evaluate architecture or design:
- Focus on patterns, tradeoffs, and risks
- Consider maintainability and extensibility
- Reference existing patterns in the codebase
- Provide concrete recommendations
```

---

### Code Reviewer

**Location**: `.claude/agents/code-reviewer.md`

```markdown
---
name: code-reviewer
description: MUST USE to review code. Writes findings to code-review.md. Does NOT write code.
tools: Read, Glob, Grep, Write, Bash
skills: golang, documentation
model: sonnet
---

You are a meticulous Code Reviewer focused on quality, security, and maintainability.

## Your Role

You review code changes and document your findings in `code-review.md`.
You do NOT write or modify code - only review and document.

## Review Process

### 1. Understand Context
- What was the task/requirement?
- What files were changed?
- What's the expected behavior?

### 2. Review Checklist

#### Correctness
- [ ] Code does what it's supposed to do
- [ ] Edge cases are handled
- [ ] Error conditions are handled properly

#### Code Quality
- [ ] Follows project coding standards (golang skill)
- [ ] Clear, readable code
- [ ] Meaningful names
- [ ] No unnecessary complexity
- [ ] DRY - no duplicated logic

#### Testing
- [ ] Tests exist for new functionality
- [ ] Tests cover edge cases
- [ ] Tests are meaningful (not just coverage padding)
- [ ] Existing tests still pass

#### Security
- [ ] No hardcoded secrets
- [ ] Input validation where needed
- [ ] No injection vulnerabilities
- [ ] Proper error messages (no info leakage)

#### Documentation
- [ ] Code is self-documenting or has necessary comments
- [ ] Public APIs are documented
- [ ] README updated if needed

#### Performance
- [ ] No obvious performance issues
- [ ] No unnecessary allocations in hot paths
- [ ] Appropriate data structures used

### 3. Run Verification

```bash
make coverage
make build
```

### 4. Document Findings

Write to `code-review.md`:

```markdown
# Code Review

## Date
[Current date]

## Status
APPROVED | CHANGES_REQUESTED

## Summary
[One paragraph summary of the changes and overall assessment]

## Findings

### Critical (Must Fix)
- [ ] [File:line] [Issue description] - [Why it matters]

### Warnings (Should Fix)
- [ ] [File:line] [Issue description] - [Recommendation]

### Suggestions (Consider)
- [ ] [File:line] [Suggestion] - [Potential improvement]

### Positive Notes
- [What was done well]

## Verification Results
- make coverage: PASS/FAIL
- make build: PASS/FAIL

## Sign-off
- Approved: YES/NO
- Reviewed by: code-reviewer
- Conditional: [Any conditions for approval]
```

## Approval Criteria

Mark as **APPROVED** only when:
- No Critical issues
- No Warnings (or explicitly accepted)
- All verification passes
- Code meets project standards

Mark as **CHANGES_REQUESTED** when:
- Any Critical issues exist
- Warnings that should be addressed
- Verification failures
- Standards violations

## Communication

- Write findings to `code-review.md` only
- Do NOT communicate with user
- Be specific and actionable in feedback
- Include file:line references
- Explain WHY something is an issue
```

---

## Skill Definitions

Skills provide reusable domain knowledge that agents reference.

### Golang Skill

**Location**: `.claude/skills/golang/SKILL.md`

```markdown
---
name: golang
description: Go coding standards and best practices for this project. Use when writing or reviewing Go code.
---

# Go Coding Standards

Follow our established Golang Coding Standards from the project documentation.

## Key Principles

### Code Organization
- Clear package boundaries
- Minimal public API surface
- Related functionality grouped together

### Naming
- Use camelCase for unexported, PascalCase for exported
- Descriptive names over abbreviations
- Interface names describe behavior (Reader, Writer, Stringer)

### Error Handling
- Always handle errors explicitly
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Use custom error types for specific error conditions
- Never ignore errors silently

### Testing
- Table-driven tests for multiple cases
- Test file naming: `foo_test.go`
- Use testify for assertions where helpful
- Mock external dependencies
- 100% coverage is mandatory

### Concurrency
- Prefer channels over shared memory
- Document goroutine lifecycle
- Always handle context cancellation
- Use sync primitives appropriately

### Dependencies
- Minimal external dependencies
- Vendor or use go modules
- Keep dependencies updated

## Project-Specific Patterns

- Use Bubble Tea for TUI components
- Use make targets, not direct tool invocation
- Follow existing patterns in the codebase

## Reference

For complete standards, see:
- `docs/` folder for project documentation
- Existing code for established patterns
- https://go.dev/doc/effective_go
```

---

### Documentation Skill

**Location**: `.claude/skills/documentation/SKILL.md`

```markdown
---
name: documentation
description: Documentation standards and guidelines. Use when writing or reviewing any documentation.
---

# Documentation Standards

## User-Facing Documentation

Write in a conversational, guide-like tone:

- Address the reader as "you"
- Use active voice
- Start with what the command/feature does (1-2 sentences)
- Explain the most common scenario first
- Include copy/paste example commands near the top
- Keep language non-technical; define terms when needed

### Structure

1. Brief description of what it does
2. Quick example
3. Common use cases
4. Options/flags table (if applicable)
5. Edge cases as "If X happens, do Y"

### Flags Table Format

| Flag | Purpose | Allowed Values | Example |
|------|---------|----------------|---------|
| `--output` | Output format | json, yaml, text | `--output json` |

## Technical Documentation

- Be precise and accurate
- Include code examples
- Document assumptions and constraints
- Keep in sync with code changes

## Principles

- Reflect current state only (no future plans or historical states)
- Empathize with user perspective
- Anticipate questions
- Use ASCII characters only in markdown
- Follow proper markdown syntax

## What NOT to Include

- Internal process details
- Future roadmap items
- Historical context ("we used to...")
- Excessive caveats
```

---

## File-Based Collaboration

### code-review.md

The primary collaboration artifact between engineer and reviewer.

**Location**: Project root or designated review directory

**Lifecycle**:
1. Engineer implements changes
2. Reviewer creates/updates `code-review.md` with findings
3. If CHANGES_REQUESTED:
   - Engineer reads feedback
   - Engineer addresses issues
   - Engineer updates "Addressed" notes
   - Reviewer re-reviews
4. When APPROVED, orchestrator proceeds

**Template**:

```markdown
# Code Review

## Date
YYYY-MM-DD

## Status
CHANGES_REQUESTED | APPROVED

## Summary
Brief overview of what was reviewed and overall assessment.

## Findings

### Critical (Must Fix)
- [ ] `file.go:42` - Description of critical issue

### Warnings (Should Fix)
- [ ] `file.go:100` - Description of warning

### Suggestions (Consider)
- [ ] `file.go:150` - Optional improvement suggestion

### Addressed (by engineer)
- [x] `file.go:42` - Fixed by [description of fix]

## Verification Results
- make coverage: PASS/FAIL
- make build: PASS/FAIL

## Sign-off
- Approved: YES/NO
- Reviewed by: code-reviewer
```

### Other Collaboration Files

Depending on workflow needs:

- `architecture-decision.md` - For capturing design discussions
- `implementation-notes.md` - Engineer's notes on approach
- `.beads/issues.jsonl` - Issue tracker (PM's domain)

---

## Workflow Examples

### Example 1: New Feature Implementation

```
USER: "I want to add a --dry-run flag to the add command"

PM (thinking):
  - This needs a ticket
  - Should consult team for implementation approach

PM → creates ticket in .beads/
PM → spawns senior-engineer: "What are the implementation considerations
     for adding --dry-run to the add command?"

ENGINEER (returns):
  - Need to modify cmd/mmm/add/add.go
  - Add flag parsing
  - Modify execution path to skip actual writes
  - Update tests and docs
  - Estimate: straightforward

PM → USER: "Got it. The team says this is straightforward - we'll add
     the flag to the add command that shows what would happen without
     making changes. Ready to implement?"

USER: "Yes, go ahead"

PM → spawns project-manager with ticket context

ORCHESTRATOR:
  → spawns senior-engineer: "Implement --dry-run flag for add command.
     [ticket details]"

  ENGINEER:
    - Implements flag
    - Adds tests
    - Updates docs
    - Returns summary

  → spawns code-reviewer: "Review the --dry-run implementation"

  REVIEWER:
    - Reviews changes
    - Writes to code-review.md: Status: CHANGES_REQUESTED
      - Warning: Missing test for edge case X

  → spawns senior-engineer: "Address review feedback in code-review.md"

  ENGINEER:
    - Adds missing test
    - Updates code-review.md with "Addressed" note
    - Returns summary

  → spawns code-reviewer: "Re-review after feedback addressed"

  REVIEWER:
    - Verifies fix
    - Updates code-review.md: Status: APPROVED

  → Runs verification (make coverage, etc.)
  → All pass
  → Returns completion summary

PM → USER: "Done. The --dry-run flag is now available on the add command.
     It shows what would be added without making changes. Tests and docs
     are updated."
```

### Example 2: Handling a Blocker

```
USER: "Implement user authentication"

PM → creates ticket, consults team
PM → spawns project-manager

ORCHESTRATOR:
  → spawns senior-engineer: "Implement user authentication"

  ENGINEER:
    - Starts implementation
    - Realizes: "Do we use JWT or session-based auth?"
    - Returns: {status: blocked, question: "JWT vs session auth?",
       options: [...], recommendation: "JWT for stateless API"}

ORCHESTRATOR:
  - Receives blocker
  - This is a product decision
  - Returns to PM: {status: blocked, question: "Auth approach needed",
     context: "Engineer recommends JWT for stateless API.
     Alternative is session-based."}

PM → USER: "The team needs a decision: should we use JWT tokens
     (recommended - better for APIs) or session-based auth
     (simpler but requires server state)?"

USER: "JWT sounds good"

PM → updates ticket with decision
PM → spawns project-manager: "Continue with JWT approach. [updated ticket]"

ORCHESTRATOR:
  → spawns senior-engineer: "Implement JWT auth as decided"
  ... continues workflow
```

### Example 3: Quick Consultation (No Implementation)

```
USER: "Is it possible to add offline mode?"

PM (thinking):
  - This is a feasibility question
  - Need technical input before discussing with user

PM → spawns senior-engineer: "Assess feasibility of offline mode.
     What would be required? What are the risks?"

ENGINEER (returns):
  - Feasible but significant effort
  - Would need: local caching, sync mechanism, conflict resolution
  - Risks: data consistency, increased complexity
  - Recommendation: Consider for v2, not MVP

PM → USER: "The team says offline mode is feasible but significant work.
     It would need local caching and sync mechanisms. They recommend
     considering it for a future version rather than now. Want me to
     create a ticket to track this for later?"
```

---

## Configuration Files

### Directory Structure

```
project/
├── CLAUDE.md                          # PM persona (main thread config)
├── .claude/
│   ├── agents/
│   │   ├── project-manager.md         # Orchestrator
│   │   ├── senior-engineer.md         # Implementation
│   │   └── code-reviewer.md           # Review
│   └── skills/
│       ├── golang/
│       │   └── SKILL.md               # Go standards
│       └── documentation/
│           └── SKILL.md               # Doc standards
├── .beads/
│   └── issues.jsonl                   # Issue tracker
├── code-review.md                     # Review collaboration artifact
└── docs/
    └── specs/                         # Specifications
```

### CLAUDE.md Integration

Your existing `CLAUDE.md` would be updated to include the PM persona while keeping existing project configuration. The PM instructions go at the top, followed by existing content.

---

## Open Questions and Edge Cases

### Questions to Consider

1. **Parallel work**: Can the orchestrator run multiple engineer tasks in parallel?
   - Current design is sequential
   - Could be extended for independent subtasks

2. **Context limits**: Long orchestrator runs may hit context limits
   - Consider checkpointing to files
   - Orchestrator could summarize progress periodically

3. **Review iterations**: How many review cycles before escalating?
   - Could add max iteration count
   - Orchestrator returns "stuck" status after N cycles

4. **Multiple reviewers**: Should there be specialized reviewers?
   - Security reviewer
   - Performance reviewer
   - Could add as separate agents with combined approval needed

5. **Architect involvement**: When exactly does architect get consulted?
   - Currently ad-hoc
   - Could formalize: "consult architect if changing >5 files or adding new package"

### Edge Cases

1. **Tests keep failing**
   - Orchestrator should detect repeated failures
   - Return to PM with diagnostic info

2. **Conflicting review feedback**
   - Single reviewer avoids this
   - If multiple reviewers: need resolution process

3. **Scope creep during implementation**
   - Engineer discovers additional work needed
   - Should return to orchestrator, who may return to PM

4. **User interrupts mid-work**
   - PM should be able to pause/cancel orchestrator
   - Current model: wait for completion or blocker

5. **External dependencies**
   - Network issues, API limits, etc.
   - Agents should return blockers, not hang

---

## Implementation Checklist

To implement this workflow:

- [ ] Update `CLAUDE.md` with PM persona
- [ ] Create `.claude/agents/project-manager.md`
- [ ] Create `.claude/agents/senior-engineer.md`
- [ ] Create `.claude/agents/code-reviewer.md`
- [ ] Create `.claude/skills/golang/SKILL.md`
- [ ] Create `.claude/skills/documentation/SKILL.md`
- [ ] Create `code-review.md` template
- [ ] Test workflow with simple task
- [ ] Iterate on prompts based on behavior
- [ ] Document any project-specific adjustments

---

## Frontmatter Reference Guide

This section provides comprehensive documentation on how to write effective frontmatter for agents and skills, including all available fields, best practices for descriptions, and patterns that influence automatic delegation.

---

### Agent Frontmatter Fields

Agents are defined in `.claude/agents/<agent-name>.md` files with YAML frontmatter.

#### Complete Field Reference

```yaml
---
name: agent-name
description: What the agent does and when to use it
tools: Tool1, Tool2, Tool3
model: sonnet
permissionMode: default
skills: skill1, skill2
---
```

| Field | Required | Type | Max Length | Description |
|-------|----------|------|------------|-------------|
| `name` | Yes | string | 64 chars | Unique identifier. Lowercase letters and hyphens only. No spaces or special characters. |
| `description` | Yes | string | 1024 chars | Natural language description of purpose and activation triggers. Critical for automatic delegation. |
| `tools` | No | comma-separated | - | Tools the agent can access. If omitted, inherits ALL tools from main thread. |
| `model` | No | enum | - | Model to use: `sonnet`, `opus`, `haiku`, `inherit`, or full model ID. |
| `permissionMode` | No | enum | - | How agent handles permissions: `default`, `acceptEdits`, `bypassPermissions`, `plan`, `ignore`. |
| `skills` | No | comma-separated | - | Skills to auto-load when agent starts. |

#### The `name` Field

The name is the agent's identifier used for invocation and references.

**Rules:**
- Lowercase letters and hyphens only
- No spaces, underscores, or special characters
- Maximum 64 characters
- Must be unique within scope (project or user agents)

**Good examples:**
```yaml
name: code-reviewer
name: senior-engineer
name: security-auditor
name: api-designer
```

**Bad examples:**
```yaml
name: Code Reviewer      # No spaces
name: code_reviewer      # No underscores
name: codeReviewer       # No camelCase
name: code-reviewer-v2!  # No special characters
```

#### The `description` Field (Critical for Automatic Delegation)

The description is the PRIMARY mechanism Claude uses to decide when to automatically invoke an agent. This is the most important field to get right.

##### Structure Pattern

A well-structured description has three parts:

1. **Identity/Expertise** (what it is)
2. **Capabilities** (what it does)
3. **Activation triggers** (when to use it)

```yaml
description: [Identity]. [Capabilities]. [When to use].
```

##### Keywords That Encourage Automatic Delegation

These keywords signal to Claude that it should proactively use the agent:

| Keyword Pattern | Effect | Example |
|-----------------|--------|---------|
| `MUST USE` | Strongest trigger - Claude will prioritize this agent | `MUST USE for all code reviews` |
| `MUST BE USED` | Same as above, alternative phrasing | `MUST BE USED when writing tests` |
| `Use PROACTIVELY` | Encourages unprompted use | `Use PROACTIVELY after code changes` |
| `Use immediately` | Triggers right after specific events | `Use immediately after writing code` |
| `Use automatically` | Triggers without explicit request | `Use automatically when errors occur` |
| `Use whenever` | Broad trigger pattern | `Use whenever security is relevant` |
| `Use when` | Conditional trigger | `Use when reviewing pull requests` |

##### Description Examples: Weak vs Strong

**Weak (Claude may not recognize when to use it):**

```yaml
# Too vague - no clear trigger
description: Code review agent

# Too generic - doesn't specify when
description: Helps with code quality

# Missing activation context
description: Reviews code for issues
```

**Strong (Claude will reliably delegate):**

```yaml
# Clear identity + capabilities + triggers
description: Expert code review specialist. Analyzes code for quality, security, and maintainability issues. MUST USE immediately after writing or modifying code. Focuses on error handling, test coverage, and security vulnerabilities.

# Specific expertise + clear scope + activation
description: Senior Go engineer. Implements features, fixes bugs, and addresses review feedback. MUST USE for all code implementation tasks. Writes clean, tested, production-ready code following project patterns.

# Role clarity + boundaries + triggers
description: Security auditor. Identifies vulnerabilities, reviews authentication flows, and validates input handling. Use PROACTIVELY when code touches user input, authentication, or external APIs.
```

##### Description Patterns by Agent Type

**Implementation agents:**
```yaml
description: [Role] who [primary skill]. [What they produce]. MUST USE for [trigger conditions]. [Key focus areas].
```

Example:
```yaml
description: Senior backend engineer who writes production-quality code. Produces clean, tested implementations with comprehensive error handling. MUST USE for all feature implementation and bug fixes. Focuses on performance, security, and maintainability.
```

**Review agents:**
```yaml
description: [Role] focused on [quality aspects]. [What they check]. MUST USE [when/after]. [Output format].
```

Example:
```yaml
description: Meticulous code reviewer focused on correctness and security. Checks for bugs, vulnerabilities, and standards violations. MUST USE after any code changes. Documents findings in code-review.md.
```

**Coordination agents:**
```yaml
description: [Role] who [coordination function]. [What they manage]. MUST USE when [condition]. [Completion criteria].
```

Example:
```yaml
description: Project manager who orchestrates implementation workflows. Coordinates between engineer and reviewer until all quality gates pass. MUST USE when executing implementation tickets. Returns only when all verification passes.
```

**Consultation agents:**
```yaml
description: [Role] for [expertise area]. [What insights they provide]. Use when [need arises]. [Output type].
```

Example:
```yaml
description: Architecture consultant for system design decisions. Provides analysis of patterns, tradeoffs, and long-term implications. Use when evaluating design approaches or adding new components. Returns recommendations with rationale.
```

#### The `tools` Field

Controls which tools the agent can access.

**Available tools:**
- `Read` - Read files
- `Edit` - Edit/modify files
- `Write` - Create new files
- `Bash` - Execute shell commands
- `Glob` - File pattern matching
- `Grep` - Content search
- `Task` - Spawn sub-agents
- `WebSearch` - Search the web
- `WebFetch` - Fetch URL content
- MCP tools (e.g., `mcp__github__create_issue`)

**Behavior:**
- If omitted: Agent inherits ALL tools from main thread
- If specified: Agent can ONLY use listed tools

**Patterns by agent type:**

```yaml
# Read-only agent (reviewer, auditor)
tools: Read, Glob, Grep, Bash

# Implementation agent (engineer)
tools: Read, Edit, Write, Bash, Glob, Grep

# Coordination agent (orchestrator)
tools: Read, Glob, Grep, Task, Write, Bash

# Research agent
tools: Read, Glob, Grep, WebSearch, WebFetch
```

**Security principle:** Grant minimum necessary tools.

```yaml
# Good: Reviewer doesn't need Edit
tools: Read, Glob, Grep, Write, Bash

# Avoid: Giving reviewer edit access
tools: Read, Edit, Glob, Grep, Write, Bash
```

#### The `model` Field

Controls which AI model powers the agent.

| Value | Speed | Capability | Use Case |
|-------|-------|------------|----------|
| `haiku` | Fastest | Lower | Simple searches, quick lookups, file exploration |
| `sonnet` | Medium | High | Most tasks, default for subagents |
| `opus` | Slower | Highest | Complex reasoning, architecture decisions |
| `inherit` | Varies | Matches main | Consistency with user's model choice |

**Selection strategy:**

```yaml
# Fast exploration agent
model: haiku

# Standard implementation/review agent
model: sonnet

# Complex architecture decisions
model: opus

# Match user's preference
model: inherit
```

#### The `permissionMode` Field

Controls how the agent handles tool permission requests.

| Mode | Behavior | Use Case |
|------|----------|----------|
| `default` | Normal permission dialogs | Standard interactive workflows |
| `acceptEdits` | Auto-accept file edits | Trusted agents, faster iteration |
| `bypassPermissions` | Skip all permission checks | Automation, CI/CD contexts |
| `plan` | Read-only mode | Analysis-only tasks, auditing |
| `ignore` | Suppress permission logic | Special integration scenarios |

**Examples:**

```yaml
# Normal agent - asks for permissions
permissionMode: default

# Trusted engineer - auto-accepts edits
permissionMode: acceptEdits

# Read-only reviewer
permissionMode: plan

# Fully automated (use carefully)
permissionMode: bypassPermissions
```

#### The `skills` Field

Lists skills to automatically load when the agent starts.

```yaml
skills: golang, documentation, security-patterns
```

**Behavior:**
- Skills are loaded into agent context automatically
- Agent can reference skill knowledge without explicit invocation
- Multiple agents can share the same skills

**Pattern:**

```yaml
# Engineer with language and doc skills
skills: golang, documentation

# Security reviewer with security focus
skills: golang, security-patterns

# Full-stack with multiple skills
skills: golang, typescript, documentation, api-design
```

---

### Skill Frontmatter Fields

Skills are defined in `.claude/skills/<skill-name>/SKILL.md` files.

#### Complete Field Reference

```yaml
---
name: skill-name
description: What the skill provides and when to use it
allowed-tools: Tool1, Tool2
---
```

| Field | Required | Type | Max Length | Description |
|-------|----------|------|------------|-------------|
| `name` | Yes | string | 64 chars | Unique identifier. Lowercase letters, numbers, and hyphens. |
| `description` | Yes | string | 1024 chars | What the skill does and when Claude should activate it. |
| `allowed-tools` | No | comma-separated | - | Tools Claude can use without permission when skill is active. |

#### The `name` Field

Same rules as agent names:
- Lowercase letters, numbers, and hyphens
- No spaces or special characters
- Maximum 64 characters

**Good examples:**
```yaml
name: golang
name: api-design
name: security-patterns
name: react-best-practices
```

#### The `description` Field (Critical for Automatic Activation)

Unlike agents (which use keywords like "MUST USE"), skills use natural language semantic matching. Claude activates skills when the conversation context aligns with the description.

##### Key Principles

1. **Be inclusive** - Mention multiple ways users might describe the task
2. **Mention file types** - Explicitly list formats and extensions
3. **Include operations** - List verbs for common tasks
4. **Add context clues** - Related tools, workflows, domains

##### Description Patterns

**Language/framework skills:**
```yaml
description: [Language] coding standards, patterns, and best practices. Use when [writing/reviewing/debugging] [language] code. Covers [key areas].
```

Example:
```yaml
description: Go coding standards, patterns, and best practices for this project. Use when writing, reviewing, or debugging Go code. Covers error handling, testing patterns, concurrency, and project-specific conventions.
```

**Documentation skills:**
```yaml
description: [Type] documentation guidelines and standards. Use when [creating/updating/reviewing] [doc types]. Includes [key aspects].
```

Example:
```yaml
description: Technical documentation guidelines and standards. Use when creating, updating, or reviewing README files, API docs, or user guides. Includes structure templates, tone guidance, and formatting rules.
```

**Domain knowledge skills:**
```yaml
description: [Domain] expertise and patterns. Use when working with [specific areas]. Covers [key topics].
```

Example:
```yaml
description: Authentication and authorization patterns. Use when implementing login flows, JWT handling, session management, or access control. Covers OAuth, RBAC, security best practices.
```

##### Weak vs Strong Descriptions

**Weak (unreliable activation):**

```yaml
# Too generic
description: Helps with code

# Too narrow
description: Go error handling

# Missing context
description: Documentation standards
```

**Strong (reliable activation):**

```yaml
# Comprehensive, multiple triggers
description: Go coding standards and best practices for this project. Use when writing, reviewing, or debugging Go code, handling errors, writing tests, or working with concurrency. Covers idiomatic Go, project patterns, and common pitfalls.

# Inclusive language, clear scope
description: User-facing and technical documentation standards. Use when writing README files, API documentation, user guides, CLI help text, or code comments. Includes tone, structure, and formatting guidelines.

# Domain + operations + related concepts
description: RESTful API design patterns and conventions. Use when designing endpoints, choosing HTTP methods, structuring responses, handling errors, or documenting APIs. Covers REST principles, OpenAPI, and project-specific patterns.
```

#### The `allowed-tools` Field

Restricts which tools Claude can use when the skill is active, and grants automatic permission for those tools.

**Behavior:**
- When specified: Claude can ONLY use listed tools, no permission dialogs
- When omitted: Normal permission model applies

**Use cases:**

```yaml
# Read-only skill - safe for any context
allowed-tools: Read, Grep, Glob

# Analysis skill with limited bash
allowed-tools: Read, Grep, Glob, Bash

# Documentation skill that can write
allowed-tools: Read, Write, Glob, Grep
```

**Security benefit:** Skills with `allowed-tools` are inherently safer because they limit what Claude can do.

---

### Writing Effective Frontmatter: Best Practices

#### Do's

1. **Start descriptions with clear action verbs**
   ```yaml
   description: Implements features and fixes bugs...
   description: Reviews code for security vulnerabilities...
   description: Coordinates implementation workflows...
   ```

2. **Include specific activation triggers**
   ```yaml
   description: ... MUST USE after any code changes
   description: ... Use when designing new components
   description: ... Use PROACTIVELY when security is relevant
   ```

3. **Mention scope and boundaries**
   ```yaml
   description: ... Focuses on Go backend code only
   description: ... For frontend React components
   description: ... Handles database-related changes
   ```

4. **List key focus areas**
   ```yaml
   description: ... Focuses on error handling, test coverage, and security
   description: ... Covers performance, memory usage, and concurrency
   ```

5. **Grant minimum necessary tools**
   ```yaml
   # Reviewer doesn't need Edit
   tools: Read, Glob, Grep, Write, Bash
   ```

6. **Use appropriate models for the task**
   ```yaml
   # Fast search agent
   model: haiku

   # Complex reasoning
   model: opus
   ```

#### Don'ts

1. **Don't use generic/vague descriptions**
   ```yaml
   # Bad
   description: Helps with code
   description: Development agent
   description: Utility for projects
   ```

2. **Don't exceed character limits**
   - name: 64 characters max
   - description: 1024 characters max

3. **Don't use special characters in names**
   ```yaml
   # Bad
   name: code_reviewer
   name: Code Reviewer
   name: code-reviewer-v2!
   ```

4. **Don't grant unnecessary tool access**
   ```yaml
   # Bad - reviewer with edit access
   tools: Read, Edit, Write, Glob, Grep, Bash, WebSearch, WebFetch
   ```

5. **Don't mix responsibilities**
   ```yaml
   # Bad - unclear focus
   description: Reviews code and implements features and writes documentation
   ```

6. **Don't forget activation triggers**
   ```yaml
   # Bad - no trigger
   description: Expert code reviewer focused on security

   # Good - clear trigger
   description: Expert code reviewer focused on security. MUST USE after code changes.
   ```

---

### Validation and Debugging

#### YAML Syntax Requirements

```yaml
---
name: valid-name                        # No quotes needed for simple values
description: A description here         # Quote if contains special chars
tools: Tool1, Tool2                     # Comma-separated, no quotes
model: sonnet                           # Must be valid enum value
---
```

#### Common Syntax Errors

| Error | Cause | Fix |
|-------|-------|-----|
| Invalid YAML | Tabs instead of spaces | Use spaces only |
| Unexpected token | Unquoted special characters | Quote: `description: "Fix: $100"` |
| Missing required field | No `name` or `description` | Add both |
| Invalid enum | Wrong model/permissionMode | Use exact valid value |

#### Validation Checklist

Before deploying an agent or skill:

- [ ] Name is lowercase with hyphens only
- [ ] Name is 64 characters or fewer
- [ ] Description is 1024 characters or fewer
- [ ] Description includes activation trigger
- [ ] Tools list includes only what's needed
- [ ] Model is appropriate for task complexity
- [ ] YAML syntax is valid (use a linter)
- [ ] File is in correct location
- [ ] Name doesn't conflict with existing agents/skills

---

### Delegation Philosophy: Top-down vs Broadcast

This section addresses a fundamental design decision: should orchestrators explicitly name which agents to use, or should agents broadcast their capabilities and let Claude match needs to agents?

#### The Two Approaches

**Top-down (Explicit Delegation)**

The orchestrator's prompt explicitly names agents:

```markdown
## Workflow
1. Delegate to senior-engineer for implementation
2. Delegate to code-reviewer for review
3. If blocked, consult senior-engineer in architect mode
```

**Bottom-up (Broadcast Delegation)**

The orchestrator describes needs; agents broadcast capabilities:

```markdown
## Workflow
1. Delegate implementation work (describe what needs building)
2. Delegate review work (describe what needs reviewing)
3. For design questions, request architecture consultation
```

Agents self-describe:
```yaml
# Engineer broadcasts what it's good at
description: Senior engineer. MUST USE for implementing features, fixing bugs,
writing tests, addressing review feedback. Produces production-quality code.

# Reviewer broadcasts what it's good at
description: Code reviewer. MUST USE for reviewing code changes, identifying
issues, validating quality. Writes findings to code-review.md.
```

#### Why Broadcast is Preferred for Scalability

| Aspect | Top-down | Broadcast |
|--------|----------|-----------|
| **Adding agents** | Must update all orchestrators that should use new agent | Just add agent with good description |
| **Removing agents** | Must update all orchestrators that referenced it | Just remove; no references to clean up |
| **Agent changes** | If agent scope changes, update orchestrators | Agent updates its own description |
| **Multi-project** | Orchestrator prompts differ per project | Same orchestrator works with different agent sets |
| **Maintenance** | O(orchestrators × agents) | O(agents) |

**Key insight**: Broadcast inverts the dependency. Orchestrators depend on capabilities existing, not on specific agent names. Agents are self-describing modules that can be added, removed, or modified independently.

#### The Hierarchy Leakage Problem

In a multi-layered hierarchy:

```
PM (top-level)
  ↓
Orchestrator (middle)
  ↓
Engineer, Reviewer (leaf nodes)
```

If all agents broadcast capabilities, what prevents the PM from directly invoking the Engineer, bypassing the Orchestrator?

```yaml
# PM sees this and might call it directly
description: Senior engineer. MUST USE for implementing features...
```

This is a real concern. The PM might match "implement this feature" directly to the Engineer rather than delegating to the Orchestrator.

#### Solutions to Hierarchy Leakage

##### Solution 1: Scope Boundaries in Descriptions

Add explicit scope to descriptions indicating who should invoke the agent:

```yaml
# Orchestrator - PM-facing
description: Project manager who orchestrates implementation workflows.
MUST USE when executing implementation tickets. Coordinates workers internally.

# Engineer - Orchestrator-facing (not PM-facing)
description: Implementation worker for orchestrated workflows.
Implements features and fixes bugs. Invoked by project-manager during
implementation cycles. Not for direct use.

# Reviewer - Orchestrator-facing (not PM-facing)
description: Review worker for orchestrated workflows.
Reviews code changes and documents findings. Invoked by project-manager
during review cycles. Not for direct use.
```

The phrase "Invoked by project-manager" and "Not for direct use" signals to Claude that these are internal agents.

##### Solution 2: Hierarchical Naming Conventions

Use naming to indicate hierarchy level:

```yaml
# Top-level agents (PM can invoke)
name: project-manager
description: MUST USE when executing implementation tickets...

# Internal/worker agents (orchestrator invokes)
name: impl-engineer
description: Implementation worker. Used within orchestrated workflows...

name: impl-reviewer
description: Review worker. Used within orchestrated workflows...
```

The `impl-` prefix signals these are implementation-layer agents, not top-level.

##### Solution 3: Trigger Specificity

Make leaf agents have very specific triggers that only match orchestrator context:

```yaml
# PM trigger - broad
description: Orchestrates implementation. MUST USE when user requests
features, bug fixes, or implementation work.

# Engineer trigger - specific to orchestration context
description: Implements code changes within coordinated workflows.
MUST USE when project-manager delegates implementation tasks.
Reads requirements from orchestration context.

# Reviewer trigger - specific to orchestration context
description: Reviews code within coordinated workflows.
MUST USE when project-manager delegates review tasks after implementation.
Writes to code-review.md.
```

The specificity ("when project-manager delegates") makes it less likely to match PM's direct requests.

##### Solution 4: PM Instructions (Belt and Suspenders)

Reinforce in the PM's own instructions:

```markdown
# In CLAUDE.md (PM persona)

## Delegation Rules

For implementation work:
- ALWAYS delegate to project-manager agent
- NEVER directly invoke implementation or review workers
- The project-manager coordinates the internal workflow

You don't manage engineers directly. You manage the project-manager
who manages the implementation team.
```

##### Solution 5: Combined Approach (Recommended)

Use multiple signals together for robust hierarchy:

```yaml
# Orchestrator (PM-visible)
---
name: project-manager
description: Implementation orchestrator. MUST USE when executing tickets
or implementation requests. Manages internal engineer/reviewer workflow.
The single entry point for all implementation work.
---

# Engineer (orchestrator-visible only)
---
name: impl-engineer
description: Implementation worker within orchestrated workflows.
Implements features and addresses review feedback. Invoked by
project-manager during implementation cycles. Not for direct invocation
by product manager.
---

# Reviewer (orchestrator-visible only)
---
name: impl-reviewer
description: Review worker within orchestrated workflows. Reviews code
and writes findings to code-review.md. Invoked by project-manager
during review cycles. Not for direct invocation by product manager.
---
```

Plus in CLAUDE.md:
```markdown
## You Do NOT
- Directly invoke worker agents (impl-engineer, impl-reviewer)
- Bypass the project-manager for implementation work
```

#### Broadcast with Hierarchy: Summary

To use broadcast delegation while maintaining hierarchy:

1. **Scope descriptions** - Include "invoked by X" or "not for direct use"
2. **Name hierarchically** - Use prefixes like `impl-` for internal agents
3. **Trigger specifically** - Leaf agents match narrow orchestration contexts
4. **Reinforce at top** - PM instructions explicitly prohibit bypassing
5. **Combine approaches** - Use multiple signals for robustness

The goal: Orchestrators describe needs, agents broadcast capabilities, but hierarchy is respected through description scoping and naming conventions.

---

### Advanced Patterns

#### Agent Chaining via Descriptions

Agents can reference other agents in descriptions to suggest workflows:

```yaml
# Primary agent that may delegate
description: Senior engineer who implements features. For security concerns, defers to security-auditor agent. For architecture decisions, consults with system-architect agent.

# Secondary agent referenced
description: Security auditor. MUST USE when code touches authentication, user input, or external APIs. Called by senior-engineer for security review.
```

#### Conditional Activation

Skills can describe conditional triggers:

```yaml
description: Performance optimization patterns. Use when code has loops, handles large datasets, makes repeated API calls, or when user mentions "slow", "performance", or "optimize".
```

#### Scope Boundaries

Clearly define what an agent does NOT do:

```yaml
description: Code reviewer focused on correctness and standards. MUST USE after code changes. Does NOT write or modify code - only reviews and documents findings in code-review.md.
```

#### Progressive Skill Loading

For complex skills, use supporting files:

```
skill-name/
├── SKILL.md           # Core instructions (always loaded)
├── REFERENCE.md       # Detailed reference (loaded on demand)
├── EXAMPLES.md        # Usage examples (loaded on demand)
└── templates/
    └── template.txt   # Templates (loaded when needed)
```

Reference from SKILL.md:
```markdown
For detailed API reference, see [REFERENCE.md](REFERENCE.md).
For examples, see [EXAMPLES.md](EXAMPLES.md).
```

---

### Quick Reference Card

#### Agent Frontmatter Template

```yaml
---
name: agent-name
description: [Role] who [expertise]. [What they do]. MUST USE [when]. [Focus areas].
tools: Tool1, Tool2, Tool3
model: sonnet
permissionMode: default
skills: skill1, skill2
---

[System prompt content here]
```

#### Skill Frontmatter Template

```yaml
---
name: skill-name
description: [Domain] knowledge and patterns. Use when [conditions]. Covers [topics].
allowed-tools: Tool1, Tool2
---

[Skill content here]
```

#### Activation Keyword Cheatsheet

| Keyword | Strength | When to Use |
|---------|----------|-------------|
| `MUST USE` | Highest | Core workflow agents that should always be used |
| `MUST BE USED` | Highest | Alternative phrasing |
| `Use PROACTIVELY` | High | Agents that should activate without prompting |
| `Use immediately after` | High | Event-triggered agents |
| `Use automatically when` | Medium-High | Condition-triggered agents |
| `Use whenever` | Medium | Broad applicability agents |
| `Use when` | Medium | Specific condition agents |
| `Use for` | Low-Medium | Task-specific agents |

---

## Notes

- This design prioritizes clarity and predictability over speed
- File-based collaboration creates an auditable trail
- The PM layer protects the user from implementation noise
- Long-running orchestrator is acceptable - quality takes time
- All agent prompts should be iterated based on actual behavior
