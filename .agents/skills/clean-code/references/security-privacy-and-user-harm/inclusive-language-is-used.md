# Inclusive language is used

Naming, docs, messages, and user-facing outputs avoid unnecessary exclusionary or misleading language.

Inclusive language avoids unnecessary exclusion or harmful ambiguity in names, docs, and user-facing messages. It also improves clarity by choosing precise and respectful terms. In review, this is about whether quality is defined broadly enough to include the people who must use, read, or depend on the system. Strong signals are wording and behavior that are precise, respectful, and accessible without unnecessary assumptions. Weak signals are exclusionary terminology, user-facing flows that presume one kind of user, or semantics that make correct use harder for part of the audience. The educational point is that maintainable systems do not externalize avoidable difficulty onto users. For this specific symptom, the reviewer should ask whether the change makes 'Inclusive language is used' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
master = connectDatabase()
slaves = master.getSlaves()
blacklist = loadBlacklist()
```

### Good

```text
primaryDatabase = connectDatabase()
readReplicas = primaryDatabase.getReadReplicas()
blockedAccountIds = loadBlockedAccountIds()
```
