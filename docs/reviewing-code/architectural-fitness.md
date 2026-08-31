# Architectural fitness

Correct behaviour is necessary but not sufficient. A change must also put each responsibility in
the right owner, preserve the intended dependency boundaries, and use established capabilities at
the seam that owns them. Establish the target architecture from accepted decisions, the nearest
workspace guides, applicable skills, and repository configuration. Treat neighbouring legacy code
as current-state evidence, not proof that its architecture remains the target.

Trace each materially affected control and data flow across its full path rather than judging each
changed file in isolation. Architectural problems include, but are not limited to:

- a user interface recreating product behaviour or orchestration that belongs behind the API
  boundary, or structural integrity and access-control rules that belong at the database boundary;
- a consumer duplicating an existing business rule or capability instead of depending on its
  canonical owner, creating another source of truth;
- a layer bypassing a supported contract and reaching into another layer's internal implementation;
- a feature hand-rolling interface behaviour that an established shared component already provides,
  including its interaction and accessibility contract; and
- a local workaround reconstructing another component's responsibilities or private execution
  environment instead of fixing or extending the owning abstraction.

Reuse must fit the full contract. Similar names, markup, or call shapes do not prove that an existing
abstraction has the required semantics, lifecycle, dependencies, failure behaviour, or accessibility.
Do not demand an unsuitable abstraction, speculative reuse, or a broader redesign merely because it
would look more consistent. Trivial duplication does not by itself justify a shared abstraction.

Do not accept a change that spreads a legacy defect, moves a material responsibility into the wrong
layer, duplicates a shared rule, creates another source of truth, or exposes a poor design to new
consumers. Classify the finding by its concrete consequence under the canonical review standard:
block a material architectural inconsistency and report safe, localised debt as non-blocking.
