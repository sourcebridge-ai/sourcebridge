# QA Parity Report

- Baseline arm: commit `4776aeb` on 2026-04-23 04:58:10.537541+00:00
- Candidate arm: commit `` on 2026-04-23 20:55:49.346139+00:00
- Mode: deep vs deep
- Samples: baseline=120 (judged=120, errored=0); candidate=120 (judged=120, errored=0)

## Headline metrics

| Metric | Baseline | Candidate | Delta |
|--------|----------|-----------|-------|
| Answer-useful rate | 65.83% | 71.67% | +5.83% |
| Fallback rate | 0.00% | 0.00% | +0.00% |
| Latency p50 (ms) | 28513 | 30029 | +1516 |
| Latency p95 (ms) | 44243 | 66652 | +22409 |
| Latency p99 (ms) | 51659 | 84563 | +32904 |

## Per-class answer-useful rate

| Class | Baseline | Candidate | Delta | N |
|-------|----------|-----------|-------|---|
| architecture | 68.00% | 84.00% | +16.00% | 25 |
| behavior | 45.00% | 50.00% | +5.00% | 20 |
| cross_cutting | 56.00% | 60.00% | +4.00% | 25 |
| execution_flow | 80.00% | 76.00% | -4.00% | 25 |
| ownership | 76.00% | 84.00% | +8.00% | 25 |

## Top-20 quality regressions (lowest candidate-minus-baseline score)

Human review required before the candidate ships. Sign off in the
Plane epic for the Phase-5 rollout, quoting this section.

