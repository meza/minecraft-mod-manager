# Comprehension and intent

A reader can build a correct mental model of the code from its naming, structure, documented contracts, and stated assumptions, without reconstructing hidden intent from implementation detail.

Use the descriptions below to decide which individual quality references are relevant. Open only the references that can materially affect the current decision, code, or failure.

- [Intent is obvious](comprehension-and-intent/intent-is-obvious.md) - A reader can understand what this unit is for without reverse engineering the whole surrounding system.
- [Names are domain-specific and precise](comprehension-and-intent/names-are-domain-specific-and-precise.md) - Names use the language of the domain and distinguish meaning clearly enough that readers do not have to infer intent from surrounding context.
- [Levels of abstraction are aligned](comprehension-and-intent/levels-of-abstraction-are-aligned.md) - A unit of code operates at one conceptual level at a time instead of mixing policy, orchestration, formatting, and low-level detail.
- [Public contracts are documented](comprehension-and-intent/public-contracts-are-documented.md) - Public or easy-to-misuse interfaces explain their guarantees, constraints, error cases, side effects, and expected usage.
- [Comments explain why](comprehension-and-intent/comments-explain-why.md) - Comments justify intent, tradeoffs, or non-obvious constraints rather than narrating what the code already says.
- [Code is easy to follow locally](comprehension-and-intent/code-is-easy-to-follow-locally.md) - Control flow and structure are linear and bounded enough that a reviewer can understand a unit without chasing large amounts of external context.
- [Function shape is cohesive and tractable](comprehension-and-intent/function-shape-is-cohesive-and-tractable.md) - A function has one clear job, stays small enough to understand as a whole, and takes a set of inputs that reflects its true responsibility.
- [Complexity and control flow are justified](comprehension-and-intent/complexity-and-control-flow-are-justified.md) - The solution has no more abstraction, branching, indirection, or special-case handling than the problem actually requires.
- [Reviewers do not need tribal memory](comprehension-and-intent/reviewers-do-not-need-tribal-memory.md) - Correctness does not depend heavily on undocumented norms that only long-time contributors know.
- [Assumptions, intent, and structural decisions are explicit](comprehension-and-intent/assumptions-intent-and-structural-decisions-are-explicit.md) - The code and its surrounding artifacts make assumptions, change intent, and important architectural decisions visible enough that reviewers and future maintainers do not have to reconstruct them from guesswork.

