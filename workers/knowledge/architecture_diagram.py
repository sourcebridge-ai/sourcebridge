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
    build_architecture_diagram_retry_prompt,
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


def _system_view_fallback_mermaid(snapshot_json: str) -> str | None:
    try:
        payload = json.loads(snapshot_json)
    except (json.JSONDecodeError, TypeError, ValueError):
        return None
    components = payload.get("system_components") or []
    flows = payload.get("system_flows") or []
    if not isinstance(components, list) or not components:
        return None

    lines: list[str] = ["flowchart LR"]
    component_ids: set[str] = set()
    interface_present = False
    for component in components:
        if not isinstance(component, dict):
            continue
        comp_id = str(component.get("id", "")).strip()
        label = str(component.get("label", "")).strip()
        if not comp_id or not label:
            continue
        component_ids.add(comp_id)
        if component.get("kind") == "interface":
            interface_present = True
        node_id = comp_id
        lines.append(f'    {node_id}["{label}"]')

    if not component_ids:
        return None
    if interface_present:
        lines.insert(1, '    user["User"]')
        lines.insert(2, "    user --> user_interfaces")

    seen_edges: set[tuple[str, str]] = set()
    for flow in flows:
        if not isinstance(flow, dict):
            continue
        src = str(flow.get("source_id", "")).strip()
        tgt = str(flow.get("target_id", "")).strip()
        if not src or not tgt or src == tgt:
            continue
        if src not in component_ids or tgt not in component_ids:
            continue
        edge = (src, tgt)
        if edge in seen_edges:
            continue
        seen_edges.add(edge)
        summary = str(flow.get("summary", "")).strip()
        if summary:
            lines.append(f"    {src} -->|{summary}| {tgt}")
        else:
            lines.append(f"    {src} --> {tgt}")
    return "\n".join(lines)


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
    usage = LLMUsageRecord(
        provider=response.provider_name or "llm",
        model=response.model,
        input_tokens=response.input_tokens,
        output_tokens=response.output_tokens,
        operation="architecture_diagram",
        entity_name=repository_name,
    )
    try:
        validation = validate_and_repair_mermaid(response.content)
    except ValueError as exc:
        retry_prompt = build_architecture_diagram_retry_prompt(
            repository_name,
            audience,
            depth,
            deterministic_diagram_json,
            response.content,
        )
        retry_response: LLMResponse = require_nonempty(
            await complete_with_optional_model(
                provider,
                retry_prompt,
                system=ARCHITECTURE_DIAGRAM_SYSTEM,
                temperature=0.0,
                max_tokens=3072,
                model=model_override,
            ),
            context="architecture_diagram:repository_retry",
        )
        usage = LLMUsageRecord(
            provider=retry_response.provider_name or response.provider_name or "llm",
            model=retry_response.model or response.model,
            input_tokens=response.input_tokens + retry_response.input_tokens,
            output_tokens=response.output_tokens + retry_response.output_tokens,
            operation="architecture_diagram",
            entity_name=repository_name,
        )
        try:
            validation = validate_and_repair_mermaid(retry_response.content)
            repair_summary = validation.repair_summary.strip()
            validation.repair_summary = "; ".join(
                part for part in [repair_summary, f"retry regenerated invalid Mermaid: {exc}"] if part
            )
        except ValueError as retry_exc:
            fallback = _system_view_fallback_mermaid(snapshot_json)
            if not fallback:
                raise retry_exc
            validation = validate_and_repair_mermaid(fallback)
            repair_summary = validation.repair_summary.strip()
            validation.repair_summary = "; ".join(
                part
                for part in [
                    repair_summary,
                    f"retry regenerated invalid Mermaid: {exc}",
                    f"fell back to deterministic system view: {retry_exc}",
                ]
                if part
            )
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
    return result, usage
