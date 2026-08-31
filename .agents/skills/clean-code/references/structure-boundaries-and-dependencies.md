# Structure boundaries and dependencies

The arrangement of modules reflects real responsibilities: related things live together, variation and shared rules have one owner, and dependencies point the right way through narrow contracts.

Use the descriptions below to decide which individual quality references are relevant. Open only the references that can materially affect the current decision, code, or failure.

- [Related things live together](structure-boundaries-and-dependencies/related-things-live-together.md) - Logic, data, and rules that belong together are not scattered across distant files or layers.
- [Duplication is intentional or absent](structure-boundaries-and-dependencies/duplication-is-intentional-or-absent.md) - Repeated logic is either eliminated or clearly justified, rather than copied accidentally across the codebase.
- [Variation is localized](structure-boundaries-and-dependencies/variation-is-localized.md) - When behavior varies, the points of variation are narrow and explicit rather than smeared through conditionals everywhere.
- [Dependencies and module interactions are explicit and well-shaped](structure-boundaries-and-dependencies/dependencies-and-module-interactions-are-explicit-and-well-shaped.md) - Collaborators, dependency direction, coupling, cohesion, and interface shape work together so modules depend on the right things, for the right reasons, through narrow and comprehensible contracts.
- [Shared rules have a single owner](structure-boundaries-and-dependencies/shared-rules-have-a-single-owner.md) - Core rules are defined in one authoritative place rather than being re-stated in multiple modules and tests.
- [Boundaries and layers reflect real responsibilities](structure-boundaries-and-dependencies/boundaries-and-layers-reflect-real-responsibilities.md) - Module boundaries, layering, boundary crossings, and separation of policy from mechanism are arranged so the structure of the system matches its real responsibilities instead of incidental technical detail.
- [Framework does not own the design](structure-boundaries-and-dependencies/framework-does-not-own-the-design.md) - The architecture is not merely a mirror of the chosen framework's defaults or constraints.
- [Architectural integrity is preserved across domains and interfaces](structure-boundaries-and-dependencies/architectural-integrity-is-preserved-across-domains-and-interfaces.md) - The architecture keeps a clear organizing principle, respects domain boundaries, preserves intent across APIs and integrations, resists drift, enforces system-wide rules structurally, and remains organized around domain meaning rather than utility sprawl.

