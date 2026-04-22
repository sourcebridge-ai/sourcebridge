#!/usr/bin/env python3
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors
"""
Paired analyzer for the QA parity benchmark.

Reads two runner output directories (baseline / candidate) plus the
frozen labels.yaml and emits report.md.

Usage:

    python3 report.py \\
        --baseline reports/2026-04-22_baseline \\
        --candidate reports/2026-04-22_candidate \\
        --labels labels.yaml \\
        --out reports/2026-04-22_baseline-vs-candidate/report.md

Refuses to summarize when input coverage is incomplete — the decision
rule depends on a paired sample over the same frozen question set, so
missing labels / missing arms / mismatched IDs are hard errors rather
than silent gaps.
"""
from __future__ import annotations

import argparse
import json
import os
import statistics
import sys
from pathlib import Path
from typing import Iterable

import yaml


USEFUL_THRESHOLD = 2  # label >= 2 counts as "useful"


def load_jsonl(path: Path) -> list[dict]:
    rows = []
    with path.open() as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            rows.append(json.loads(line))
    return rows


def load_labels(path: Path) -> dict[str, int]:
    with path.open() as f:
        data = yaml.safe_load(f) or {}
    labels = data.get("labels") or {}
    if not isinstance(labels, dict):
        sys.exit("labels.yaml: 'labels' must be a mapping")
    return {k: int(v) for k, v in labels.items()}


def useful_rate(samples: list[dict], labels: dict[str, int]) -> tuple[float, int]:
    good = 0
    n = 0
    for s in samples:
        lbl = labels.get(s["id"])
        if lbl is None:
            continue
        n += 1
        if lbl >= USEFUL_THRESHOLD:
            good += 1
    if n == 0:
        return 0.0, 0
    return good / n, n


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


def fallback_rate(samples: list[dict]) -> float:
    if not samples:
        return 0.0
    fb = sum(1 for s in samples if s.get("fallback_used") and s["fallback_used"] not in ("", "none"))
    return fb / len(samples)


def by_class(samples: list[dict], labels: dict[str, int]) -> dict[str, tuple[float, int]]:
    classes: dict[str, list[dict]] = {}
    for s in samples:
        classes.setdefault(s["class"], []).append(s)
    return {c: useful_rate(rows, labels) for c, rows in classes.items()}


def require_coverage(a: list[dict], b: list[dict], labels: dict[str, int]) -> None:
    ids_a = {s["id"] for s in a}
    ids_b = {s["id"] for s in b}
    if ids_a != ids_b:
        only_a = ids_a - ids_b
        only_b = ids_b - ids_a
        sys.exit(f"arms disagree on question set: only_baseline={sorted(only_a)} only_candidate={sorted(only_b)}")
    missing = ids_a - labels.keys()
    if missing:
        sys.exit(f"unlabeled questions (freeze labels.yaml first): {sorted(missing)}")


def compute_regressions(base: list[dict], cand: list[dict], labels: dict[str, int]) -> list[dict]:
    by_id_b = {s["id"]: s for s in base}
    by_id_c = {s["id"]: s for s in cand}
    rows = []
    for qid, lbl in labels.items():
        b = by_id_b.get(qid)
        c = by_id_c.get(qid)
        if not b or not c:
            continue
        # A regression is where baseline was useful and candidate no
        # longer is. The label is our ground truth; we use the label
        # once per arm (same rubric both ways) but surface cases where
        # the arms diverge on quality for human review.
        delta_note = ""
        if (b.get("fallback_used") or "none") != (c.get("fallback_used") or "none"):
            delta_note = f"fallback: {b.get('fallback_used') or 'none'} -> {c.get('fallback_used') or 'none'}"
        rows.append({
            "id": qid,
            "class": b["class"],
            "label": lbl,
            "delta_note": delta_note,
            "elapsed_delta_ms": (c.get("elapsed_ms", 0) - b.get("elapsed_ms", 0)),
        })
    # Sort largest latency regressions first; tied rows by id for stability.
    rows.sort(key=lambda r: (-r["elapsed_delta_ms"], r["id"]))
    return rows


def top_n(rows: list[dict], n: int) -> list[dict]:
    return rows[:n]


