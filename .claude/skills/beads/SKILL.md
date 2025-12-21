---
name: beads
description: >
  Interact with the Beads file-based issue tracker. MUST USE when: (1) working with files
  in the `.beads/` folder, (2) managing issues, tasks, bugs, or work items in projects
  containing a `.beads/` directory, (3) user asks about issue tracking, task management,
  or work item operations. Provides guidance for using the `bd` CLI and `mcp__beads__*`
  functions correctly.
---

# Beads Issue Tracker

A file-based issue tracker storing issues in `.beads/` at project root. Presence of `.beads/` folder indicates Beads usage.

## Critical Rules

- NEVER directly access `.beads/` folder contents
- Always use `bd` CLI or `mcp__beads__*` functions
- Use `--no-db --json` flags for automated/agent processes
- Synchronization occurs automatically via git integration

## Issue Types

| Type      | Purpose                                    |
|-----------|--------------------------------------------|
| `bug`     | Defects requiring fixes                    |
| `feature` | New capabilities                           |
| `task`    | Work items including testing/documentation |
| `epic`    | Substantial features with subtasks         |
| `chore`   | Maintenance activities                     |

## Priority Levels

| Priority | Meaning                                        |
|----------|------------------------------------------------|
| `0`      | Critical (security, data integrity, build)     |
| `1`      | High (major features, significant bugs)        |
| `2`      | Medium (default)                               |
| `3`      | Low (enhancements, performance tuning)         |
| `4`      | Backlog (future considerations)                |

## Standard Workflow

1. View available work: `bd --no-db ready`
2. Claim an issue: `bd --no-db update <id> --status in_progress`
3. Execute work with testing and documentation
4. Link discovered issues: `bd --no-db create "title" -p 1 --deps discovered-from:<parent-id>`
5. Finalize: `bd --no-db close <id> --reason "Done"`

## Common Commands

```bash
# List ready issues
bd --no-db ready

# List all issues
bd --no-db list

# Create issue
bd --no-db create "Issue title" --type bug --priority 1

# Update status
bd --no-db update <id> --status in_progress

# Close issue
bd --no-db close <id> --reason "Completed"

# JSON output for parsing
bd --no-db --json list
```

## Key Findings from CLI Exploration

### Dependencies/Linking

```bash
# Add dependency (issue depends on another)
bd --no-db dep add <issue-id> <depends-on-id>

# Add with specific type (blocks, related, parent-child, discovered-from)
bd --no-db dep add <issue-id> <other-id> --type related
bd --no-db dep add <child-id> <parent-id> --type parent-child
bd --no-db dep add <new-id> <source-id> --type discovered-from

# View dependency tree
bd --no-db dep tree <issue-id>

# Remove dependency
bd --no-db dep remove <issue-id> <depends-on-id>
```

### Viewing Issue Details (including links)

```bash
# Show full issue with dependencies
bd --no-db show <id>

# JSON output shows deps in structured format
bd --no-db show <id> --json
```

The output shows "Depends on" and "Blocks" sections.

### Issue Fields (NOT notes/comments - these are different)

Beads has these distinct fields - agents confuse them:

| Field         | Purpose                                    |
|---------------|--------------------------------------------|
| `description` | Main issue body                            |
| `notes`       | Additional notes field                     |
| `design`      | Design notes                               |
| `acceptance`  | Acceptance criteria                        |
| `comments`    | Separate comment thread (NOT a field)      |

### Updating Fields

```bash
# Update description
bd --no-db update <id> --description "new description"

# Update notes (this IS a field, not comments)
bd --no-db update <id> --notes "additional notes"

# Update design notes
bd --no-db update <id> --design "design details"

# Update acceptance criteria
bd --no-db update <id> --acceptance "criteria here"
```

### Comments (separate from notes!)

```bash
# View comments on an issue
bd --no-db comments <id>

# Add a comment
bd --no-db comments add <id> "comment text"

# Add comment from file (for large text)
bd --no-db comments add <id> -f path/to/file.txt
```

### Adding Large Amounts of Text

For large content, use file-based approaches:

```bash
# Comments from file
bd --no-db comments add <id> -f my-notes.txt

# For description/notes/design, use edit command (opens $EDITOR)
bd --no-db edit <id> --description
bd --no-db edit <id> --notes
bd --no-db edit <id> --design
bd --no-db edit <id> --acceptance

# Or write to temp file and use as HEREDOC workaround:
cat << 'EOF' > /tmp/desc.txt
Large description content here...
Multiple lines...
EOF
bd --no-db update <id> --description "$(cat /tmp/desc.txt)"
```

### Common Mistakes to Avoid

1. **notes vs comments**: `notes` is a field on the issue, NOT the same as `comments`
2. **comments is separate**: `comments` is a separate thread, viewed/added via `bd comments`
3. **Bidirectional display**: Dependencies show both "Depends on" and "Blocks" in output
4. **Always use --no-db**: When running as an agent, always include the `--no-db` flag

## Installation

If `bd` is not available:

- NPM: `npm install -g @beads/bd`
- Unix: `curl -sSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash`
