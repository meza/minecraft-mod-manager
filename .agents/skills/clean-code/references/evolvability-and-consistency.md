# Evolvability and consistency

The code can be changed, extended, and deleted safely and cheaply, and it solves problems the way the rest of the codebase already solves them.

Use the descriptions below to decide which individual quality references are relevant. Open only the references that can materially affect the current decision, code, or failure.

- [Blast radius is predictable](evolvability-and-consistency/blast-radius-is-predictable.md) - The likely impact of changing a rule, dependency, schema, or interface is limited and understandable.
- [Incremental evolution is supported](evolvability-and-consistency/incremental-evolution-is-supported.md) - The structure allows change in small safe steps rather than demanding broad rewrites for modest improvements.
- [Extension points are deliberate](evolvability-and-consistency/extension-points-are-deliberate.md) - The system has explicit seams for likely variation instead of forcing invasive modification for every new need.
- [Deletion is easy](evolvability-and-consistency/deletion-is-easy.md) - Dead code, outdated paths, and obsolete abstractions can be removed without fear of hidden dependency mines.
- [Local conventions are consistent across the codebase](evolvability-and-consistency/local-conventions-are-consistent-across-the-codebase.md) - Similar problems are solved with the same naming, structure, documentation, and cross-cutting conventions so that a reviewer sees one coherent local style instead of competing patterns.
- [The codebase is easy to navigate and hard to misuse](evolvability-and-consistency/the-codebase-is-easy-to-navigate-and-hard-to-misuse.md) - Repository shape, API design, and local structure help contributors find the right place to work, guide them toward correct usage, and make incorrect usage conspicuous or difficult.
- [The system is easy to reason about under change](evolvability-and-consistency/the-system-is-easy-to-reason-about-under-change.md) - When requirements evolve, it is obvious where to look, what to protect, and what else might be affected.
- [Long-term cost is visible in local choices](evolvability-and-consistency/long-term-cost-is-visible-in-local-choices.md) - Local shortcuts that create systemic maintenance debt are either rejected or explicitly acknowledged.
- [The system behaves like one coherent product](evolvability-and-consistency/the-system-behaves-like-one-coherent-product.md) - From function semantics up to architecture, the whole system expresses a unified set of rules, patterns, and expectations rather than conflicting local micro-cultures.