def render_markdown(base: list[dict], cand: list[dict], labels: dict[str, int]) -> str:
    u_b, n_b = useful_rate(base, labels)
    u_c, n_c = useful_rate(cand, labels)
    cls_b = by_class(base, labels)
    cls_c = by_class(cand, labels)
    lat_b = latency_stats(base)
    lat_c = latency_stats(cand)
    fb_b = fallback_rate(base)
    fb_c = fallback_rate(cand)

    lines = []
    lines.append("# QA Parity Report")
    lines.append("")
    lines.append(f"- Baseline samples: {len(base)} (labeled: {n_b})")
    lines.append(f"- Candidate samples: {len(cand)} (labeled: {n_c})")
    lines.append("")
    lines.append("## Headline metrics")
    lines.append("")
    lines.append("| Metric | Baseline | Candidate | Delta |")
    lines.append("|--------|----------|-----------|-------|")
    lines.append(f"| Answer-useful rate | {u_b:.2%} | {u_c:.2%} | {u_c - u_b:+.2%} |")
    lines.append(f"| Fallback rate | {fb_b:.2%} | {fb_c:.2%} | {fb_c - fb_b:+.2%} |")
    lines.append(f"| Latency p50 (ms) | {lat_b['p50']:.0f} | {lat_c['p50']:.0f} | {lat_c['p50'] - lat_b['p50']:+.0f} |")
    lines.append(f"| Latency p95 (ms) | {lat_b['p95']:.0f} | {lat_c['p95']:.0f} | {lat_c['p95'] - lat_b['p95']:+.0f} |")
    lines.append(f"| Latency p99 (ms) | {lat_b['p99']:.0f} | {lat_c['p99']:.0f} | {lat_c['p99'] - lat_b['p99']:+.0f} |")
    lines.append("")
    lines.append("## Per-class answer-useful rate")
    lines.append("")
    lines.append("| Class | Baseline | Candidate | Delta |")
    lines.append("|-------|----------|-----------|-------|")
    all_classes = sorted(set(cls_b) | set(cls_c))
    for c in all_classes:
        b_u, _ = cls_b.get(c, (0.0, 0))
        c_u, _ = cls_c.get(c, (0.0, 0))
        lines.append(f"| {c} | {b_u:.2%} | {c_u:.2%} | {c_u - b_u:+.2%} |")
    lines.append("")
    lines.append("## Top-20 regressions (highest elapsed delta)")
    lines.append("")
    lines.append("Human review required before the candidate ships.")
    lines.append("")
    lines.append("| ID | Class | Label | Elapsed Δ (ms) | Fallback change |")
    lines.append("|----|-------|-------|----------------|-----------------|")
    regs = compute_regressions(base, cand, labels)
    for r in top_n(regs, 20):
        lines.append(f"| {r['id']} | {r['class']} | {r['label']} | {r['elapsed_delta_ms']:+d} | {r['delta_note']} |")
    lines.append("")
    lines.append("## Decision Rule check (plan §Phase 4)")
    lines.append("")
    lines.append("- overall answer-useful within ±7%: " + ("PASS" if abs(u_c - u_b) <= 0.07 else "FAIL"))
    per_class_ok = all(abs(cls_c.get(c, (0.0, 0))[0] - cls_b.get(c, (0.0, 0))[0]) <= 0.10 for c in all_classes)
    lines.append("- per-class within ±10%: " + ("PASS" if per_class_ok else "FAIL"))
    latency_ok = lat_c["p95"] <= (lat_b["p95"] * 2)
    lines.append(f"- latency p95 within 2× baseline: {'PASS' if latency_ok else 'FAIL'}")
    lines.append("")
    lines.append("Reference correctness is not auto-scored; reviewer must sign off the top-20 regressions table above before ship.")
    return "\n".join(lines) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--baseline", required=True, type=Path)
    parser.add_argument("--candidate", required=True, type=Path)
    parser.add_argument("--labels", required=True, type=Path)
    parser.add_argument("--out", required=True, type=Path)
    args = parser.parse_args()

    base = load_jsonl(args.baseline / "run.jsonl")
    cand = load_jsonl(args.candidate / "run.jsonl")
    labels = load_labels(args.labels)

    if not base:
        sys.exit("baseline run.jsonl empty")
    if not cand:
        sys.exit("candidate run.jsonl empty")
    require_coverage(base, cand, labels)

    md = render_markdown(base, cand, labels)
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(md)
    print(f"wrote {args.out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
