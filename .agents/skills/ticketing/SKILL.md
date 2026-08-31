---
name: ticketing
description: Use when asked to write, draft, create, or refine a ticket/issue, or to define work and how it will be verified.
---

# Writing Good Tickets

## Purpose

A ticket is a coordination artifact. It turns intent into something that can be executed, reviewed, and verified by someone who is not inside your head.
It is how work survives across people and across time.

When tickets are weak, teams do not just lose time. They lose coherence.
Implementers fill gaps by inventing constraints, inferring priorities, and choosing trade offs without authority.
That is how drift happens. The work technically ships, but it disagrees with what the author meant, what the reviewer expects, or what the system can safely support.

## World model

Ticket writing is not documentation for its own sake. It is risk management.
The risk is not that the implementer is incompetent. The risk is that the implementer is competent and will make reasonable decisions in a world where key constraints are invisible.

The author has a mental model that includes motivation, constraints, and hidden assumptions.
The reader has a blank page, a schedule, and the need to make progress.
In that gap, the reader will choose a path that looks correct. If the ticket does not make the forces visible, the path will often be wrong in ways that are expensive to discover late.

Good tickets contain uncertainty instead of pretending it does not exist.
They do not eliminate unknowns. They name them, bound them, and attach a way to resolve them.

## What good looks like

A good ticket makes it hard to misread what success is.
It makes it hard to build the wrong thing while still feeling correct.
It makes review faster because the reviewer can check the result against intent rather than reconstruct intent from diffs.
It makes delivery safer because verification and rollback are considered as part of the work, not as an afterthought.

Impact is part of correctness.
If the ticket does not say what changes for the user or the business when the work is done, the reader cannot tell what is essential versus incidental.
That is how tickets turn into implementation exercises that miss the point.
Stating impact is not motivation writing. It is how you define what the work is protecting and why trade offs should be made in one direction rather than another.

A good ticket also makes change safety real.
Most systems fail during change, not during steady state.
If the ticket does not include a view of rollout, rollback, and blast radius, the work can look correct in isolation and still be unsafe to ship.
Change safety does not always require a plan, but it always requires intent: how you will limit harm if reality disagrees with the design.

A good ticket also respects the systems complexity budget.
It does not smuggle in broad refactors, speculative infrastructure, or unrelated cleanup.
When there is real complexity, the ticket explains why owning that complexity is worth it.

## What the ticket must preserve

A ticket does not need to be long. It needs to be complete in the specific way that prevents guesswork.
It must preserve the problem being solved and for whom, why it matters now, what success looks like, which constraints are real and non negotiable, and how success will be verified.

Everything else exists only to make those answers unambiguous and actionable.

## References and reading

Tickets often depend on existing documentation: product notes, architecture decisions, design files, standards, runbooks, etc.
Links are valuable only when the reader can tell what they mean.
A pile of links forces the reader to guess which documents are normative, which are background, and whether critical requirements are hidden off-ticket.
That guesswork recreates the original problem: invisible constraints and late discovery.

When you link to reference material, you are not just providing convenience. You are defining authority.
If the ticket depends on external requirements, the ticket must say so explicitly, and it must be clear what the reader is expected to learn from the linked material.

## Operating procedure: link discipline

This is an operating procedure because it prevents a common failure mode: requirements that live only in external documents, discovered late and interpreted inconsistently.

1. Link only to material that actually matters to the work, because irrelevant links add noise and reduce trust.
2. For each link, state whether reading is required or optional, because readers need to know what is a prerequisite versus supporting context.
3. For required links, state what requirements or constraints must be understood, because the ticket must not hide normative rules behind a URL.
4. If a linked document contains additional requirements that apply to the work, summarize the relevant ones in the ticket or quote the exact section, because the ticket must make obligations visible without forcing archaeology.

## Operating procedure: draft a ticket

This is an operating procedure because it is the minimal mechanism that prevents the most common failure mode: the author assumes context that the reader does not have.
Each step exists to remove a specific kind of guesswork.

1. State intent in plain language, because a ticket that begins with implementation details has already lost the real goal.
2. State impact and why it matters now, because without impact the reader cannot tell what is essential and will optimize for the wrong thing.
3. Define success as observable outcomes, because vague success creates divergent implementations that are all arguably correct.
4. State constraints and non goals, because boundaries are what keep work small and keep teams aligned on what will not be done.
5. Name assumptions and unknowns, because uncertainty that is not written down becomes an implicit requirement later.
6. Describe the expected shape of the change, because the implementer needs a mental model of impact, but should still have room to apply judgment.
7. Define verification, because work that cannot be verified cannot be confidently reviewed, shipped, or rolled back.
8. State change safety intent, because most harm happens during rollout and the ticket must not assume shipping is automatically safe.
9. Record risks and dependencies, because hidden dependencies are a primary driver of stalled work and surprise coordination.

## Operating procedure: refine a ticket

This is an operating procedure because tickets decay as context changes.
Refinement keeps the ticket aligned with reality before someone spends time building the wrong thing.

1. Re-read the ticket as if you did not write it, because this is the fastest way to detect missing context and ambiguous language.
2. Identify where the ticket invites invention, because invention is how invisible requirements appear.
3. Reduce scope until the ticket describes one coherent change, because multi-goal tickets create review friction and make rollback unsafe.
4. Check that verification is still possible, because tickets often rot by losing the ability to prove success.
5. Update assumptions, risks, and dependencies, because these are the first things to drift when plans change.

