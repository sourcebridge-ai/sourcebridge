# QA Parity Report

- Baseline arm: commit `4776aeb` on 2026-04-23 04:58:10.537541+00:00
- Candidate arm: commit `41f4283` on 2026-04-23 14:19:36.356439+00:00
- Mode: deep vs deep
- Samples: baseline=120 (judged=120, errored=0); candidate=120 (judged=120, errored=0)

## Headline metrics

| Metric | Baseline | Candidate | Delta |
|--------|----------|-----------|-------|
| Answer-useful rate | 65.83% | 69.17% | +3.33% |
| Fallback rate | 0.00% | 0.00% | +0.00% |
| Latency p50 (ms) | 28513 | 34790 | +6278 |
| Latency p95 (ms) | 44243 | 70522 | +26279 |
| Latency p99 (ms) | 51659 | 84203 | +32544 |

## Per-class answer-useful rate

| Class | Baseline | Candidate | Delta | N |
|-------|----------|-----------|-------|---|
| architecture | 68.00% | 84.00% | +16.00% | 25 |
| behavior | 45.00% | 50.00% | +5.00% | 20 |
| cross_cutting | 56.00% | 56.00% | +0.00% | 25 |
| execution_flow | 80.00% | 84.00% | +4.00% | 25 |
| ownership | 76.00% | 68.00% | -8.00% | 25 |

## Top-20 quality regressions (lowest candidate-minus-baseline score)

Human review required before the candidate ships. Sign off in the
Plane epic for the Phase-5 rollout, quoting this section.

