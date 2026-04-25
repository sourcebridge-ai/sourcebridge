# Audience profile: for-operators

Target reader: an SRE, platform engineer, or on-call responder who is
either setting up this component in production or is reading this page
at 2am because something is wrong. Their time is more constrained than
any other reader's. The page must be scannable under pressure. The most
important content must be visible without scrolling or reconstructing
context.

---

## Voice rules

Write as the engineer who built this component writing the runbook entry
they wish they had had the last time it paged them at 3am.

**Put the decision-critical content first.** An operator reading this
page during an incident is not reading linearly; they are scanning for
the line that tells them what to check. Structure every section so the
actionable content appears before the explanatory content. "Check X
first" before "X works by..."

**Distinguish between observability surfaces and operational levers.**
Observability (what to look at) and levers (what you can change) are
different categories of information that operators need at different
moments. Do not mix them in the same prose block. Structure them as
distinct items.

**Failure modes are the primary content.** This audience needs a
complete enumeration of how the component fails, not just a note that
it "might have issues." For each failure mode: what does it look like
in logs or metrics, what caused it, what is the recovery action,
and what is the blast radius if you do nothing.

**Name the actual metrics, log fields, and alert names.** An operator
who reads "check the error rate metric" cannot do anything useful with
that. An operator who reads "look for `grpc_server_handled_total`
with `grpc_code != OK` and filter to `grpc_service = 'qa.QAService'`"
can open their dashboard immediately. Specificity is the entire value
of this section.

**Distinguish recoverable from catastrophic.** Flag clearly: which
failure modes are self-recovering, which require operator action, and
which require an engineer escalation because the recovery procedure
requires code-level access or decisions an operator cannot make alone.

**No implementation detail that does not affect operations.** The
operator audience does not need to know how the component is
internally structured; they need to know what breaks, what to look at,
what to do, and who to call. Omit everything else.

---

## Section schema

Pages for this audience follow this section order. Sections marked
optional may be omitted when the content is genuinely absent; they may
not be omitted to save space if the content exists.

1. **What this runs as** — Deployment unit (container, service, daemon),
   resource requirements (CPU/memory envelope under normal and peak load),
   dependencies required at startup (databases, queues, external services),
   and where the process logs to.

2. **Observability surfaces** — In this order:
   - Key metrics: the 3–5 metrics an operator should have on a dashboard,
     with exact metric names and the threshold that constitutes a problem.
   - Log signals: the log fields or patterns that indicate the component
     is degraded. Include the log level and any structured field names.
   - Traces: if distributed tracing is instrumented, the span name or
     service name to filter on.
   - Alerts: any alerts that fire from this component, what condition
     triggers them, and their expected urgency.

3. **Failure modes** — A structured list. For each mode:
   - **Symptom:** what an operator sees (alert fires, metric spikes,
     log pattern repeats).
   - **Likely cause:** the most common root causes, ordered by
     probability.
   - **Recovery action:** the exact steps to take, in order.
   - **Blast radius:** what is affected if this failure mode is not
     addressed immediately.
   - **Escalation:** when to page an engineer rather than attempting
     self-recovery.

4. **Operational levers** — Configuration knobs that an operator can
   adjust at runtime (feature flags, rate limits, queue depths, timeouts).
   For each lever: the name of the config key or environment variable,
   the valid range, the default, and the expected effect of changing it.

5. **Runbook entry points** — Links to detailed runbooks for the most
   common incident scenarios. If runbooks do not exist yet, this section
   should note their absence explicitly — "no runbook exists for X; the
   recovery procedure is Y" is better than silence.

6. **Dependencies and shared infrastructure** — Other services or
   infrastructure this component depends on at runtime. For each
   dependency: what happens to this component when the dependency is
   slow, unavailable, or returning errors. Is there a fallback path?
   What degrades gracefully versus what fails hard?

7. **Deployment notes** *(optional)* — Anything an operator needs to
   know that is specific to deploying a new version: migration steps,
   required ordering of deployments, rollback procedure, known transient
   errors during rollout.

---

## Length envelope

- **Floor:** 350 words. A page without a complete failure-modes section
  is not a useful operator page — that section alone should account for
  most of the floor.
- **Ceiling:** 1,000 words. Operator pages that run longer are probably
  covering too many distinct components. If the page approaches the
  ceiling, consider splitting by subsystem.
- **Code blocks:** Appropriate and encouraged for command-line procedures,
  log filter examples, and metric query examples. Not appropriate for
  implementation code.
- **Scannability:** Every failure mode and every operational lever must
  be identifiable by scanning the heading structure alone. An operator
  who needs to find "what do I do when the gRPC server starts returning
  RESOURCE_EXHAUSTED" should be able to find the relevant subsection
  in under 10 seconds.
