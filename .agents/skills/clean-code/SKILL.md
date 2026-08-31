---
name: clean-code
description: Use when designing, writing, refactoring, reviewing, or explaining code, tests, APIs, modules, or architecture. Provides a progressive catalog of clean-code qualities covering comprehension, domain modeling, state and failure behavior, boundaries, evolvability, operability, safety, and test design.
---

# Clean Code

Clean code makes correct change safe. Its value is not tidiness or conformity for their own sake. It reduces interpretation work, hidden risk, and accidental coupling for the people who must understand, operate, test, and evolve the system.

The qualities in this skill reinforce one another. Clear types can protect invariants. Explicit boundaries can localize change. Realistic tests can make those boundaries trustworthy. A locally elegant choice is not clean when it shifts complexity, risk, or ambiguity elsewhere.

## Find the relevant qualities

Start from the actual decision, code, or failure under consideration. Use the descriptions below to select the concern areas that can materially affect it. Open only those navigation documents first. Each one provides enough context to decide which individual quality references need deeper reading.

- [Comprehension and intent](references/comprehension-and-intent.md) - A reader can build a correct mental model of the code from its naming, structure, documented contracts, and stated assumptions, without reconstructing hidden intent from implementation detail.
- [Domain modeling and data expressiveness](references/domain-modeling-and-data-expressiveness.md) - Types, signatures, and data structures express domain concepts and the invariants that govern them, instead of flattening meaning into primitives, flags, and generic containers that invite misuse.
- [State effects and failure behavior](references/state-effects-and-failure-behavior.md) - Runtime behavior is legible and safe: what mutates and who owns it, what must happen in what order, what enters from outside, and what happens on every failure path.
- [Structure boundaries and dependencies](references/structure-boundaries-and-dependencies.md) - The arrangement of modules reflects real responsibilities: related things live together, variation and shared rules have one owner, and dependencies point the right way through narrow contracts.
- [Evolvability and consistency](references/evolvability-and-consistency.md) - The code can be changed, extended, and deleted safely and cheaply, and it solves problems the way the rest of the codebase already solves them.
- [Operability and production behavior](references/operability-and-production-behavior.md) - The system can be configured, observed, scaled, migrated, and released by people who did not write it, and its signals reflect what is really happening.
- [Security privacy and user harm](references/security-privacy-and-user-harm.md) - Trust boundaries, privileges, secrets, data exposure, auditability, defaults, and user-facing language protect the people the system touches.
- [Test meaning and evidence](references/test-meaning-and-evidence.md) - The tests state and protect meaningful promises about behavior, cover invariants and unhappy paths, and give a reader enough evidence to trust a pass and diagnose a failure.
- [Test placement and isolation](references/test-placement-and-isolation.md) - Each test sits at the cheapest layer that can provide trustworthy evidence and is insulated from the world, so results reflect product behavior rather than environment.

Read more than one concern area when the decision crosses boundaries. Do not load the entire catalog by default.

## Apply judgment

Treat every quality as a reasoning lens, not an automatic finding or universal prohibition. Establish the requirements, target architecture, project conventions, and evidence first. A quality matters when it changes correctness, comprehension, failure risk, operability, safety, or the cost of future change in the current context.

When qualities pull in different directions, prefer the option that protects correctness and safety, keeps responsibilities and dependencies explicit, and owns the least unjustified complexity. Make the trade-off visible. Do not improve one unit by exporting ambiguity or maintenance cost to its callers, tests, operators, or neighboring modules.

This catalog does not authorize edits, broaden a task, replace language-specific guidance, or override project instructions. It sharpens judgment inside the authority and scope already established.

## Communicate the judgment

Explain the relevant quality, the concrete evidence, and the consequence. Distinguish an observed property from a proposed remedy. When several qualities expose one root cause, describe the root cause once rather than multiplying equivalent concerns.

Recommendations should say what boundary or behavior needs to improve and why. Avoid prescribing a pattern merely because its name resembles the problem. Prefer a solution that fits the codebase's target architecture and makes the desired quality structural.

Make uncertainty explicit when the available evidence cannot establish a quality. Missing context is a reason to narrow confidence, not to invent a defect.

## Self-verification

Before handing over work shaped by this skill, verify that:

- concern areas were selected from their descriptions rather than loaded indiscriminately;
- the relevant individual references were read rather than inferred from their link titles;
- facts, assumptions, trade-offs, and unknowns remain distinguishable;
- recommendations address a demonstrated consequence rather than aesthetic preference;
- interacting qualities and downstream effects were considered;
- the result fits the target architecture and project conventions;
- no catalog entry was treated as mutation authority or an inflexible rule.

