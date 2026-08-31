# Data ownership is clear

It is obvious which component or service is authoritative for a given piece of data or business rule.

Clear data ownership prevents competing sources of truth. Reviewers should be able to tell which component governs a datum or rule and which others merely consume or replicate it. In review, this is about whether the codebase makes its own reasoning inspectable from evidence, boundaries, and structural cues. Strong signals are explicit assumptions, discoverable architectural intent, stable ownership, and interfaces that preserve domain meaning across boundaries. Weak signals are hidden sources of truth, accidental drift, undocumented structural constraints, and claims that rely on personal trust instead of inspectable proof. The educational point is that a codebase should teach maintainers how it is meant to work, not force them to rediscover that from history. For this specific symptom, the reviewer should ask whether the change makes 'Data ownership is clear' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
customerService.rename(id, name) -> sharedCustomers[id].name = name
billingService.correctName(id, name) -> sharedCustomers[id].name = name
```

### Good

```text
customerService.rename(id, name) -> ownedCustomers[id].name = name
billingService.correctName(id, name) -> customerService.rename(id, name)
```
