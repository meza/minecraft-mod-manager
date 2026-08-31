# Local conventions are consistent across the codebase

Similar problems are solved with the same naming, structure, documentation, and cross-cutting conventions so that a reviewer sees one coherent local style instead of competing patterns.

A healthy codebase develops a recognizable local grammar. Similar modules, APIs, and features should tend to use the same naming style, the same documentation habits, the same broad structural patterns, and the same conventions for repeated concerns such as validation, logging, tracing, auth, or typing. This does not mean every line must look identical. It means contributors should not repeatedly encounter multiple equally valid local dialects for the same kind of problem.

Strong signs include stable naming conventions, repeated solutions to repeated problems, alignment between code, tests, and documentation, and cross-cutting concerns handled through familiar patterns. Weak signs include several competing local styles for the same task, docs that teach one behavior while code implements another, inconsistent naming for the same concept, and ad hoc handling of recurring concerns depending on who last touched the file.

This symptom matters because consistency reduces cognitive load and improves review accuracy. When the codebase has one local grammar, readers can transfer understanding from one area to another. When it has many, every new file becomes a relearning exercise and defects hide in the gaps between styles. This row is about local coherence across repeated practices. Whole-system product coherence remains a separate concern.

## Examples

### Bad

```text
function createOrder(command):
  validate(command)
  audit.record("order_created")
function cancelOrder(command):
  ensureValid(command)
  logger.info("cancelled")
```

### Good

```text
function createOrder(command):
  validate(command)
  audit.record("order_created")
function cancelOrder(command):
  validate(command)
  audit.record("order_cancelled")
```
