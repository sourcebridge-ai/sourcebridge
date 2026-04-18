# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Code tour generation using LLM."""

from __future__ import annotations

import json
from dataclasses import dataclass, field

import structlog

from workers.common.llm.provider import (
    LLMProvider,
    LLMResponse,
    check_prompt_budget,
    complete_with_optional_model,
    require_nonempty,
)
from workers.knowledge.cliff_notes import _parse_sections
from workers.knowledge.evidence import evaluate_evidence_gate, extract_code_tour_stop_evidence
from workers.knowledge.prompts.code_tour import (
    CODE_TOUR_SYSTEM,
    build_code_tour_prompt,
)
from workers.reasoning.types import LLMUsageRecord

log = structlog.get_logger()


@dataclass
class TourStop:
    """A single stop in a code tour."""

    order: int
    title: str
    description: str  # markdown
    file_path: str
    line_start: int = 0
    line_end: int = 0
    trail: str = ""
    modification_hints: list[str] = field(default_factory=list)
    confidence: str = "medium"
    refinement_status: str = ""


@dataclass
class CodeTourResult:
    """The full code tour generation result."""

    stops: list[TourStop] = field(default_factory=list)


def _parse_stops(raw: str) -> list[dict[str, object]]:
    """Parse JSON array from LLM response using the shared robust parser."""
    return _parse_sections(raw)


async def generate_code_tour(
    provider: LLMProvider,
    repository_name: str,
    audience: str,
    depth: str,
    snapshot_json: str,
    theme: str = "",
    model_override: str | None = None,
) -> tuple[CodeTourResult, LLMUsageRecord]:
    """Generate a code tour from a repository snapshot."""
    prompt = build_code_tour_prompt(repository_name, audience, depth, snapshot_json, theme)

    check_prompt_budget(
        prompt,
        system=CODE_TOUR_SYSTEM,
        context="code_tour:repository",
    )

    response: LLMResponse = require_nonempty(
        await complete_with_optional_model(
            provider,
            prompt,
            system=CODE_TOUR_SYSTEM,
            temperature=0.0,
            model=model_override,
        ),
        context="code_tour:repository",
    )

    try:
        raw_stops = _parse_stops(response.content)
    except (json.JSONDecodeError, ValueError, TypeError) as exc:
        log.warning("code_tour_parse_fallback", error=str(exc))
        raw_stops = [
            {
                "order": 1,
                "title": "Overview",
                "description": response.content,
                "file_path": "",
                "line_start": 0,
                "line_end": 0,
            }
        ]

    stops: list[TourStop] = []
    for raw in raw_stops:
        if not isinstance(raw, dict):
            raw = {"title": str(raw)[:160], "description": str(raw)}
        stops.append(
            TourStop(
                order=raw.get("order", len(stops) + 1),
                title=raw.get("title", "Untitled"),
                description=raw.get("description", ""),
                file_path=raw.get("file_path", ""),
                line_start=raw.get("line_start", 0),
                line_end=raw.get("line_end", 0),
                trail=raw.get("trail", ""),
                modification_hints=raw.get("modification_hints", []),
            )
        )

    if depth == "deep":
        for stop in stops:
            gate = evaluate_evidence_gate(
                text=f"{stop.description}\n" + "\n".join(stop.modification_hints),
                evidence=extract_code_tour_stop_evidence(stop.file_path, stop.line_start, stop.line_end),
                minimum=1,
            )
            if gate.below_threshold or gate.forbidden_phrases or not stop.trail:
                stop.confidence = "low"
                stop.refinement_status = "needs_evidence"

    usage = LLMUsageRecord(
        provider="llm",
        model=response.model,
        input_tokens=response.input_tokens,
        output_tokens=response.output_tokens,
        operation="code_tour",
        entity_name=repository_name,
    )

    return CodeTourResult(stops=stops), usage