## Operating procedure: concern scan

Most failed tickets fail in predictable ways. They omit a cross cutting concern that was obvious to someone, but not written down.
The omission is rarely malicious. It is a consequence of attention: when you are focused on the primary change, second order concerns disappear until they show up as bugs, production incidents, or painful review feedback.

This concern scan exists to make those omissions deliberate. For each concern that matters in your environment, you make an explicit choice for the ticket: either it applies and you capture what is needed, or it does not apply and you say so.
The point is not to create paperwork. The point is to prevent silent guessing and late discovery.
These questions are reminders, not a universal checklist. Use them to prompt the right conversations in the context of your project, and adapt them when your domain uses different language or constraints.

This scan is for the author. The ticket should only surface concerns that apply.
If a concern does not apply, do not mention it. If it applies, include it in the ticket so it survives beyond the moment it was written.
If you cannot tell whether it applies, treat that uncertainty as a blocking question and resolve it before implementation.

1. Determine whether the concern applies to this piece of work, because applicability is what creates real requirements.
2. If it applies, add a section for that concern in the ticket, because cross cutting requirements must be visible where work is coordinated.
3. In that section, repeat each question and answer it in the context of the current requirements, because the ticket must preserve intent, constraints, and verification.

### Content requirements

* Are the assets ready?
* Is the content coming from the correct source?
* Who owns the content?
* How does the content get updated if it changes? (i.e., CMS, hardcoded, etc)

### Accessibility requirements

* Will accessibility tests pass if we implement the work as currently specified?
* Will the screen reader experience make sense if we implement the work as currently specified?
* Will the feature be usable with keyboard only if we implement the work as currently specified?
* Are the current requirements aligned with ARIA best practices? https://www.w3.org/WAI/ARIA/apg

### Analytics and telemetry requirements

* Do we have any new analytics/telemetry events to track?
* Have we defined the events to be tracked?
* Do we know how we are going to use the data?
* Are we tracking the right things?

### SEO requirements (web projects only)

* Any new pages have proper meta titles and descriptions that need to be added?
* Do we know the target keywords for the new content?
* Are we following best practices for on-page SEO? (header structure, alt text, etc)
* Are there any new URLs? If so, are they SEO-friendly?
* Are we using proper canonical tags to avoid duplicate content issues?

### Security requirements

* Have we mitigated against potential security risks?
* Are there any new dependencies? If so, have they been vetted?
* Are there any secrets to be added? If so, are they stored securely?
* Are we handling user data correctly? (storage, transmission, encryption, etc)
* Are we compliant with relevant regulations? (GDPR, CCPA, HIPAA, etc)
* Are there any new permissions required? If so, are they justified and documented?
* Are we following best practices for authentication and authorization?

### Feature flags

* Do we need feature flags for something?

### UX requirements

* Are the UX requirements clearly defined?
* Are there wireframes or mockups to follow?
* Are there any user flows to consider?
* Are there any edge cases to consider?

### Design requirements

* Are the design requirements clearly defined?
* Are there design assets to follow?
* Are we following the design system? (if applicable)
* Are there any animations or interactions to consider?
* Are we considering responsive design? (if applicable)
* Are we considering dark mode? (if applicable)
* Are we considering internationalization/localization? (if applicable)
* Are we considering branding guidelines?
* Are we considering print styles? (if applicable)

### Testing requirements

* Aside from the evident, what other tests should we be considering?
* What are the happy paths?
* What are the edge cases?
* Anything we know we need to be especially careful about?
* Is there anything that might be flaky?
* How are we making sure that we don't have regressions?
* Do we have confidence that our users can use the thing?

### Performance requirements

* Is there a performance budget?
* Do we foresee any performance bottlenecks?
* Are there any lazy loading opportunities?
* Are we following best practices for performance? (caching, compression, etc)
* Do we need to add anything new to our existing performance monitoring?

### Observability requirements

* How & when do we get notified about potential production errors?
* Are the correct metrics & thresholds understood and defined?
* Do we need to add any new metrics, logs, or traces?
* Are we following best practices for observability? (structured logging, correlation IDs, etc)
* Is there anything new we need to add runbooks for?
* Are there any new dashboards to be created or updated?
* Are there any new alerts to be created or updated?
* Are we considering SLOs/SLIs for this functionality?
* Are we considering error budgets for this functionality?

### Documentation requirements

* Do we need to update any architecture diagrams?
* Do we need to update any API documentation?
* Do we need to update any user guides or manuals?
* Do we need to update any onboarding materials?
* Do we need to create any new documentation?
* Are the business requirements properly documented and up-to-date?

## Common failure modes

One failure mode is writing a ticket that is really a conversation. It reads as a vague intention and expects the implementer to discover the real requirements during implementation. This feels efficient at creation time and is expensive later.

Another failure mode is writing a ticket that is really a solution. It hard codes an approach without stating the forces that justified it. This forces implementers to follow a brittle plan even when reality contradicts it.

Another failure mode is scope smuggling. The ticket claims to do one thing but contains a long tail of implied refactors, cleanup, and new infrastructure. This breaks estimation, breaks review, and breaks trust.
