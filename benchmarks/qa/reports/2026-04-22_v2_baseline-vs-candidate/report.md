# QA Parity Report

- Baseline arm: commit `ae92b38` on 2026-04-23 01:39:27.416493+00:00
- Candidate arm: commit `ae92b38` on 2026-04-23 01:05:23.470028+00:00
- Mode: deep vs deep
- Samples: baseline=120 (judged=120, errored=0); candidate=120 (judged=120, errored=0)

## Headline metrics

| Metric | Baseline | Candidate | Delta |
|--------|----------|-----------|-------|
| Answer-useful rate | 63.33% | 62.50% | -0.83% |
| Fallback rate | 0.00% | 0.00% | +0.00% |
| Latency p50 (ms) | 25956 | 6169 | -19788 |
| Latency p95 (ms) | 43903 | 9038 | -34865 |
| Latency p99 (ms) | 47670 | 9939 | -37731 |

## Per-class answer-useful rate

| Class | Baseline | Candidate | Delta | N |
|-------|----------|-----------|-------|---|
| architecture | 76.00% | 68.00% | -8.00% | 25 |
| behavior | 40.00% | 45.00% | +5.00% | 20 |
| cross_cutting | 44.00% | 44.00% | +0.00% | 25 |
| execution_flow | 76.00% | 72.00% | -4.00% | 25 |
| ownership | 76.00% | 80.00% | +4.00% | 25 |

## Top-20 quality regressions (lowest candidate-minus-baseline score)

Human review required before the candidate ships. Sign off in the
Plane epic for the Phase-5 rollout, quoting this section.

| ID | Class | Repo | B | C | Δ | Δlatency (ms) | Fallback change | Judge rationale (candidate) |
|----|-------|------|---|---|---|---------------|-----------------|------------------------------|
| arch-024 | architecture | acme-api | 3 | 1 | -2 | -3908 |  | The question asks about a three-tier system (free/pro/enterprise), but the answer only describes free/pro and does no... |
| own-003 | ownership | sourcebridge | 3 | 1 | -2 | -33357 |  | The answer punts on the question, stating the resolver implementation is not in the provided evidence, rather than id... |
| flow-015 | execution_flow | acme-api | 2 | 1 | -1 | -955 |  | The answer punts on the question, stating the context is insufficient rather than describing the rate-limiting check ... |
| mix-014 | behavior | acme-api | 2 | 1 | -1 | -1661 |  | The answer punts on the actual question, explicitly stating the verification logic isn't in the provided context. It ... |
| mix-012 | behavior | acme-api | 2 | 1 | -1 | -2600 |  | The answer invents specific implementation details (verifyPassword, AuthenticationError, 'Invalid credentials', handl... |
| cross-023 | cross_cutting | acme-api | 2 | 1 | -1 | -2856 |  | The answer punts on the question, stating that the implementation is not available in the provided context rather tha... |
| flow-023 | execution_flow | multi-lang-repo | 2 | 1 | -1 | -4675 |  | The answer punts by stating the context is insufficient, rather than providing concrete information about approval re... |
| flow-021 | execution_flow | multi-lang-repo | 2 | 1 | -1 | -5329 |  | The answer punts on the question, stating insufficient evidence rather than walking through ProcessPayment's flow. It... |
| flow-025 | execution_flow | multi-lang-repo | 2 | 1 | -1 | -5939 |  | The answer punts by saying the context doesn't contain the validation logic, rather than describing concrete checks o... |
| cross-022 | cross_cutting | acme-api | 2 | 1 | -1 | -6921 |  | The question specifically asks about mutations (likely GraphQL mutations or a mutation-specific mechanism), but the a... |
| flow-008 | execution_flow | sourcebridge | 3 | 2 | -1 | -19192 |  | The answer directly traces lane acquisition with concrete mechanics (buffered channel send, select with ctx, release ... |
| flow-006 | execution_flow | sourcebridge | 2 | 1 | -1 | -21406 |  | The answer describes the mutation's input/output shape and mentions generated resolver code, but doesn't explain what... |
| flow-010 | execution_flow | sourcebridge | 3 | 2 | -1 | -21930 |  | The answer directly describes a plausible authentication pipeline with concrete components (JWTManager, currentActorI... |
| own-005 | ownership | sourcebridge | 2 | 1 | -1 | -33590 |  | The answer punts by claiming no evidence exists rather than identifying the deep-QA context assembly code. This is a ... |
| arch-010 | architecture | sourcebridge | 2 | 1 | -1 | -35005 |  | The answer punts by saying the context doesn't show tenant-filtering, rather than identifying the actual mechanism. T... |
| own-009 | ownership | sourcebridge | 2 | 1 | -1 | -35232 |  | The answer punts on the question, stating the evidence doesn't contain the registration, rather than identifying wher... |
| arch-014 | architecture | sourcebridge | 2 | 1 | -1 | -35843 |  | The answer punts on the question, stating the context doesn't contain sufficient evidence to describe the three-tier ... |
| cross-009 | cross_cutting | sourcebridge | 2 | 1 | -1 | -35856 |  | The answer punts by saying the context doesn't reveal hallucination mitigation measures, rather than naming concrete ... |
| cross-007 | cross_cutting | sourcebridge | 2 | 1 | -1 | -37669 |  | The answer punts on the question, explicitly stating the evidence doesn't describe behavior for mid-request revision ... |
| arch-019 | architecture | acme-api | 3 | 3 | +0 | +101 |  | The answer directly addresses the layering question with concrete modules (auth-service, jwt, session, middleware), n... |

## Decision Rule check (plan §Phase 4)

- overall answer-useful within ±7%: **PASS** (Δ=-0.83%)
- per-class within ±10%: **PASS**
- latency p95 within 2× baseline: **PASS**
- top-20 regressions reviewed and signed off by a human: ☐ (tick manually after review)

