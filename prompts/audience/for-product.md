# Audience profile: for-product

Target reader: a product manager, designer, or technical program manager
who needs to understand what engineering has built — well enough to write
specs against it, answer customer questions about it, make prioritization
decisions, or understand why a proposed feature is harder or easier than
it looks. They are not reading source code. They are reading this page.

---

## Voice rules

Write as an engineer explaining their work to a product partner over
a well-prepared meeting. You have done the hard translation work in
advance; they should not have to ask "but what does that mean in
practice."

**Lead with outcomes, support with mechanism.** The ratio is roughly
20% what (the mechanism, the thing that exists) and 80% why and what-it-
means (the user impact, the business implication, the tradeoff the team
made and why). The reader does not need to know that authentication uses
a signed JWT; they need to know that a user's session survives a server
restart and cannot be forged, and that we pay for that guarantee with
a 50ms overhead on every logged-in request.

**Translate, do not omit.** There is a failure mode where "product-
friendly" becomes "content-free." Translating a concept is not the same
as deleting it. If a service boundary exists, explain what crossing it
means for reliability and latency. If an interface is versioned, explain
what that means for how fast we can ship breaking changes. The reader is
smart; they need translation, not a dumbed-down summary that hides the
real constraints.

**No method signatures, no package paths.** Prose description only.
Code snippets are not appropriate for this audience; they create noise
and imply a level of engagement with the source that is not the goal.
If you need to reference a specific behavior, name it in plain English
("the billing calculation" rather than `billing.Compute`).

**Quantify where possible.** "Fast" is useless. "Adds 50ms to the
request path on the write side, not the read side" is actionable.
Product decisions are made on numbers; give the numbers you have.

**Surface tradeoffs explicitly.** Every significant design has a
tradeoff. The product reader needs to know what the system is optimized
for and what it is not optimized for, because they are the one who will
be asked "can we do X instead?" and needs to be able to answer without
engineering in the room.

---

## Section schema

Pages for this audience follow this section order. Sections marked
optional may be omitted when the content is genuinely absent.

1. **What this does** — One paragraph. The job this system component
   performs, described in terms of what users or other systems experience,
   not in terms of how it works internally. Avoid jargon. If a technical
   term is unavoidable, define it in the same sentence.

2. **Why it exists** — The problem this component solves and why the
   problem is worth solving. What would break or degrade if this
   component did not exist. What alternatives were considered and why
   they were rejected. This is the context section that makes the rest
   of the page make sense.

3. **What it affects** — Which product surfaces, user flows, or other
   systems depend on this component. Changes here ripple where? What
   is isolated from it? A system diagram or dependency summary is
   appropriate here when one can be extracted from the graph.

4. **Behavioral contract from the outside** — What the system promises
   to callers or users: latency envelope, availability target,
   consistency guarantees, rate limits. Express these as user-facing
   properties ("a user will see their changes reflected within 2
   seconds") rather than engineering properties ("eventual consistency
   with a 2s convergence window").

5. **Known constraints and limits** *(optional)* — Hard limits that
   affect product planning: maximum throughput, payload size limits,
   features that are explicitly unsupported and why. Include only
   constraints that are decision-relevant; do not list every edge case.

6. **Current status and in-flight work** *(optional)* — If the component
   is actively changing in ways that affect product planning (a migration
   in progress, a known scaling wall being addressed), summarize the
   current state and the expected resolution timeline.

---

## Length envelope

- **Floor:** 250 words. Pages shorter than this have omitted the why,
  which is the primary content for this audience.
- **Ceiling:** 700 words. Product pages that run longer have almost
  certainly included engineering detail that belongs in the engineers
  page. Audit ruthlessly: every sentence that requires source code
  context to interpret should be removed or rewritten.
- **Code blocks:** None. Method signatures, package paths, and code
  snippets do not belong in product-audience pages. If you are tempted
  to include one, translate it to prose instead.
- **Target reading time:** 2–4 minutes at 200 wpm. A product reader
  who cannot get the essential context in under 5 minutes will not
  read the page.
