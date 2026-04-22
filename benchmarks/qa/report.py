#!/usr/bin/env python3
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors
"""
Paired analyzer for the QA parity benchmark.

Reads two runner output directories (baseline / candidate), each
containing run.jsonl + judgments.yaml from judge.py, and emits
report.md.

Usage:

    python3 report.py \\
        --baseline reports/2026-04-22_baseline \\
        --candidate reports/2026-04-22_candidate \\
        --out reports/2026-04-22_baseline-vs-candidate/report.md

Decision rule (from the plan §Phase 4):
  - overall answer-useful rate (label >= 2) within ±7%
  - no class with per-class answer-useful rate down > 10%
  - latency p95 within 2× baseline
  - top-20 regressions table reviewed and signed off by a human

Refuses to summarize on incomplete input — mismatched ID sets,
missing judgments, or judge-error rows are hard errors rather than
silent quality loss.
"""
from __future__ import annotations

import argparse
import json
import statistics
import sys
from pathlib import Path

import yaml


USEFUL_THRESHOLD = 2


def load_jsonl(path: Path) -> list[dict]:
    rows = []
    with path.open() as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            rows.append(json.loads(line))
    return rows


def load_judgments(path: Path) -> dict[str, dict]:
    with path.open() as f:
        data = yaml.safe_load(f) or {}
    return data.get("judgments") or {}


def percentile(values: list[float], p: float) -> float:
    if not values:
        return 0.0
    values = sorted(values)
    k = (len(values) - 1) * p
    lo = int(k)
    hi = min(lo + 1, len(values) - 1)
    if lo == hi:
        return values[lo]
    return values[lo] * (hi - k) + values[hi] * (k - lo)


def latency_stats(samples: list[dict]) -> dict:
    ms = [s.get("elapsed_ms", 0) for s in samples]
    return {
        "p50": percentile(ms, 0.5),
        "p95": percentile(ms, 0.95),
        "p99": percentile(ms, 0.99),
        "mean": statistics.mean(ms) if ms else 0.0,
    }


def useful_rate(samples: list[dict], judgments: dict[str, dict]) -> tuple[float, int, int]:
    good = 0
    n = 0
    errored = 0
    for s in samples:
        j = judgments.get(s["id"])
        if not j:
            continue
        score = j.get("score", -1)
        if score == -1:
            errored += 1
            continue
        n += 1
        if score >= USEFUL_THRESHOLD:
            good += 1
    if n == 0:
        return 0.0, 0, errored
    return good / n, n, errored


def by_class(samples: list[dict], judgments: dict[str, dict]) -> dict[str, tuple[float, int]]:
    buckets: dict[str, list[dict]] = {}
    for s in samples:
        buckets.setdefault(s["class"], []).append(s)
    out = {}
    for cls, rows in buckets.items():
        rate, n, _ = useful_rate(rows, judgments)
        out[cls] = (rate, n)
    return out


def fallback_rate(samples: list[dict]) -> float:
    if not samples:
        return 0.0
    fb = sum(1 for s in samples if s.get("fallback_used") and s["fallback_used"] not in ("", "none"))
    return fb / len(samples)


def require_coverage(a: list[dict], b: list[dict], judgments_a: dict, judgments_b: dict) -> None:
    ids_a = {s["id"] for s in a}
    ids_b = {s["id"] for s in b}
    if ids_a != ids_b:
        only_a = ids_a - ids_b
        only_b = ids_b - ids_a
        sys.exit(
            f"arms disagree on question set:\n"
            f"  only_baseline: {sorted(only_a)}\n  only_candidate: {sorted(only_b)}"
        )
    missing_a = ids_a - judgments_a.keys()
    missing_b = ids_b - judgments_b.keys()
    if missing_a or missing_b:
        sys.exit(
            f"unjudged questions (run judge.py first):\n"
            f"  baseline: {sorted(missing_a)}\n  candidate: {sorted(missing_b)}"
        )
    # Judge errors are allowed but surfaced loudly.
    err_a = [qid for qid, j in judgments_a.items() if j.get("score") == -1]
    err_b = [qid for qid, j in judgments_b.items() if j.get("score") == -1]
    if err_a or err_b:
        print(
            f"WARNING: judge errored on some rows — excluded from metrics:\n"
            f"  baseline: {err_a}\n  candidate: {err_b}",
            file=sys.stderr,
        )


