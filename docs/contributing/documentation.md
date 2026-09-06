# Documentation contributions

Start with the reader's task and the contract the document owns. Product users need purpose,
inputs, outcomes and recovery guidance. Contributors need the applicable procedure, prerequisites,
authority and verification. Update the owning document rather than adding a competing explanation
to README for every change.

## Choose the owner

| Content | Owner |
| --- | --- |
| Product purpose and user entry | Root README |
| Desired behavior and acceptance | [Product intent](../intent.md) |
| Shared vocabulary | [Glossary](../GLOSSARY.md) |
| Command inputs, workflows and outcomes | [Command guides](../commands/README.md) |
| Contribution procedure and verification | [CONTRIBUTING](../../CONTRIBUTING.md) and its selected contributor guides |
| Local responsibilities and contracts | README or focused documentation beside the affected package |
| Enduring architectural decisions | [Decision records](../../doc/adr/decisions.md) |

Use clear language consistent with the surrounding documentation. Explain prerequisites before
commands, connect actions to expected outcomes, and make consequential failure or recovery behavior
explicit. Examples must demonstrate the contract described, including any context they need.
For source comments, follow the [comment policy](code-comments.md).

Distinguish a desired target from a claim about the current product. Do not present conceptual
examples, historical records or acceptance requirements as proof that the implementation conforms.
Preserve explicit historical boundaries instead of modernizing historical evidence into an active
instruction.

## Instruction and contribution guidance

Keep common contribution rules in the root guide and detailed procedures with their scoped owners.
Provide a task-based link from the entry point to each necessary specialist. Local READMEs and
CONTRIBUTING files remain applicable within their directory trees.

When consolidating instructions, preserve each applicable trigger, owner, authority, condition,
exception and verification duty. A moved rule needs both a concrete destination and a required
route from every entry that previously supplied it. Keep a local invariant when its separate entry
context needs it; do not copy whole procedures merely to make them easier to discover.

Check conflicts in a reachable task context. If two applicable requirements demand incompatible
outcomes and their authority does not resolve the conflict, surface the decision before adopting
replacement wording. Do not silently narrow a requirement for brevity or create a new approval
step merely by reorganizing the documents.

## Verification

For documentation-only work, inspect content, terminology, examples and related contracts; check
changed links and anchors; and follow the reader's path from its entry point. A reader should be
able to identify the relevant owner, action, expected result and completion evidence without
reconstructing the author's conversation.

For contribution routing changes, walk both human and agent entry paths and confirm that required
specialist rules remain reachable. Account for deliberate removals and replacements during the
change, without adding a permanent audit workflow to contributor documentation.

Do not require a production test or application build to prove prose or navigation. Mixed changes
also follow the [verification policy](../../CONTRIBUTING.md#verification) for their other surfaces.
An independent documentation reader checks clarity and internal coherence using the documents and
their referenced documentation; report claims or prerequisites that remain unclear.
