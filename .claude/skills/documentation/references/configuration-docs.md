# Configuration Documentation Guide

Configuration is a first-class documentation topic. Treat it as essential content, not an appendix.

## Documentation Structure

### Quick Start

Begin with the minimal viable configuration. Show a working example before explaining options.

```yaml
# Minimal configuration to get started
setting: value
```

Explain what this achieves and when readers might need more.

### Common Scenarios

Document real usage patterns before the option catalog. Scenarios answer "how do I..." questions:

- Development environment setup
- Production deployment
- Testing configuration
- Common customizations

Each scenario should be a complete, runnable example with context.

## Option Catalog

Document every configuration option with consistent fields. Not all fields apply to every option.

### Required Fields

| Field | Description |
|-------|-------------|
| **Name** | The exact key or flag name |
| **Purpose** | What this option controls, in plain language |
| **Type** | Data type and allowed values (string, integer, enum, etc.) |
| **Default** | The value when not specified, or "none/required" |
| **Required** | Whether the system fails without it |

### Contextual Fields

| Field | When to Include |
|-------|-----------------|
| **Sources/Precedence** | When multiple sources exist (file, env, flag) |
| **Reload Behavior** | For runtime-changeable settings |
| **Security Notes** | For sensitive values or security implications |
| **Interactions** | When options affect or conflict with others |
| **Versioning** | When option was added, deprecated, or changed |
| **Example** | For complex types or non-obvious usage |

### Example Option Entry

```markdown
### database_url

**Purpose:** Connection string for the primary database.

**Type:** String (URI format)

**Default:** None (required)

**Required:** Yes

**Sources:** Environment variable `DATABASE_URL`, config file `database.url`, CLI flag `--database-url`. Environment variable takes precedence.

**Security:** Store in environment variable or secrets manager. Never commit to version control.

**Example:**
postgres://user:pass@localhost:5432/mydb?sslmode=require
```

## File Formats and Locations

Document where configuration files live and what formats are supported.

### Search Order

List locations in precedence order:

1. Command-line flags (highest precedence)
2. Environment variables
3. User config file (`~/.config/app/config.yaml`)
4. Project config file (`./config.yaml`)
5. System config file (`/etc/app/config.yaml`)
6. Built-in defaults (lowest precedence)

### Supported Formats

For each format, document:

- File extension(s) recognized
- Any format-specific behaviors
- Example of the same configuration in each format

## Environment Variables

### Naming Convention

Describe the pattern for deriving environment variable names:

```
PREFIX_SECTION_OPTION
Example: MYAPP_DATABASE_URL
```

### Environment Variable Reference

Provide a complete mapping table:

| Variable | Config Equivalent | Description |
|----------|-------------------|-------------|
| `MYAPP_DATABASE_URL` | `database.url` | Primary database connection |
| `MYAPP_LOG_LEVEL` | `logging.level` | Logging verbosity |

## CLI Flags

### Flag Reference

Map CLI flags to their configuration equivalents:

| Flag | Config Equivalent | Description |
|------|-------------------|-------------|
| `--database-url` | `database.url` | Primary database connection |
| `-v, --verbose` | `logging.level=debug` | Enable verbose output |

### Flag-only Options

Document options that exist only as CLI flags (no config file equivalent) and explain why.

## Validation and Failure Modes

### Validation Rules

Document constraints beyond type checking:

- Value ranges and bounds
- Format requirements (regex patterns, URI schemes)
- Required combinations (if A then B required)
- Mutual exclusions (cannot set both X and Y)

### Error Messages

Provide examples of validation errors and how to resolve them:

```
Error: database_url must be a valid PostgreSQL URI
Solution: Ensure the URL starts with postgres:// or postgresql://
```

### Failure Behavior

Document what happens when configuration is invalid:

- Does the system refuse to start?
- Are invalid values ignored with warnings?
- Are there fallback behaviors?

## Migration Guidance

### Between Versions

When configuration changes between versions:

| Old Option | New Option | Migration Notes |
|------------|------------|-----------------|
| `db_host` | `database.url` | Combine host, port, and database into URI |

### Deprecation Notices

For deprecated options, document:

- When it was deprecated
- When it will be removed
- What replaces it
- Automatic migration behavior (if any)

## Example Library

Collect complete, tested examples for common setups:

### Development

```yaml
# Full development configuration
# Optimized for local development with verbose logging
```

### Production

```yaml
# Production configuration template
# Includes security hardening and performance tuning
```

### Testing

```yaml
# Test configuration
# Isolated settings for automated testing
```

Each example should include comments explaining non-obvious choices.
