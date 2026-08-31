# Data migrations are survivable

Schema and data evolution are designed to happen safely across versions and deployments.

Safe migration design matters wherever data schemas, state machines, or distributed contracts evolve over time. A maintainable system can move from old to new without trapping operators in unsafe all-at-once transitions. In review, this is about whether the system can survive evolution, skew, retries, failures, and operational reality without losing coherence. Strong signals are deliberate recovery stories, compatibility-aware boundaries, reproducible delivery, navigable structure, and interfaces that make correct use easier than misuse. Weak signals are brittle migrations, unexamined version skew, hidden build assumptions, and local shortcuts that quietly increase long-term coupling. The educational point is that maintainability is proven over time, especially when the system is under stress or in transition. For this specific symptom, the reviewer should ask whether the change makes 'Data migrations are survivable' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
ALTER customers DROP name, ADD full_name REQUIRED
UPDATE customers SET full_name = name
DEPLOY read_and_write(full_name)
```

### Good

```text
ALTER customers ADD full_name OPTIONAL
DEPLOY write(name, full_name); read(full_name ?? name)
BACKFILL full_name IN resumable_batches
VERIFY no_missing_full_name AND no_legacy_reads
ADD AND VALIDATE required(full_name)
DEPLOY read_and_write(full_name)
ALTER customers DROP name AFTER compatibility_window
```
