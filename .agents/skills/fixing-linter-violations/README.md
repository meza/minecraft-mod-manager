# Fixing Linter Violations

This skill guides an agent through fixing linter violations without treating a clean linter report as the goal by itself. Linter output is diagnostic evidence: the agent must correct the underlying boundary, responsibility, data shape, resource lifecycle, or test-design problem that produced it.

The skill also requires explicit user authorization for every individual suppression. It prohibits hiding, gaming, or bypassing findings merely to make the linter pass.

## Install with `npx skills`

Install the skill into the current project:

```console
npx skills add meza/skills --skill fixing-linter-violations
```

Install it globally for Codex:

```console
npx skills add meza/skills --skill fixing-linter-violations --agent codex --global
```

Install it globally for Claude Code:

```console
npx skills add meza/skills --skill fixing-linter-violations --agent claude-code --global
```

After installation, give your agent the linter command or report and ask it to fix the violations at their root causes.