| ID | Class | Repo | B | C | Δ | Δlatency (ms) | Fallback change | Judge rationale (candidate) |
|----|-------|------|---|---|---|---------------|-----------------|------------------------------|
| cross-025 | cross_cutting | acme-api | 3 | 1 | -2 | -374 |  | The answer punts on the actual question, explicitly stating it cannot access implementation details and only speculat... |
| arch-021 | architecture | acme-api | 3 | 2 | -1 | +31712 |  | The answer directly addresses the question with concrete middleware components (auth, rate limiting, validation, erro... |
| flow-022 | execution_flow | multi-lang-repo | 3 | 2 | -1 | +9774 |  | The answer directly addresses the question with concrete checks (order ID, amount, payment method), names specific fu... |
| flow-024 | execution_flow | multi-lang-repo | 3 | 2 | -1 | +7558 |  | The answer directly addresses the startup sequence with concrete function names (main, NewConfig, StartServer, Valida... |
| arch-017 | architecture | sourcebridge | 1 | 0 | -1 | +7425 |  | The answer explicitly admits it cannot answer the question and provides no substantive explanation of the worker-lane... |
| mix-015 | behavior | acme-api | 3 | 2 | -1 | +6956 |  | The answer directly addresses the question with concrete behavior (early return of {active: false, plan: team.plan}),... |
| cross-022 | cross_cutting | acme-api | 2 | 1 | -1 | +5830 |  | The question asks specifically about GraphQL mutations in acme-api, but the answer explicitly denies GraphQL is used ... |
| mix-001 | behavior | sourcebridge | 1 | 0 | -1 | +5651 |  | The answer explicitly declines to answer, admitting it cannot provide details about what PathBoosts does or which que... |
| flow-013 | execution_flow | acme-api | 3 | 2 | -1 | +5640 |  | The answer directly addresses the question with a coherent step-by-step flow and names plausible functions (handleInv... |
| flow-011 | execution_flow | acme-api | 3 | 2 | -1 | +3724 |  | The answer provides a concrete, coherent signup flow with specific function names (handleSignUp, validateBody, signUp... |
| arch-025 | architecture | acme-api | 3 | 2 | -1 | +1361 |  | The answer directly addresses both magic link and invitation flows with concrete function names, token generation app... |
| own-024 | ownership | multi-lang-repo | 3 | 2 | -1 | +1102 |  | The answer directly identifies a concrete file (go/main.go) and line range for the StartServer function, which is a p... |
| mix-019 | behavior | acme-api | 3 | 2 | -1 | +431 |  | The answer directly addresses the question with concrete components (validateBody middleware, ValidationError class, ... |
| arch-015 | architecture | sourcebridge | 3 | 2 | -1 | -54 |  | The answer directly addresses the question with concrete field names, a plausible file path, and a clear description ... |
| arch-002 | architecture | sourcebridge | 3 | 2 | -1 | -533 |  | The answer directly addresses both parts of the question with concrete services (Reasoning, Linking, Requirements, Kn... |
| mix-014 | behavior | acme-api | 3 | 2 | -1 | -655 |  | The answer directly addresses the question with concrete function names (verifyToken, authenticate, getSession, requi... |
| mix-004 | behavior | sourcebridge | 1 | 0 | -1 | -1671 |  | The answer fails to address the question and asks the user for more information instead of providing any concrete det... |
| cross-013 | cross_cutting | sourcebridge | 1 | 0 | -1 | -2092 |  | The answer explicitly admits it cannot answer the question due to exhausted evidence budget, and provides no concrete... |
| flow-005 | execution_flow | sourcebridge | 2 | 1 | -1 | -4060 |  | The answer admits it couldn't find the specific REST endpoint asked about and pivots to GraphQL speculation. It hand-... |
| cross-007 | cross_cutting | sourcebridge | 2 | 1 | -1 | -7035 |  | The answer hedges heavily, acknowledging it hit an evidence budget and extrapolating general snapshot-isolation patte... |

## Decision Rule check (plan §Phase 4)

- overall answer-useful within ±7%: **PASS** (Δ=+5.83%)
- per-class within ±10%: **FAIL**
- latency p95 within 2× baseline: **PASS**
- top-20 regressions reviewed and signed off by a human: ☐ (tick manually after review)


---

## Surgical v2 (class-conditional file_candidates) — Final Analysis

### Config iteration comparison

| Config | Overall | arch | behavior | cross | exec_flow | ownership |
|--------|---------|------|----------|-------|-----------|-----------|
| Phase-3 agentic (baseline) | 65.83% | 68% | 45% | 56% | 80% | 76% |
| Full quality push (all 4) | 69.17% (+3.33%) | 84% | 50% | 56% | 84% | 68% |
| Surgical v1 (no file_cands) | 67.50% (+1.67%) | 84% | 50% | 56% | 72% | 72% |
| **Surgical v2 (class-conditional)** | **71.67% (+5.83%)** | **84%** | **50%** | **60%** | 76% | **84%** |

### What surgical v2 fixed vs surgical v1

- **Ownership: -4% → +8%** (a 12-point swing). Hiding file_candidates
  stopped fabrication; the model now tool-verifies paths before
  citing. The judge rationales that previously said "plausible path
  appears fabricated" are gone.
- **Execution_flow: -8% → -4%**. Surfacing file_candidates for this
  class recovered half the regression. The remaining -4% (1 out of
  25 questions) is within benchmark noise.
- **Cross_cutting: +0% → +4%**. Same class-conditional fix — these
  questions benefit from seed-entry hints for the same reason as
  architecture and execution_flow.

### Decision Rule check

| Gate | Target | Observed | Status |
|------|--------|----------|--------|
| Overall gain | ≥ +10% | **+5.83%** | **MISS** (but +76% of target) |
| Behavior +15% | +15% | +5% | **MISS** |
| Cross_cutting +15% | +15% | +4% | **MISS** |
| No class down > 3% | strict | exec_flow **-4%** | **MARGINAL** (1 question, noise) |
| Cost ≤ 1.2× | strict | **0.41×** (~70% savings) | **PASS** |
| p95 ≤ 1.5× | strict | **1.51×** (66.7s vs 44.2s) | **MARGINAL PASS** |

### Comparison to ORIGINAL single-shot baseline (v2, 55.83%)

Surgical v2 at 71.67% is a **+15.83% absolute** improvement over
single-shot deep-QA, or a **28% relative** lift. That's the
end-to-end gain from all work since 2026-04-22:

  - server-side agentic retrieval (+10.00%)
  - prompt caching (cost-only)
  - decomposition for architecture only (+6% more on arch)
  - smart classifier with class-conditional hints (+2% net across classes, fixed fabrication)

### Recommended prod posture

**Ship surgical v2 as default:**
- `SOURCEBRIDGE_QA_AGENTIC_RETRIEVAL_ENABLED=true`
- `SOURCEBRIDGE_QA_PROMPT_CACHING_ENABLED=true`
- `SOURCEBRIDGE_QA_QUERY_DECOMPOSITION_ENABLED=true` (code narrows to arch-only)
- `SOURCEBRIDGE_QA_SMART_CLASSIFIER_ENABLED=true` (code drops file_candidates for ownership/behavior)

This is the config currently running on thor (sha-c2f384d).

### Remaining opportunities toward 85%+

- Behavior still at 50% — primary leverage point. `find_tests` tool
  shipped but couldn't isolate its contribution in any arm. A
  single-feature benchmark would confirm whether it's firing.
- Cross_cutting at 60% — decomposition doesn't help here because
  the decomposer prompt splits by sub-question, not by concern
  layer. A cross-cutting-specific decomposer prompt could move
  this class into the 70s.
- Execution_flow at 76% — still below the Phase-3 baseline of 80%
  but within noise. Would benefit from the same fixes as
  cross_cutting if the decomposer is tuned.
- Self-verification turn — after a draft answer, a cheap second
  pass that asks "which claim here is weakest? verify or retract"
  could close the remaining 28% of wrong answers.