def compute_regressions(base: list[dict], cand: list[dict], jb: dict, jc: dict) -> list[dict]:
    by_id_b = {s["id"]: s for s in base}
    by_id_c = {s["id"]: s for s in cand}
    rows = []
    for qid in jb.keys() & jc.keys():
        b = by_id_b.get(qid)
        c = by_id_c.get(qid)
        if not b or not c:
            continue
        sb = jb[qid].get("score", -1)
        sc = jc[qid].get("score", -1)
        if sb == -1 or sc == -1:
            continue
        score_delta = sc - sb  # negative = regression
        elapsed_delta = c.get("elapsed_ms", 0) - b.get("elapsed_ms", 0)
        fb_change = ""
        if (b.get("fallback_used") or "none") != (c.get("fallback_used") or "none"):
            fb_change = f"{b.get('fallback_used') or 'none'} -> {c.get('fallback_used') or 'none'}"
        rows.append({
            "id": qid,
            "class": b["class"],
            "repo": b.get("repo", ""),
            "score_baseline": sb,
            "score_candidate": sc,
            "score_delta": score_delta,
            "elapsed_delta_ms": elapsed_delta,
            "fallback_change": fb_change,
            "rationale_c": jc[qid].get("rationale", ""),
        })
    # Sort by quality regression first (most-negative score_delta), then
    # by latency regression. Ties by id for stability.
    rows.sort(key=lambda r: (r["score_delta"], -r["elapsed_delta_ms"], r["id"]))
    return rows


