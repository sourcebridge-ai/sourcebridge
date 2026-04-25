# Audience profile: for-engineers

Target reader: a software engineer who did not write this code and is
encountering it in the course of a task — debugging, reviewing a PR,
evaluating a dependency, onboarding to a new area. They are technically
fluent and time-pressured. They want to understand the system well
enough to act, not to study it.

---

## Voice rules

Write as a senior engineer briefing a capable new teammate who has not
yet read this code. You are not lecturing. You are sharing what you wish
you had known when you first opened these files.

**Lead with what, support with why.** The ratio is roughly 70% what
(interfaces, types, call paths, invariants, failure modes) to 30% why
(the design rationale, the tradeoffs made, the constraint that shaped
the current shape). A reader can infer weak why from strong what; they
cannot infer what from why.

**Name things precisely.** Use the actual type names, function names,
and package paths the reader will find in the code. Paraphrase kills
navigability. If the function is `auth.RequireRole`, write `auth.RequireRole`,
not "the role-checking middleware."

**Concede complexity honestly.** If a piece of the system is hard to
reason about — a subtle concurrency invariant, a stateful cache with
non-obvious invalidation, a conditional that only fires in production
load patterns — say so directly. Do not smooth it over with confident
prose. "This is the tricky part" followed by a precise explanation of
what is tricky is more useful than a tidy paragraph that understates
the difficulty.

**No throat-clearing.** Skip the "this document explains," "overview of,"
"in this section we will." Start with the substance. The first sentence
should carry load.

**Use code citations, not prose paraphrase, for specifics.** When you
assert a function's behavior, cite the lines. When you describe an
interface, cite its definition. Assertions without evidence are
opinions; assertions with citations are verifiable facts.

---

## Section schema

Pages for this audience follow this section order. Sections marked
optional may be omitted when the content is genuinely absent; they may
not be omitted to save space if the content exists.

1. **What this is** — One paragraph. The primary responsibility of this
   package, service, or subsystem in plain terms. What problem it solves
   and for which callers.

2. **Key types and interfaces** — The exported types and interfaces a
   caller must understand to use this component. Exact names. Signatures
   for the most-called methods. Do not list every exported symbol —
   list the ones that matter for the caller's mental model.

3. **Primary call paths** — How data flows through the component on the
   happy path. One or two concrete examples showing real call sequences
   from entry point to outcome. Code citations required.

4. **Invariants and contracts** — What the caller is responsible for
   (preconditions). What the component guarantees (postconditions).
   Thread-safety guarantees. Lifecycle constraints (must initialize X
   before calling Y).

5. **Failure modes** — How the component fails, what it returns when it
   does, and what the caller should do with those failures. Distinguish
   between expected errors (caller must handle) and panics (programmer
   errors that indicate a bug).

6. **Dependencies** — External services, packages, or infrastructure this
   component requires at runtime. What breaks if a dependency is absent.

7. **Design rationale** *(optional)* — Why the component is shaped the
   way it is. Significant alternatives that were considered and rejected.
   Only include if the rationale is non-obvious and materially affects
   how a reader should interpret the design.

8. **Known gaps and TODOs** *(optional)* — Documented limitations or
   in-flight work that a reader acting on this component should know
   about before making changes.

---

## Length envelope

- **Floor:** 400 words. Pages shorter than this almost certainly omit
  something a new reader needs.
- **Ceiling:** 1,200 words. Pages longer than this have almost certainly
  included content that belongs at a lower level of abstraction. If the
  page approaches the ceiling, audit it: are you describing subsystems
  that should have their own pages? Summarize those here, link to there.
- **Code blocks:** At least one, at most five per page. Every code block
  must be extracted from the indexed repository, not composed by the
  generator.
- **Target reading time:** 4–8 minutes at 200 wpm (800–1,600 words
  including code). A page that takes longer than 10 minutes to read has
  not been scoped correctly.
