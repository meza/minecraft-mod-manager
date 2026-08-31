---
name: conventional-commit-message
description: Generate the final Conventional Commit message text by inspecting the intended git commit surface, with staged changes as the default source of truth. Use this whenever the user asks for a commit message, wants staged changes or the working tree summarized into a Conventional Commit, asks what type or scope to use, or needs help turning staged changes into release-note-ready commit text instead of code-level narration.
---

# Conventional Commit Message

## Purpose

Commit messages are release metadata, not a diary of implementation work.

Your job is to inspect the intended commit surface, classify the change correctly, and then write the message through the branch of the decision tree that matches that classification.

The process comes first. The writing stance belongs inside the branch you land in.

## Output Contract

When the user asks for the commit message itself, return the final commit message text and nothing else.

Default to a single-line Conventional Commit subject:

`<type>[optional scope]: <description>`

Return a full commit message only when the change is breaking and the impact cannot be stated responsibly in the subject alone.

Do not return analysis, alternatives, or a walkthrough unless the user explicitly asks for them.

If the user asks only for the type, the scope, or whether the change is breaking, answer that narrower question directly instead of forcing a full commit message.

## Operating Procedure

Follow this process in order. Do not skip ahead. Do not let a later wording choice silently change an earlier classification decision.

Carry this hidden working state through the whole procedure:

- `surface`: the change set you decided to describe
- `type`: the Conventional Commit type chosen from that surface
- `branch`: `external` or `internal`, derived from the chosen type
- `effect_sentence`: the plain-language sentence used only in the external branch

Set each value once when its step is reached. Do not silently replace it later.

### 1. Identify the Commit Surface

Infer the message from the change set most likely intended for the next commit.

Use staged changes as the default source of truth when they exist. For commit message generation, the staged diff is primary unless the user explicitly asks for the full working tree or there are no staged changes at all.

Inspect the surface with lightweight git reads first:

- `git status --short`
- `git diff --staged` when staged changes exist
- `git diff` when nothing is staged

Use recent history only as a tie-breaker for naming consistency. It must not override the current patch.

If staged and unstaged changes point to materially different messages and the user is not clearly asking about the staged set, ask a short clarifying question instead of inventing a blended message.

Tests are evidence about intent, not proof of external impact. Do not infer `fix` or `feat` from tests or internal implementation work alone.

At the end of this step, your working `surface` should be settled.

Before leaving this step, do not invent a summary label for the patch. Do not restate the change as an architecture move, implementation technique, or mechanism description. Move directly from the settled surface to type selection.

### 2. Classify the Change

Choose the type from release meaning, not from effort, size, or how much code moved around.

This step is a strict protocol. Execute it silently and in order:

1. Answer the classification questions internally from the settled surface.
2. If you need surrounding reads, use them only to resolve the boundary question.
3. Do not narrate the patch while deciding.
4. Do not produce a bridge sentence, tentative summary, or comparison between possible types.
5. End the step with only the chosen `type` settled.

From the start of this step until `type` is settled, produce no user-facing prose at all. Tool reads are allowed. Visible assistant text is not.

Default to these Conventional Commit types unless the repo or user clearly uses a different set:

- `feat`
- `fix`
- `ci`
- `docs`
- `style`
- `refactor`
- `test`
- `chore`

Use them this way:

- `feat` for a new user-facing capability or materially expanded behavior
- `fix` for a change to an existing surface that prevents wrong outcomes during normal use
- `ci` for CI, workflow, or pipeline changes whose primary outcome is CI behavior
- `docs` for documentation changes whose primary outcome is documentation
- `style` for formatting-only changes with no behavioral effect
- `refactor` for internal restructuring that preserves the same outcomes at the surface boundary
- `test` for test-only changes
- `chore` for maintenance or project work that does not fit the categories above, is not CI-specific, and does not describe user-facing behavior

Be conservative with `feat` and `fix`. They are release signals.

