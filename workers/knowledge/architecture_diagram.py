# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

from __future__ import annotations

import json

from workers.common.llm.provider import (
    LLMProvider,
    LLMResponse,
    check_prompt_budget,
    complete_with_optional_model,
    require_nonempty,
)
from workers.common.mermaid.validator import infer_edge_labels, validate_and_repair_mermaid
from workers.knowledge.prompts.architecture_diagram import (
    ARCHITECTURE_DIAGRAM_SYSTEM,
    build_architecture_diagram_prompt,
)
from workers.knowledge.types import EvidenceRef
from workers.reasoning.types import LLMUsageRecord


def _build_evidence(scaffold_json: str) -> list[EvidenceRef]:
    try:
        scaffold = json.loads(scaffold_json)
    except (json.JSONDecodeError, TypeError, ValueError):
        return []
    evidence: list[EvidenceRef] = []
    for module in scaffold.get("modules", [])[:8]:
        if not isinstance(module, dict):
            continue
        module_path = str(module.get("path", "")).strip()
        file_paths = module.get("file_paths", [])
        if not module_path or not isinstance(file_paths, list) or not file_paths:
            continue
        evidence.append(
            EvidenceRef(
                source_type="file",
                file_path=str(file_paths[0]),
                rationale=f"Representative file for module {module_path}",
            )
        )
    return evidence


def _deterministic_edges(scaffold_json: str) -> set[tuple[str, str]]:
    try:
        scaffold = json.loads(scaffold_json)
    except (json.JSONDecodeError, TypeError, ValueError):
        return set()
    edges: set[tuple[str, str]] = set()
    for module in scaffold.get("modules", []):
        if not isinstance(module, dict):
            continue
        src = str(module.get("path", "")).strip()
        if not src:
            continue
        for target in module.get("outbound_paths", []):
            tgt = str(target).strip()
            if tgt:
                edges.add((src, tgt))
    return edges


async def generate_architecture_diagram(
    provider: LLMProvider,
    repository_name: str,
    audience: str,
    depth: str,
    snapshot_json: str,
    deterministic_diagram_json: str,
    model_override: str | None = None,
) -> tuple[dict[str, object], LLMUsageRecord]:
    prompt = build_architecture_diagram_prompt(
        repository_name,
        audience,
        depth,
        snapshot_json,
        deterministic_diagram_json,
    )
    check_prompt_budget(
        prompt,
        system=ARCHITECTURE_DIAGRAM_SYSTEM,
        context="architecture_diagram:repository",
    )

    response: LLMResponse = require_nonempty(
        await complete_with_optional_model(
            provider,
            prompt,
            system=ARCHITECTURE_DIAGRAM_SYSTEM,
            temperature=0.0,
            max_tokens=4096,
            model=model_override,
        ),
        context="architecture_diagram:repository",
    )

    validation = validate_and_repair_mermaid(response.content)
    deterministic_edges = _deterministic_edges(deterministic_diagram_json)
    ai_edges = infer_edge_labels(validation)
    inferred_edges = sorted(f"{src} -> {tgt}" for src, tgt in ai_edges if (src, tgt) not in deterministic_edges)
    if inferred_edges and validation.validation_status == "valid":
        validation.validation_status = "repaired"
        repair_summary = validation.repair_summary.strip()
        validation.repair_summary = "; ".join(
            part for part in [repair_summary, "flagged inferred edges outside deterministic scaffold"] if part
        )

    result = {
        "mermaid_source": validation.mermaid_source,
        "raw_mermaid_source": validation.raw_mermaid_source,
        "validation_status": validation.validation_status,
        "repair_summary": validation.repair_summary,
        "diagram_summary": f"AI-authored architecture diagram for {repository_name}",
        "evidence": _build_evidence(deterministic_diagram_json),
        "inferred_edges": inferred_edges,
    }
    return result, response.usage
