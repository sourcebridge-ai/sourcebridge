# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Shared evidence extraction and DEEP quality gates."""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from pathlib import PurePosixPath
from typing import Iterable

from workers.knowledge.types import EvidenceRef

FORBIDDEN_GENERIC_PHRASES = [
    "various components",
    "the system handles",
    "as needed",
    "and more",
    "etc.",
    "several modules",
    "in some cases",
    "this functionality",
]

_INLINE_FILE_LINE_RE = re.compile(r"(?P<path>[A-Za-z0-9_./-]+\.[A-Za-z0-9]+):(?P<start>\d+)(?:-(?P<end>\d+))?")
_SYMBOL_MENTION_RE = re.compile(r"`?[A-Za-z_][A-Za-z0-9_]*\(\)`?")
_INVALID_EVIDENCE_PATHS = {"", "none", "null", "repository", "repo", "unknown"}


@dataclass
class ExtractedEvidence:
    path: str
    line_start: int | None = None
    line_end: int | None = None
    kind: str = "code_ref"


@dataclass
class EvidenceGateResult:
    evidence: list[ExtractedEvidence] = field(default_factory=list)
    below_threshold: bool = False
    forbidden_phrases: list[str] = field(default_factory=list)


def find_forbidden_generic_phrases(text: str) -> list[str]:
    lowered = (text or "").lower()
    return [phrase for phrase in FORBIDDEN_GENERIC_PHRASES if phrase in lowered]


def is_valid_evidence_path(path: str) -> bool:
    candidate = (path or "").strip()
    if candidate.lower() in _INVALID_EVIDENCE_PATHS:
        return False
    name = PurePosixPath(candidate).name
    return "." in name and not name.endswith(".")


def extract_inline_evidence(text: str) -> list[ExtractedEvidence]:
    out: list[ExtractedEvidence] = []
    for match in _INLINE_FILE_LINE_RE.finditer(text or ""):
        start = int(match.group("start"))
        end = int(match.group("end")) if match.group("end") else start
        path = match.group("path")
        if not is_valid_evidence_path(path):
            continue
        out.append(ExtractedEvidence(path=path, line_start=start, line_end=end, kind="code_ref"))
    return out


def extract_section_evidence_refs(evidence: Iterable[EvidenceRef]) -> list[ExtractedEvidence]:
    out: list[ExtractedEvidence] = []
    for ref in evidence:
        if not is_valid_evidence_path(ref.file_path):
            continue
        out.append(
            ExtractedEvidence(
                path=ref.file_path,
                line_start=ref.line_start if ref.line_start > 0 else None,
                line_end=ref.line_end if ref.line_end > 0 else None,
                kind="section_field",
            )
        )
    return out


def extract_step_file_symbol_evidence(content: str, file_paths: Iterable[str]) -> list[ExtractedEvidence]:
    has_symbol = bool(_SYMBOL_MENTION_RE.search(content or ""))
    out: list[ExtractedEvidence] = []
    for file_path in file_paths:
        file_path = (file_path or "").strip()
        if not is_valid_evidence_path(file_path):
            continue
        out.append(ExtractedEvidence(path=file_path, kind="section_field" if has_symbol else "code_ref"))
    return out


def extract_code_tour_stop_evidence(file_path: str, line_start: int, line_end: int) -> list[ExtractedEvidence]:
    if not is_valid_evidence_path(file_path) or line_start <= 0 or line_end <= 0:
        return []
    return [ExtractedEvidence(path=file_path, line_start=line_start, line_end=line_end, kind="section_field")]


def evaluate_evidence_gate(*, text: str, evidence: Iterable[ExtractedEvidence], minimum: int) -> EvidenceGateResult:
    extracted = list(evidence) + extract_inline_evidence(text)
    return EvidenceGateResult(
        evidence=extracted,
        below_threshold=len(extracted) < minimum,
        forbidden_phrases=find_forbidden_generic_phrases(text),
    )
