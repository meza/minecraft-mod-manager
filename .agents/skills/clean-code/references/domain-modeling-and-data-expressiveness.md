# Domain modeling and data expressiveness

Types, signatures, and data structures express domain concepts and the invariants that govern them, instead of flattening meaning into primitives, flags, and generic containers that invite misuse.

Use the descriptions below to decide which individual quality references are relevant. Open only the references that can materially affect the current decision, code, or failure.

- [Concepts have explicit types](domain-modeling-and-data-expressiveness/concepts-have-explicit-types.md) - Important domain concepts are represented explicitly rather than being flattened into vague primitives, strings, flags, and maps.
- [Return values are meaningful](domain-modeling-and-data-expressiveness/return-values-are-meaningful.md) - Outputs communicate useful meaning rather than ambiguous booleans, magic values, or overloaded null semantics.
- [Invariants are protected](domain-modeling-and-data-expressiveness/invariants-are-protected.md) - The code structure makes invalid states difficult or impossible to represent or persist.
- [Boolean blindness is avoided](domain-modeling-and-data-expressiveness/boolean-blindness-is-avoided.md) - Flags and ambiguous booleans are not used where richer types or named concepts would express intent better.
- [Primitive obsession is avoided](domain-modeling-and-data-expressiveness/primitive-obsession-is-avoided.md) - Repeated low-level primitives do not stand in for richer domain concepts that deserve first-class representation.
- [Data shape matches behavior](domain-modeling-and-data-expressiveness/data-shape-matches-behavior.md) - Data structures reflect access patterns and invariants instead of being generic containers that invite misuse.
- [Data ownership is clear](domain-modeling-and-data-expressiveness/data-ownership-is-clear.md) - It is obvious which component or service is authoritative for a given piece of data or business rule.