Answer classification through boundary questions, not through patch narration:

1. Is this changing an existing surface or adding a new one?
2. If it changes an existing surface, what wrong outcome at that surface stops happening after this patch?
3. If you can name that changed outcome in ordinary boundary terms, classify from that outcome.
4. If you cannot name any changed boundary outcome and can only describe internal rearrangement, helper movement, storage mechanics, or defensive technique, do not classify it as `fix`.

Treat "surface" literally as the behavior other parts of the system already rely on. It is not limited to UI changes.

Do not classify something as internal only because the code change sits in storage, model, or data-layer code. Classify from the relied-on behavior of the existing operation that code serves.

Use this gate for `fix` versus internal work:

- choose `fix` when the patch changes the behavior of an existing surface in a way that prevents wrong outcomes during normal use
- choose `refactor` when the patch reorganizes internals while preserving the same outcomes at the surface boundary
- do not demote a real behavioral correction to `refactor` just because the implementation technique is defensive, structural, or cleanup-oriented

Use this stricter check before choosing `refactor`:

- if callers, users, operators, or downstream systems would now get different relied-on behavior from an existing operation, that is not a pure refactor
- if the patch changes an existing operation so it stops producing wrong outcomes, classify it as `fix` even when the code change is mostly protective
- choose `fix` only when that corrected boundary outcome is the primary shipped meaning of the settled surface
- if a patch is primarily extraction, decomposition, centralization, or internal relocation, do not promote it to `fix` just because one helper or code path now fails safer as a subordinate consequence of that internal work
- choose `refactor` only when the same existing operations would still produce the same relied-on outcomes after shipping

If you inspect surrounding code to understand the surface boundary, use that read only to answer the classification gate. Do not turn that extra read into a new patch summary. The classification question is "what outcome changed at the surface boundary," not "what engineering technique is this patch using."

While doing this step, do not write an intermediate description of the patch at all. Go straight from the settled surface to the chosen `type`.

Once the type is chosen, keep it settled. Writing the subject may sharpen the wording, but it must not silently demote or promote the classification.

Treat this step as the classification checkpoint. After you leave it, wording difficulty is not a reason to switch branches. If the plain-language wording is hard, solve the wording problem inside the chosen branch.

The output of this step is the chosen `type` only. Do not produce a prose explanation of the patch here, and do not describe why you are deciding between two types.

If any visible prose about the patch appears before `type` is settled, treat that as a protocol failure. Discard that reasoning and restart classification silently from the settled surface.

At the end of this step, your working `type` should be settled.

### 3. Enter the Matching Writing Branch

After classification, follow the matching branch:

- `feat` and `fix` use the external branch
- `ci`, `docs`, `style`, `refactor`, `test`, and `chore` use the internal branch

Do not reopen classification inside this step unless a reread of the change surface proves the earlier classification was wrong. Difficulty writing a good subject is not proof that the classification was wrong.

At the start of this step, set `branch` directly from the chosen `type`. After that, keep `branch` settled.

From this point on, do not alter `type`. Every rewrite in this step must preserve the chosen type and operate only on wording.

#### External Branch

This branch is for release-signaling changes.

The audience is everyone downstream of the original patch: users, operators, support, release managers, and anyone scanning history to understand what is now different for them.

The job in this branch is translation. You already know the implementation. Your task is to name the experienced effect without carrying implementation or mechanism language into the final subject.

This branch is where release metadata gets written. A subject from this branch may later appear in release notes, changelog summaries, upgrade reviews, and quick history scans by people who did not read the patch.

Do not retreat out of this branch because implementation wording feels easier. If the change was correctly classified as `feat` or `fix`, difficulty finding plain wording is a writing problem inside the branch, not evidence that the classification was wrong.

Do not begin this branch with an implementation summary of the patch. Go straight from the settled surface and settled type to the experienced-effect sentence.

