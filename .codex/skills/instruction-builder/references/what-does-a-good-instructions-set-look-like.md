# What does a good instruction look like?

An instruction set is an architectural document. It defines the cognitive environment an agent will inhabit: the mindset, boundaries, and reasoning discipline that make the agent's behavior clear, reliable, and aligned.

The instruction set does not perform the task. It establishes the conditions under which the task can be performed with clarity and integrity. The craft is the ability to translate user intent into a coherent operational identity—an explicit statement of what an agent is, what it is for, and how it should think while performing its role.

A good instruction set earns authority through precision, not verbosity.

## The Core Design Goal

The goal is coherence.

Coherence means the agent can follow its mission without contradiction, silent prioritization, or invented authority. It also means the instruction set reads as a single, continuous line of thought: purpose establishes direction, boundaries define limits, reasoning discipline governs decisions, and output discipline makes results usable.

Coherence is not achieved by adding more rules. It is achieved by making the role singular, the boundaries explicit, and the reasoning model stable under uncertainty.

## Persona as Part of System-Message Craft

Crafting an instruction set includes defining a persona, because persona is part of the cognitive environment.

Persona is not decoration. It is the framing that establishes tone, stance, and interaction expectations so the agent's role is legible and consistent. A well-defined persona supports the mission by making the agent's communication style predictable and appropriate to the domain.

Persona should remain downstream of purpose and boundaries. It should reinforce what the agent is for, not expand what the agent is allowed to do. When persona becomes a substitute for scope definition, the agent drifts.

An instruction set should speak directly to the agent being defined in clear declarative language, typically of the form "You are X. Your purpose is...". This anchors identity and makes the persona operational rather than implied.

Instruction sets should avoid self-referential or first-person phrasing. The instruction set is not a conversation with the user; it is the agent's blueprint. Direct address keeps identity stable and reduces ambiguity about authority.

## Focus and Scope Integrity

An instruction set should define one agent with one clearly bounded role.

If a request implies conflicting responsibilities or an overloaded mandate, the correct response is to stop and challenge the request. The conflict should be exposed and clarified before drafting proceeds. Without that resolution, the agent will be forced to guess what to prioritize, which undermines reliability.

Ambiguity that would distort an agent's identity, purpose, or reasoning model should not be accepted. The commitment is to coherence: the agent must be able to follow its mission without contradiction, and the instruction set should prevent role creep by design.

## Choosing the Right Level of Abstraction

Default to high-level structural guidance.

An instruction set should define purpose, boundaries, reasoning modes, and communication expectations without slipping into implementation detail. Reasoning agents require frameworks, not scripts. Frameworks generalize across situations; scripts collapse when conditions change.

Operational detail, implementation procedures, and role-specific minutiae should only appear when explicitly requested or when the role cannot be made coherent without them. Excess procedure tends to replace judgment with compliance, which produces brittle behavior and misalignment when reality differs from the imagined workflow.

## A Design Process for Writing System Messages

Crafting an instruction set is a design activity. The process is not bureaucracy; it is how coherence is protected from ambiguity and overload.

### Discovery

Discovery identifies the core purpose, the domain, and the required behaviors of the agent.

The discovery phase exists to prevent distorted identities. It resolves the few uncertainties that would otherwise force the agent to invent authority, broaden scope, or guess what "good" means. Clarifying questions should be asked only when essential. If any ambiguity threatens the coherence or singular mission of the agent, drafting should pause until the ambiguity is resolved.

### Drafting

Drafting produces a structured instruction set addressed directly to the agent.

The draft should provide only the reasoning environment: purpose, scope, tone/persona, boundaries, reasoning practices, and output discipline. Semantic markdown headings should be used to express conceptual hierarchy, not ornamentation. The structure should make the agent's identity and constraints easy to locate and difficult to misinterpret.

Alongside each draft, include a short list of targeted questions that reveal unresolved uncertainties. These questions should be minimal and surgical, aimed at eliminating ambiguity that would destabilize mission coherence.

### Iteration

Iteration refines the instruction set through user feedback while preserving unity.

The instruction set should remain one purpose, one identity, one chain of reasoning. Revision should remove redundancy, tighten causality, and prevent drift into excessive detail. The goal is not to grow the document; it is to stabilize it.

### Delivery

Delivery is a discipline of form.

When the user confirms completion, the output should be the instruction set alone, enclosed in a single fenced code block, with no commentary or meta-notes inside the block. This preserves normative clarity and prevents downstream confusion about what is instruction versus explanation.

## Reasoning Framework to Embed

A good instruction set defines how the agent should think, not what it should think.

This is the core of reliability. The reasoning framework should make it natural for the agent to remain aligned under incomplete information, ambiguity, and pressure to speculate.

### Evidence Discipline

The agent should distinguish between verified information, assumptions, and unknowns.

This distinction should be reflected in both decision-making and communication. When assumptions are necessary, they should be explicitly marked as assumptions and kept minimal.

### Verification

The agent should check conclusions against context, user intent, and available data.

Verification is a guardrail against plausible-sounding output that does not actually follow from what is known. It also reduces misalignment by ensuring the agent's reasoning remains anchored to the user's purpose and the instruction set's constraints.

### Bias Control

Every instruction set imposes design biases, because it privileges certain goals and behaviors.

Those biases should be acknowledged and monitored. The instruction set should require the agent to watch for overreach and unwarranted inference—especially where the role's purpose might tempt the agent into confidence that is not supported by evidence.

### Error Handling

When data are missing or contradictory, the instruction set should require the agent to pause, clarify, or proceed conservatively while marking assumptions.

The key is disciplined uncertainty: the agent should not mask gaps with invented certainty. Conservative proceeding is acceptable only when assumptions are visible and the consequences of error are controlled.

### Reflection

The instruction set should encourage reflection before responding.

Reflection means summarizing the reasoning path in brief, identifying uncertainty, and checking internal consistency. The aim is not verbosity; it is a final alignment check that catches contradiction and scope drift before output is delivered.

## Expression and Structure

Use clear, strictly semantic markdown headings to express conceptual hierarchy.

Write in precise, unembellished language. Authority comes from clarity and causality, not from volume. Avoid examples unless explicitly requested, because examples can accidentally narrow scope or create unintended commitments.

Do not insert first-person statements into instruction sets. Instruction sets are blueprints addressed to the agent; first-person phrasing blurs the locus of authority and invites interpretive drift.

Never introduce unnecessary detail or workflow-level instruction. The default output should be a reasoning environment, not an implementation manual.

## Completion and Self-Audit

Before delivering an instruction set, it should be audited for coherence.

The audit is not a checklist for its own sake; it is a final verification that the document forms one continuous, coherent line of thought and that the agent can follow its mission without contradiction.

A complete instruction set should satisfy the following conditions:

- The agent's purpose is singular and unambiguous.
- The identity, persona/tone, and domain boundaries are coherent.
- The reasoning framework is high-level but complete.
- Verified information, assumptions, and unknowns are treated as distinct categories.
- When data are missing or contradictory, the instruction set requires clarification or conservative proceeding with assumptions marked.
- No first-person phrasing appears in the instruction set.
- No implementation detail was included without explicit request.
- No examlars are included unless they're crucial for procedures.
- The structure is readable, semantic, and internally consistent.
- The instruction set doesn't contain any smart-quotes or other non-ascii punctuation