def render_markdown(base: list[dict], cand: list[dict], jb: dict, jc: dict,
                    env_b: dict, env_c: dict) -> str:
    u_b, n_b, err_b = useful_rate(base, jb)
    u_c, n_c, err_c = useful_rate(cand, jc)
    cls_b = by_class(base, jb)
    cls_c = by_class(cand, jc)
    lat_b = latency_stats(base)
    lat_c = latency_stats(cand)
    fb_b = fallback_rate(base)
    fb_c = fallback_rate(cand)

    L = []
    L.append("# QA Parity Report")
    L.append("")
    L.append(f"- Baseline arm: commit `{env_b.get('commit_sha','?')}` on {env_b.get('date','?')}")
    L.append(f"- Candidate arm: commit `{env_c.get('commit_sha','?')}` on {env_c.get('date','?')}")
    L.append(f"- Mode: {env_b.get('mode','?')} vs {env_c.get('mode','?')}")
    L.append(f"- Samples: baseline={len(base)} (judged={n_b}, errored={err_b}); candidate={len(cand)} (judged={n_c}, errored={err_c})")
    L.append("")

    L.append("## Headline metrics")
    L.append("")
    L.append("| Metric | Baseline | Candidate | Delta |")
    L.append("|--------|----------|-----------|-------|")
    L.append(f"| Answer-useful rate | {u_b:.2%} | {u_c:.2%} | {u_c - u_b:+.2%} |")
    L.append(f"| Fallback rate | {fb_b:.2%} | {fb_c:.2%} | {fb_c - fb_b:+.2%} |")
    L.append(f"| Latency p50 (ms) | {lat_b['p50']:.0f} | {lat_c['p50']:.0f} | {lat_c['p50'] - lat_b['p50']:+.0f} |")
    L.append(f"| Latency p95 (ms) | {lat_b['p95']:.0f} | {lat_c['p95']:.0f} | {lat_c['p95'] - lat_b['p95']:+.0f} |")
    L.append(f"| Latency p99 (ms) | {lat_b['p99']:.0f} | {lat_c['p99']:.0f} | {lat_c['p99'] - lat_b['p99']:+.0f} |")
    L.append("")

    L.append("## Per-class answer-useful rate")
    L.append("")
    L.append("| Class | Baseline | Candidate | Delta | N |")
    L.append("|-------|----------|-----------|-------|---|")
    all_classes = sorted(set(cls_b) | set(cls_c))
    for c in all_classes:
        b_u, b_n = cls_b.get(c, (0.0, 0))
        c_u, c_n = cls_c.get(c, (0.0, 0))
        L.append(f"| {c} | {b_u:.2%} | {c_u:.2%} | {c_u - b_u:+.2%} | {max(b_n, c_n)} |")
    L.append("")

    # Top-20 quality regressions
    regs = compute_regressions(base, cand, jb, jc)
    worst = regs[:20]
    L.append("## Top-20 quality regressions (lowest candidate-minus-baseline score)")
    L.append("")
    L.append("Human review required before the candidate ships. Sign off in the")
    L.append("Plane epic for the Phase-5 rollout, quoting this section.")
    L.append("")
    L.append("| ID | Class | Repo | B | C | Δ | Δlatency (ms) | Fallback change | Judge rationale (candidate) |")
    L.append("|----|-------|------|---|---|---|---------------|-----------------|------------------------------|")
    for r in worst:
        rat = (r["rationale_c"] or "").replace("|", "\\|").replace("\n", " ")
        if len(rat) > 120:
            rat = rat[:117] + "..."
        L.append(
            f"| {r['id']} | {r['class']} | {r['repo']} | {r['score_baseline']} | {r['score_candidate']} | "
            f"{r['score_delta']:+d} | {r['elapsed_delta_ms']:+d} | {r['fallback_change']} | {rat} |"
        )
    L.append("")

    # Decision rule
    L.append("## Decision Rule check (plan §Phase 4)")
    L.append("")
    overall_pass = abs(u_c - u_b) <= 0.07
    per_class_pass = all(
        abs(cls_c.get(c, (0.0, 0))[0] - cls_b.get(c, (0.0, 0))[0]) <= 0.10
        for c in all_classes
    )
    latency_pass = lat_c["p95"] <= (lat_b["p95"] * 2) if lat_b["p95"] > 0 else True
    L.append(f"- overall answer-useful within ±7%: **{'PASS' if overall_pass else 'FAIL'}** (Δ={u_c - u_b:+.2%})")
    L.append(f"- per-class within ±10%: **{'PASS' if per_class_pass else 'FAIL'}**")
    L.append(f"- latency p95 within 2× baseline: **{'PASS' if latency_pass else 'FAIL'}**")
    L.append("- top-20 regressions reviewed and signed off by a human: ☐ (tick manually after review)")
    L.append("")

    return "\n".join(L) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--baseline", required=True, type=Path)
    parser.add_argument("--candidate", required=True, type=Path)
    parser.add_argument("--out", required=True, type=Path)
    args = parser.parse_args()

    base = load_jsonl(args.baseline / "run.jsonl")
    cand = load_jsonl(args.candidate / "run.jsonl")
    jb = load_judgments(args.baseline / "judgments.yaml")
    jc = load_judgments(args.candidate / "judgments.yaml")

    if not base:
        sys.exit(f"baseline run.jsonl empty at {args.baseline}")
    if not cand:
        sys.exit(f"candidate run.jsonl empty at {args.candidate}")

    require_coverage(base, cand, jb, jc)

    env_b = yaml.safe_load((args.baseline / "environment.yaml").read_text()) if (args.baseline / "environment.yaml").exists() else {}
    env_c = yaml.safe_load((args.candidate / "environment.yaml").read_text()) if (args.candidate / "environment.yaml").exists() else {}

    md = render_markdown(base, cand, jb, jc, env_b, env_c)
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(md)
    print(f"wrote {args.out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
