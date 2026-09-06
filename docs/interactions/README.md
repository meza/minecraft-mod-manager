# Working with terminal interactions

This corpus helps contributors implement and assess shared terminal capabilities. It describes target architecture and presentation, not a claim that the current implementation conforms.

[Product intent](../intent.md) defines requirements. [Command guides](../commands/README.md) describe inputs, workflows and outcomes.

Use the canonical [execution-mode matrix](../intent.md#execution-modes-and-operator-intent) for prompting policy and terminal capabilities. Across those profiles, the [permanent transcript](../intent.md#active-display-and-permanent-transcript) shares durable result and decision text; rich controls and progress are temporary presentation.

| Task | Guide |
| --- | --- |
| Design ownership, rendering, history, scrolling or cancellation | [Terminal implementation guide](../guide-to-working-with-the-terminal.md) |
| Apply consistent controls, progress, language and accessibility | [Interaction conventions](interaction-guidelines.md) |
| Discuss active frames and durable records visually | [Component examples](component-examples.md) |
| Run or extend real-process terminal verification | [E2E testing guide](../testing/README.md) |
| Express operator journeys as acceptance scenarios | [BDD guide](../testing/bdd.md) |
