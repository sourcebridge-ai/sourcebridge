# QA Parity Benchmark

Deep-QA parity harness for the server-side orchestrator migration
(plan `2026-04-22-deep-qa-server-side-orchestrator.md`, Phase 4).

## What this proves

Paired before/after comparison of answer-useful rate, reference
correctness, latency, and fallback rate across a frozen question set.
The candidate (server-side QA) ships only when the paired report
clears the §Decision Rule defined in the plan:

- overall answer-useful rate within ±7% on N≥120 questions
- no major question class down > 10% answer-useful rate
- reference correctness within ±10%
- top-20 regressions reviewed and signed off by a human
- latency p95 within 2× the Python baseline

## Layout

- `questions.yaml`   — frozen question set (≥ 120, across 3 repos,
                        5 question classes)
- `labels.yaml`      — adjudicated 0..3 labels per question, two
                        reviewers, disputes resolved before freeze
- `cmd/runner/`      — Go binary that hits either the Python
                        subprocess baseline or the server endpoint
                        and writes `run.jsonl` + `environment.yaml`
- `report.py`        — paired analyzer; emits `report.md`
- `reports/`         — per-date directories holding paired reports

See `cmd/runner/README.md` for invocation recipes.

## Status

Infrastructure scaffold only. Authoring the 120-question set is a
follow-up task owned by whoever is tuning deep mode — pull candidates
from MCP / CLI logs, curate, adjudicate with a second reviewer, and
commit to `questions.yaml` + `labels.yaml` before any parity claim
is treated as valid.