The first sentence you write in this branch must itself be the candidate `effect_sentence`. Do not write any lead-in, rationale, or restatement before it.

Build that first sentence by rewriting the already-chosen external meaning into ordinary language. Do not use this sentence to reconsider whether the change is really `fix` or `feat`; that decision is already settled.

Aim for a sentence with this positive shape:

- it names the affected product or domain thing in ordinary terms
- it states what is now different about that thing after shipping
- it uses a verb that describes the changed consequence, not the engineering method
- it can stand alone for a reader who never saw the patch

If you can write a sentence that cleanly answers "what is safer, fixed, or newly possible now at the boundary someone interacts with?" you are on the right path.

Carry these hidden semantic roles through the external branch:

- `affected_thing`: the domain thing, user-facing thing, operator-facing thing, or bad outcome being discussed
- `changed_consequence`: what is now prevented, corrected, enabled, or otherwise different after shipping
- `actor`: the ordinary-language actor implied by the sentence, if there is one

Set these roles from the plain sentence before compression. Do not silently replace them during compression.

Use this loop exactly:

1. Write one plain-language sentence about what is different after shipping.
2. Make that sentence describe what now works, what stops going wrong, what becomes possible, or what someone needs to know.
3. Write that sentence from the perspective of someone experiencing the problem at the system boundary, not from the perspective of how the code achieves it internally.
4. If the sentence's key nouns or actors only make sense after reading the implementation, discard it and rewrite it in plain language without changing `type`.
5. Set `effect_sentence` to the surviving sentence and keep it settled.
6. Set `affected_thing`, `changed_consequence`, and `actor` from that sentence and keep them settled.
7. Treat `effect_sentence` and those semantic roles as the source of truth for the subject.
8. Compress `effect_sentence` into a Conventional Commit subject by removing detail, not by swapping in more technical wording.
9. If you need a shorter subject, delete words first. Reorder only when it preserves the same everyday nouns and verbs or obvious everyday synonyms.
10. If a shorter draft introduces a mechanism-focused noun or verb that was not needed in `effect_sentence`, discard it and go back to `effect_sentence`.
11. Validate the whole subject shape before scoring individual words.
12. Reject the draft if it changes the role of the head noun, main verb, or qualifier phrase from domain/outcome language into implementation/mechanism language.
13. Reject the draft if it no longer preserves the same `affected_thing`, `changed_consequence`, and `actor` as the settled sentence.
14. Validate noun phrases, not just single words. If a two-word or three-word phrase depends on implementation knowledge, answers "what code changed?", or shifts the subject away from the experienced effect, discard it and rewrite from `effect_sentence`.
15. Score every content word in the remaining subject for jargon proximity.
16. If any content word scores `2` or higher, rewrite from `effect_sentence` and score again.
17. Return only the first subject that passes role, shape, phrase, and word scoring while still naming the experienced effect.

Prefer nouns that a teammate could recognize from product behavior, API behavior, operator workflow, or domain language without opening the patch. Prefer verbs that describe prevention, availability, correctness, or changed capability.

Check phrase roles explicitly:

- the head noun of the subject should name the affected domain thing, user-facing thing, operator-facing thing, or bad outcome
- if the head noun names an internal container, storage unit, local code artifact, or similar implementation-facing construct, the subject failed
- trailing qualifiers should describe the changed consequence or circumstance after shipping
- if a trailing qualifier describes an implementation step, access path, lifecycle step, or engineering method, the subject failed

Check role preservation explicitly:

- the head noun should still point at `affected_thing`
- the main verb should still express `changed_consequence`
- any actor in the subject should still match the ordinary-language `actor`, not a newly introduced implementation actor
- if compression changes any of those roles, the subject failed

If a draft feels more concrete only because it names a component, code path, storage detail, or engineering mechanism, that concreteness is false precision in this branch. Rewrite it back toward the experienced effect.

