# Architectural fitness

Correct behaviour is necessary but not sufficient. A change must put each responsibility in the right owner, preserve intended dependency boundaries, and use established capabilities at the seam that owns them.

Establish the current architecture from the implementation and configuration. Establish the target architecture from accepted decisions, the nearest applicable guides, and other repository evidence. Treat neighbouring legacy code as current-state evidence, not proof that its architecture remains the target. Record material uncertainty when the target cannot be established; do not invent a target during review.

Trace each materially affected control and data flow across its full path. Consider whether the change:

- moves product behaviour, orchestration, structural integrity, or access control away from its documented owner;
- duplicates a business rule or capability and creates another source of truth;
- bypasses a supported contract to depend on another module's internals;
- recreates behaviour already owned by a suitable shared component; or
- reconstructs another component's private execution environment instead of improving the owning abstraction.

Reuse must fit the full contract. Similar names or call shapes do not establish compatible semantics, lifecycle, dependencies, failure behaviour, or accessibility. Do not recommend an unsuitable abstraction, speculative reuse, or a broad redesign solely for consistency.

When reporting an architectural finding, identify the current responsibility and dependency flow, the evidence for the target boundary, the concrete consequence of the mismatch, and the bounded outcome that restores alignment. Follow the review guide's finding requirements without assigning severity or delivery authority.