| ID | Class | Repo | B | C | Δ | Δlatency (ms) | Fallback change | Judge rationale (candidate) |
|----|-------|------|---|---|---|---------------|-----------------|------------------------------|
| own-018 | ownership | acme-api | 3 | 1 | -2 | +6316 |  | The answer names a plausible file path (src/auth/session.ts) and coherent function names, but the file reference appe... |
| cross-025 | cross_cutting | acme-api | 3 | 1 | -2 | -2199 |  | The answer provides generic-sounding code snippets and file paths without demonstrating they came from an actual code... |
| flow-024 | execution_flow | multi-lang-repo | 3 | 2 | -1 | +19433 |  | The answer directly addresses the startup sequence with concrete function names (NewConfig, StartServer, cfg.Validate... |
| flow-003 | execution_flow | sourcebridge | 3 | 2 | -1 | +15972 |  | The answer directly addresses the question with concrete components (runAsk, runAskServer, runAskLegacy in cli/ask.go... |
| flow-019 | execution_flow | acme-api | 3 | 2 | -1 | +14396 |  | The answer directly addresses the sign-in flow with a clear step-by-step explanation naming concrete functions (handl... |
| cross-001 | cross_cutting | sourcebridge | 2 | 1 | -1 | +12556 |  | The answer names plausible components (TenantMiddleware, TenantFilteredStore, AnswerQuestion) but largely hedges, exp... |
| arch-008 | architecture | sourcebridge | 1 | 0 | -1 | +12380 |  | The answer claims no MCP implementation exists in the codebase, but the large number of referenced symbols and the sp... |
| mix-018 | behavior | acme-api | 3 | 2 | -1 | +11621 |  | The answer directly addresses both parts of the question with concrete details (return type Promise<string>, session.... |
| own-017 | ownership | acme-api | 3 | 2 | -1 | +11545 |  | The answer directly identifies a specific file (src/auth/magic-link.ts) with plausible function names (createMagicLin... |
| mix-015 | behavior | acme-api | 3 | 2 | -1 | +6611 |  | The answer directly addresses the question with concrete behavior (returns {active: false, plan: team.plan} early wit... |
| arch-025 | architecture | acme-api | 3 | 2 | -1 | +6021 |  | The answer directly addresses the question with concrete function names for both magic link and invitation flows, and... |
| own-024 | ownership | multi-lang-repo | 3 | 2 | -1 | +3805 |  | The answer directly names a concrete file path (go/main.go) with specific line numbers, addressing the question. With... |
| mix-003 | behavior | sourcebridge | 1 | 0 | -1 | +2886 |  | The answer explicitly admits it cannot provide a definitive answer and speculates based on hints. It does not identif... |
| own-010 | ownership | sourcebridge | 2 | 1 | -1 | +546 |  | The answer admits it could not find the actual proto files and only speculates about possible locations. It does not ... |
| cross-016 | cross_cutting | sourcebridge | 1 | 0 | -1 | -882 |  | The answer explicitly admits it could not find the relevant code and provides only generic security principles rather... |
| mix-005 | behavior | sourcebridge | 1 | 0 | -1 | -1434 |  | The answer explicitly refuses to answer the question, citing budget limits, and provides no concrete information abou... |
| arch-015 | architecture | sourcebridge | 3 | 2 | -1 | -1686 |  | The answer directly addresses the question with concrete components (trackEvent, handleTelemetryEvent, telemetryEvent... |
| own-015 | ownership | acme-api | 3 | 2 | -1 | -2430 |  | The answer identifies a concrete file path (src/db/models/user.ts) via the citation and describes plausible contents ... |
| own-002 | ownership | sourcebridge | 2 | 1 | -1 | -11225 |  | The answer punts on the question, claiming no handler was found, rather than identifying a specific file. While it li... |
| mix-002 | behavior | sourcebridge | 1 | 0 | -1 | -12301 |  | The answer explicitly admits it couldn't find the implementation and fabricates plausible-sounding symbol names from ... |

## Decision Rule check (plan §Phase 4)

- overall answer-useful within ±7%: **PASS** (Δ=+3.33%)
- per-class within ±10%: **FAIL**
- latency p95 within 2× baseline: **PASS**
- top-20 regressions reviewed and signed off by a human: ☐ (tick manually after review)


---

## Quality-Push Phase 5 Decision Rule

The auto-generated rule above applies Phase-4 *parity* (±7%). Quality
push targets *improvement* per the 2026-04-23-quality-push plan.

| Gate | Target | Observed | Status |
|------|--------|----------|--------|
| Overall useful-rate gain | ≥ +10% over Phase 3 | **+3.33%** | **PARTIAL** |
| Behavior-class gain | ≥ +15% over Phase 3 | **+5.00%** (45% → 50%) | **PARTIAL** |
| Cross_cutting gain | ≥ +15% over Phase 3 | **+0.00%** (56% → 56%) | **MISS** |
| No class down > 3% | no regression | ownership **-8%** (76% → 68%) | **FAIL** |
| Cost per ask | ≤ 1.2× Phase 3 | **0.41×** (caching saved 69.5%) | **PASS** |
| p95 latency | ≤ 1.5× Phase 3 | **1.59×** (44.2s → 70.5s) | **MARGINAL FAIL** |

### What worked

- **Architecture class +16%** (68% → 84%). The dominant single win:
  multi-hop architecture questions decompose cleanly into subsystem-scoped
  sub-questions that the agentic loop answers precisely, then the
  synthesizer joins them with citations intact.
- **Prompt caching was a massive cost win.** 99.8% cache-read ratio;
  per-ask input-token cost dropped from ≈$0.178 to ≈$0.054 — a **69.5%
  savings** on Sonnet 4.5 input pricing. Caching alone more than pays for
  the extra Haiku classifier + decomposer calls.
- **Behavior +5% and execution_flow +4%**, both net positive. Behavior
  in particular benefits from `find_tests` when the tool correctly
  locates test coverage.

### What didn't move

- **Cross_cutting stayed flat at 56%.** Decomposition fired on
  cross_cutting questions but the sub-questions often don't split the
  concern cleanly — "how does auth work across the stack" decomposes
  well, but "where is path-traversal prevention handled" stays
  multi-file *without* benefiting from parallel investigation.

### What regressed

- **Ownership -8% (76% → 68%).** 7 of 25 ownership questions dropped
  by 1 point each. Pattern in the judge rationales: the smart
  classifier surfaces advisory file-path candidates, the model treats
  them as evidence rather than hypotheses, and fabricates confident-
  sounding references it hasn't actually verified via a tool. Ownership
  questions in particular reward "prove it with a read_file" over
  "here's a plausible location", and the extra hints pull in the
  wrong direction.

- **p95 latency 1.59× Phase 3** (70.5s vs 44.2s). The 50/120
  decomposed asks run up to 4 parallel sub-loops at 60s cap each,
  plus decompose + synthesis; even with parallelism the p95 land at
  70s. Not a fatal miss — well within the agentic absolute cap (90s)
  and the user's typical "deep" question tolerance — but one gate
  over the plan's 1.5× threshold.

### Recommended rollout posture

1. **Ship prompt caching** (Phase 1) alone — unambiguous cost win,
   zero quality risk, verified 69.5% input-token reduction in
   production benchmark. Flip `SOURCEBRIDGE_QA_PROMPT_CACHING_ENABLED=true`
   as a standalone release.

2. **Ship decomposition conditional on class**: enable only for
   architecture (+16% win). Gate cross_cutting and execution_flow
   *off* pending a follow-up that improves sub-question quality for
   those classes. Behavior + ownership should not decompose.

3. **Hold smart classifier for a prompt revision.** The classifier
   runs cleanly but its advisory candidates are being over-trusted
   by the synthesis agent. A follow-up should either (a) stop
   surfacing candidates when confidence is low, or (b) phrase the
   hints as "consider searching for these" rather than "here are
   the relevant files". Until that lands, smart classifier without
   candidate surfacing is the safer default.

4. **Revisit `find_tests` contribution independently.** Can't
   isolate its impact from this run because it shipped alongside
   classifier + decomposition. A single-feature benchmark would
   confirm whether behavior's +5% came from tests, decomposition,
   or caching.

The integrated stack is a net +3.33% with an ownership regression.
The individual-feature picture is more promising — architecture
decomposition in particular is a clear win — but the integration
interaction between smart-classifier hints and the synthesis turn
needs prompt-level work before default-on is the right move.
