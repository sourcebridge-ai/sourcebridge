#!/usr/bin/env python3
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors
"""
LLM-as-judge for the QA parity benchmark.

Reads a runner output (run.jsonl) and scores each answer against its
question using Claude Opus 4.7 (or another model via --model). Writes
judgments.yaml with one entry per question, including numeric score,
rationale, and a hash of (question, answer) so stale judgments can be
detected and regenerated.

Usage:

    ANTHROPIC_API_KEY=... python3 judge.py \\
        --run  reports/2026-04-22_candidate/run.jsonl \\
        --out  reports/2026-04-22_candidate/judgments.yaml \\
        --model claude-opus-4-7

Then the same command with --run pointing at baseline/run.jsonl to
produce a paired baseline judgments file.

Decision rubric (0..3):
  0 — misleading / wrong (actively hurts the user)
  1 — not useful (correct-ish but doesn't answer the question)
  2 — useful (answers the question, minor gaps)
  3 — excellent (answers fully, cites correct evidence)

The judge sees only (question, answer, references). It never sees
which arm produced the answer. This asymmetric-blind protocol keeps
the paired comparison honest.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
import time
from pathlib import Path
from typing import Any

import yaml

try:
    from anthropic import Anthropic
except ImportError:
    sys.exit("ERROR: pip install anthropic")


SYSTEM_PROMPT = """You are an expert software engineer grading answers to
questions about an unfamiliar codebase. You will be shown one question
and one answer. Grade the answer on a 0-3 scale:

0 — Misleading or wrong. The answer makes false claims about the code
    or actively mis-directs a reader trying to understand the system.
1 — Not useful. The answer is correct-ish in isolation but does not
    actually answer the question that was asked.
2 — Useful. The answer addresses the question with at most minor gaps.
    A reader would come away with a correct understanding.
3 — Excellent. The answer is thorough, directly responsive, and cites
    concrete evidence (file paths, function names, REQ-* IDs) when
    appropriate.

You MUST respond with a single JSON object of the form:

  {"score": <0|1|2|3>, "rationale": "<one or two sentences>"}

Do not include any text before or after the JSON. Do not wrap it in
a markdown code block."""


USER_TEMPLATE = """Question: {question}

Answer: {answer}

References cited by the answer (may be empty):
{references}

Grade this answer now."""


def answer_hash(question: str, answer: str) -> str:
    """Stable hash used to detect stale judgments when re-running."""
    h = hashlib.sha256()
    h.update(question.encode("utf-8"))
    h.update(b"\x1f")
    h.update(answer.encode("utf-8"))
    return h.hexdigest()[:16]


def judge_one(client: Anthropic, model: str, question: str, answer: str, references: list[str]) -> dict:
    prompt = USER_TEMPLATE.format(
        question=question,
        answer=answer or "(empty)",
        references="\n".join(f"- {r}" for r in references) if references else "(none)",
    )
    # Retry transient errors once; anything persistent surfaces as a
    # parse_error in the judgments file so the report.py coverage
    # check flags it loudly.
    for attempt in (0, 1):
        try:
            resp = client.messages.create(
                model=model,
                max_tokens=256,
                system=SYSTEM_PROMPT,
                messages=[{"role": "user", "content": prompt}],
            )
            text = "".join(block.text for block in resp.content if getattr(block, "type", "") == "text")
            data = json.loads(text.strip())
            if "score" not in data or data["score"] not in (0, 1, 2, 3):
                raise ValueError(f"invalid score: {data}")
            return {
                "score": int(data["score"]),
                "rationale": str(data.get("rationale", "")),
                "model": model,
            }
        except Exception as e:  # noqa: BLE001
            if attempt == 0:
                time.sleep(2)
                continue
            return {"score": -1, "rationale": f"judge error: {e}", "model": model}
    return {"score": -1, "rationale": "unreachable", "model": model}


def load_jsonl(path: Path) -> list[dict]:
    rows = []
    with path.open() as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            rows.append(json.loads(line))
    return rows


def load_existing(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {"version": 1, "judgments": {}}
    with path.open() as f:
        data = yaml.safe_load(f) or {}
    if "judgments" not in data:
        data["judgments"] = {}
    return data


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--run", required=True, type=Path, help="Path to runner run.jsonl")
    parser.add_argument("--out", required=True, type=Path, help="Path to write judgments.yaml")
    parser.add_argument("--model", default="claude-opus-4-7", help="Anthropic model ID for judgment")
    parser.add_argument("--api-key", default=os.environ.get("ANTHROPIC_API_KEY"))
    parser.add_argument("--resume", action="store_true", help="Skip questions already judged (by answer hash)")
    args = parser.parse_args()

    if not args.api_key:
        sys.exit("ERROR: --api-key or ANTHROPIC_API_KEY required")

    samples = load_jsonl(args.run)
    if not samples:
        sys.exit(f"no samples in {args.run}")

    existing = load_existing(args.out)
    judgments: dict[str, dict] = existing.get("judgments") or {}
    client = Anthropic(api_key=args.api_key)

    print(f"judging {len(samples)} samples with {args.model}", file=sys.stderr)
    for i, s in enumerate(samples, 1):
        qid = s["id"]
        question = s["question"]
        answer = s.get("answer", "")
        refs = s.get("references") or []
        h = answer_hash(question, answer)
        if args.resume and qid in judgments and judgments[qid].get("answer_hash") == h:
            print(f"  [{i}/{len(samples)}] {qid} cached", file=sys.stderr)
            continue
        print(f"  [{i}/{len(samples)}] {qid} ...", end="", file=sys.stderr, flush=True)
        verdict = judge_one(client, args.model, question, answer, refs)
        judgments[qid] = {
            "score": verdict["score"],
            "rationale": verdict["rationale"],
            "model": verdict["model"],
            "answer_hash": h,
        }
        print(f" {verdict['score']}", file=sys.stderr)
        # Checkpoint every 10 samples so a long run can be resumed.
        if i % 10 == 0:
            args.out.parent.mkdir(parents=True, exist_ok=True)
            with args.out.open("w") as f:
                yaml.safe_dump({"version": 1, "judgments": judgments}, f, sort_keys=True)

    args.out.parent.mkdir(parents=True, exist_ok=True)
    with args.out.open("w") as f:
        yaml.safe_dump({"version": 1, "judgments": judgments}, f, sort_keys=True)
    print(f"wrote {args.out}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