Compression in this branch is subtractive, not substitutive.

Good compression removes extra detail from the plain-language sentence.
Bad compression replaces the sentence's plain-language framing with implementation-facing framing.

If the plain sentence says what stops going wrong for someone using or operating the system, the final subject should still say that same thing in shorter form.

Use this scale for step 15:

- `0` plain everyday language
- `1` slightly specialized but still natural to a non-technical reader
- `2` mildly technical or software-adjacent
- `3` clear engineering or implementation jargon
- `4` strong implementation terminology
- `5` highly technical term of art

Only `0` and `1` are acceptable in this branch.

Shape beats vocabulary in this branch. A subject with plain-looking words still fails if those words are arranged around implementation knowledge instead of changed outcome.

Treat these as drift signals in the external branch unless the user-facing problem genuinely requires naming them:

- file or function identifiers
- helper or algorithm mechanics
- code structure or local architecture terms
- storage, transport, or lifecycle framing that depends on implementation knowledge rather than experienced behavior
- software shorthand that is natural to engineers but unnatural in release notes

If the subject answers "what code thing changed?" instead of "what is different after shipping?", the branch failed. Rewrite it around the experienced effect.

If the plain sentence and the final subject seem to be about different nouns or different actors, the branch failed. Rewrite from the sentence instead of editing the bad subject.

If the plain sentence itself is built around nouns or actors that only make sense after reading the implementation, the branch failed before compression started. Rewrite the sentence from the experienced problem instead.

Apply the same checks to scope when scope is present. In the external branch, a technically precise scope can still make an otherwise plain subject fail.

If you are unsure whether a candidate sentence is good, prefer the one that a release reader could understand fastest without extra context.

#### Internal Branch

This branch is for changes whose honest meaning is the engineering work itself.

The audience here is maintainers and contributors reading history to understand what kind of work happened. The job is not translation into release-note language. The job is accurate naming without inflating internal work into product impact.

Use this process:

1. Name the work according to the chosen type.
2. Keep the subject concise and truthful.
3. Describe the engineering work itself, not the diff line by line.
4. Do not invent user-facing or operator-facing impact that the patch did not actually create.

Do not alter `type` or `branch` during this step.

### 4. Decide Scope

Scope is optional. Omit it by default.

Add scope only when it provides clear meaning that survives outside the immediate patch context. Good scopes name a stable component, package, or recognizable product area.

Do not manufacture scope from whichever directory changed the most.

In the external branch, scope should usually stay omitted. Do not use technical internals such as storage layers, helpers, or local implementation modules as scope unless that surface is itself a stable user-facing or operator-facing area.

### 5. Decide Whether It Is Breaking

Mark a change as breaking with `!` only when it changes public behavior, interface expectations, or operator workflow in a way that requires action or creates incompatibility.

Do not mark a change as breaking merely because the implementation was rewritten.

When you use `!`, make the impact explicit. If the subject cannot carry that burden alone, return a full commit message with a short explanation of the breaking impact.

## Final Check

Before returning the message, verify:

- you used the intended change surface
- you chose the type from release meaning rather than implementation effort
- you stayed inside the correct branch for that type
- the returned type still matches the settled `type`
- the writing path still matches the settled `branch`
- after classification, no later rewrite changed `type`
- if you were in the external branch, the subject still names the experienced effect from the plain-language sentence
- if you were in the external branch, the subject still comes directly from the settled `effect_sentence`
- if you were in the external branch, every content word in the final subject scored `0` or `1`
- if you were in the external branch, you did not retreat into internal wording or internal scope just because it felt easier to write
- you did not insert an implementation-summary paraphrase between reading the surface and choosing the type or effect sentence
- if you were in the internal branch, the subject names the engineering work honestly without inventing external impact
- scope, if present, adds meaning rather than patch narration
- breaking syntax, if present, reflects a real compatibility break

If any check fails, revise before returning the final message.
