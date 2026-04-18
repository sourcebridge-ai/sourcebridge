# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

from __future__ import annotations

import json
import re

from workers.common.llm.provider import (
    LLMProvider,
    LLMResponse,
    check_prompt_budget,
    complete_with_optional_model,
    require_nonempty,
)
from workers.common.mermaid.validator import validate_and_repair_mermaid
from workers.knowledge.prompts.architecture_diagram import (
    ARCHITECTURE_DIAGRAM_SYSTEM,
    build_architecture_component_detail_prompt,
    build_architecture_component_detail_retry_prompt,
    build_architecture_diagram_prompt,
    build_architecture_diagram_retry_prompt,
)
from workers.knowledge.types import EvidenceRef
from workers.reasoning.types import LLMUsageRecord

_EDGE_LABEL_RE = re.compile(r"-->\s*\|([^|]+)\|\s*([A-Za-z0-9_]+)")
_GENERIC_EDGE_LABELS = {
    "primary flow",
    "major flow",
    "secondary flow",
    "data flow",
    "flow",
}

_DETAIL_PREFERRED_KEYWORDS = [
    ("api", 5),
    ("graphql", 5),
    ("worker", 5),
    ("auth", 5),
    ("knowledge", 4),
    ("db", 4),
    ("store", 4),
    ("repo", 3),
    ("orchestr", 4),
]
_DETAIL_PENALTY_KEYWORDS = ["test", "benchmark", "fixture", "generated", "docs"]


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


def _detail_evidence_from_modules(modules: list[dict[str, object]]) -> list[EvidenceRef]:
    evidence: list[EvidenceRef] = []
    for module in modules[:8]:
        file_paths = module.get("file_paths") or []
        if not isinstance(file_paths, list) or not file_paths:
            continue
        evidence.append(
            EvidenceRef(
                source_type="file",
                file_path=str(file_paths[0]),
                rationale=f"Representative file for detail module {module.get('path', '')}",
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


def _load_scaffold_modules(scaffold_json: str) -> list[dict[str, object]]:
    try:
        scaffold = json.loads(scaffold_json)
    except (json.JSONDecodeError, TypeError, ValueError):
        return []
    modules = scaffold.get("modules") or []
    return [module for module in modules if isinstance(module, dict)]


def _rank_detail_candidates(scaffold_json: str) -> list[str]:
    scored: list[tuple[int, str]] = []
    for module in _load_scaffold_modules(scaffold_json):
        path = str(module.get("path", "")).strip()
        if not path:
            continue
        lowered = path.lower()
        score = len(module.get("outbound_paths") or [])
        for keyword, weight in _DETAIL_PREFERRED_KEYWORDS:
            if keyword in lowered:
                score += weight
        for keyword in _DETAIL_PENALTY_KEYWORDS:
            if keyword in lowered:
                score -= 6
        scored.append((score, path))
    scored.sort(key=lambda item: (-item[0], item[1]))
    return [path for _, path in scored if _ > 0]


def _module_id(path: str) -> str:
    normalized = re.sub(r"[^a-zA-Z0-9]+", "_", path.strip("/"))
    return normalized.strip("_") or "module"


def _select_detail_modules(scaffold_json: str, subsystem_name: str) -> list[dict[str, object]]:
    modules = _load_scaffold_modules(scaffold_json)
    selected: list[dict[str, object]] = []
    prefix = subsystem_name.rstrip("/")
    for module in modules:
        path = str(module.get("path", "")).strip()
        if path == prefix or path.startswith(prefix + "/"):
            selected.append(module)
    if not selected:
        for module in modules:
            path = str(module.get("path", "")).strip()
            if prefix in path:
                selected.append(module)
    return selected[:8]


def _system_flow_edges(snapshot_json: str) -> set[tuple[str, str]]:
    _, flows = _load_system_context(snapshot_json)
    edges: set[tuple[str, str]] = set()
    for flow in flows:
        if not isinstance(flow, dict):
            continue
        src = str(flow.get("source_id", "")).strip()
        tgt = str(flow.get("target_id", "")).strip()
        if src and tgt:
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
    labels_by_id: dict[str, str] = {}
    for component in components:
        if not isinstance(component, dict):
            continue
        comp_id = str(component.get("id", "")).strip()
        label = str(component.get("label", "")).strip()
        if not comp_id or not label:
            continue
        component_ids.add(comp_id)
        labels_by_id[comp_id] = label
        if component.get("kind") == "interface":
            interface_present = True

    if not component_ids:
        return None
    group_specs = [
        ("interfaces", "Interfaces", ["user_interfaces"]),
        ("core_platform", "Core Platform", ["api_auth", "knowledge_orchestration", "code_graph_index"]),
        ("execution", "Execution", ["background_workers"]),
        ("storage_external", "Storage & External", ["repository_access", "persistence", "llm_provider"]),
    ]

    if interface_present:
        lines.append('    user["User"]')
    for group_id, group_label, members in group_specs:
        members = [member for member in members if member in component_ids]
        if not members:
            continue
        lines.append(f'    subgraph {group_id}["{group_label}"]')
        for member in members:
            lines.append(f'        {member}["{labels_by_id[member]}"]')
        lines.append("    end")
    lines.extend(
        [
            "    classDef primary fill:#1f3b5b,stroke:#9fd3ff,color:#f5fbff,stroke-width:2px;",
            "    classDef support fill:#263238,stroke:#90a4ae,color:#f5f7fa,stroke-width:1px;",
            "    classDef external fill:#3f2f21,stroke:#f2c078,color:#fff7ea,stroke-width:1px;",
            "    class user,user_interfaces,api_auth,knowledge_orchestration,background_workers primary;",
            "    class code_graph_index,repository_access,persistence support;",
            "    class llm_provider external;",
        ]
    )
    if interface_present:
        lines.append("    user --> user_interfaces")

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


def _load_system_context(snapshot_json: str) -> tuple[list[dict[str, object]], list[dict[str, object]]]:
    try:
        payload = json.loads(snapshot_json)
    except (json.JSONDecodeError, TypeError, ValueError):
        return [], []
    components = payload.get("system_components") or []
    flows = payload.get("system_flows") or []
    if not isinstance(components, list):
        components = []
    if not isinstance(flows, list):
        flows = []
    return components, flows


def _system_view_summary(repository_name: str, snapshot_json: str) -> str:
    components, _ = _load_system_context(snapshot_json)
    component_ids = {str(component.get("id", "")).strip() for component in components if isinstance(component, dict)}
    if not component_ids:
        return f"{repository_name} is shown as a high-level system view."
    parts = [f"{repository_name} routes user requests through the interfaces and API"]
    if "knowledge_orchestration" in component_ids:
        parts.append("hands knowledge generation to the orchestration layer")
    if "background_workers" in component_ids:
        parts.append("executes jobs in background workers")
    if "code_graph_index" in component_ids:
        parts.append("grounds analysis in the code graph and repository understanding")
    if "persistence" in component_ids:
        parts.append("persists artifacts and job state")
    if "llm_provider" in component_ids:
        parts.append("and calls the configured LLM provider when synthesis is needed")
    return ", ".join(parts) + "."


def _component_detail_summary(subsystem_name: str, modules: list[dict[str, object]]) -> str:
    labels = [str(module.get("path", "")).strip() for module in modules[:4] if str(module.get("path", "")).strip()]
    if labels:
        return f'Detail view of "{subsystem_name}" covering ' + ", ".join(labels) + "."
    return f'Detail view of "{subsystem_name}".'


def _diagram_quality_issues(mermaid_source: str) -> list[str]:
    issues: list[str] = []
    validation = validate_and_repair_mermaid(mermaid_source)
    node_count = len(validation.node_labels)
    edge_count = len(validation.edge_pairs)
    if node_count > 8:
        issues.append(f"too many boxes ({node_count} > 8)")
    if edge_count > 10:
        issues.append(f"too many edges ({edge_count} > 10)")
    if node_count and edge_count > node_count + 2:
        issues.append("too much cross-linking for a system-context view")

    raw_pairs: set[tuple[str, str]] = set()
    generic_labels = False
    for line in mermaid_source.splitlines():
        match = _EDGE_LABEL_RE.search(line)
        if match and match.group(1).strip().lower() in _GENERIC_EDGE_LABELS:
            generic_labels = True
        parts = line.split("-->")
        if len(parts) != 2:
            continue
        src = parts[0].strip().split()[-1] if parts[0].strip() else ""
        rhs = parts[1].strip()
        if "|" in rhs:
            rhs = rhs.rsplit("|", 1)[-1].strip()
        tgt = rhs.split()[0] if rhs else ""
        if src and tgt:
            raw_pairs.add((src, tgt))
    if generic_labels:
        issues.append("uses generic edge labels")
    reciprocal_pairs = sum(1 for src, tgt in raw_pairs if (tgt, src) in raw_pairs and src < tgt)
    if reciprocal_pairs:
        issues.append("contains reciprocal edge pairs")
    return issues


def _detail_quality_issues(mermaid_source: str) -> list[str]:
    issues = _diagram_quality_issues(mermaid_source)
    validation = validate_and_repair_mermaid(mermaid_source)
    node_count = len(validation.node_labels)
    edge_count = len(validation.edge_pairs)
    issues = [issue for issue in issues if "too many boxes" not in issue and "too many edges" not in issue]
    if node_count > 8:
        issues.append(f"too many boxes ({node_count} > 8)")
    if edge_count > 12:
        issues.append(f"too many edges ({edge_count} > 12)")
    if node_count < 3:
        issues.append("too few boxes for a meaningful detail view")
    return issues


def _component_detail_fallback_mermaid(subsystem_name: str, modules: list[dict[str, object]]) -> str | None:
    if not modules:
        return None
    selected = modules[:8]
    lines = ["flowchart LR"]
    module_paths = {str(module.get("path", "")).strip() for module in selected}
    for module in selected:
        path = str(module.get("path", "")).strip()
        if not path:
            continue
        label = path.split("/")[-1] or path
        lines.append(f'    {_module_id(path)}["{label}"]')
    seen_edges: set[tuple[str, str]] = set()
    for module in selected:
        src = str(module.get("path", "")).strip()
        if not src:
            continue
        for target in module.get("outbound_paths") or []:
            tgt = str(target).strip()
            if tgt not in module_paths:
                continue
            edge = (src, tgt)
            if edge in seen_edges:
                continue
            seen_edges.add(edge)
            lines.append(f"    {_module_id(src)} --> {_module_id(tgt)}")
    if len(lines) <= 2:
        return None
    return "\n".join(lines)


def _graph_alignment(
    validation, snapshot_json: str, deterministic_diagram_json: str
) -> tuple[list[str], list[str], list[str]]:
    system_edges = _system_flow_edges(snapshot_json)
    supported_basis = system_edges or _deterministic_edges(deterministic_diagram_json)
    ai_edges = validation.edge_pairs
    supported: list[str] = []
    inferred: list[str] = []
    contradictory: list[str] = []
    for src, tgt in sorted(ai_edges):
        edge = f"{src} -> {tgt}"
        if (src, tgt) in supported_basis:
            supported.append(edge)
        elif (tgt, src) in supported_basis:
            contradictory.append(edge)
        else:
            inferred.append(edge)
    return supported, inferred, contradictory


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
    generation_strategy = "llm"
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
            generation_strategy = "fallback"
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
    quality_issues = _diagram_quality_issues(validation.mermaid_source)
    if quality_issues:
        retry_prompt = build_architecture_diagram_retry_prompt(
            repository_name,
            audience,
            depth,
            deterministic_diagram_json,
            response.content + "\n\nQuality issues: " + "; ".join(quality_issues),
        )
        retry_response = require_nonempty(
            await complete_with_optional_model(
                provider,
                retry_prompt,
                system=ARCHITECTURE_DIAGRAM_SYSTEM,
                temperature=0.0,
                max_tokens=3072,
                model=model_override,
            ),
            context="architecture_diagram:repository_quality_retry",
        )
        usage = LLMUsageRecord(
            provider=retry_response.provider_name or response.provider_name or "llm",
            model=retry_response.model or response.model,
            input_tokens=usage.input_tokens + retry_response.input_tokens,
            output_tokens=usage.output_tokens + retry_response.output_tokens,
            operation="architecture_diagram",
            entity_name=repository_name,
        )
        try:
            retry_validation = validate_and_repair_mermaid(retry_response.content)
            retry_issues = _diagram_quality_issues(retry_validation.mermaid_source)
            if retry_issues:
                raise ValueError("; ".join(retry_issues))
            validation = retry_validation
            repair_summary = validation.repair_summary.strip()
            validation.repair_summary = "; ".join(
                part for part in [repair_summary, "regenerated diagram to satisfy system-view quality gate"] if part
            )
        except ValueError as quality_exc:
            fallback = _system_view_fallback_mermaid(snapshot_json)
            if not fallback:
                raise quality_exc
            validation = validate_and_repair_mermaid(fallback)
            generation_strategy = "fallback"
            repair_summary = validation.repair_summary.strip()
            validation.repair_summary = "; ".join(
                part
                for part in [
                    repair_summary,
                    "regenerated diagram to satisfy system-view quality gate",
                    f"fell back to deterministic system view: {quality_exc}",
                ]
                if part
            )
    supported_edges, inferred_edges, contradictory_edges = _graph_alignment(
        validation,
        snapshot_json,
        deterministic_diagram_json,
    )
    if contradictory_edges:
        if generation_strategy != "fallback":
            generation_strategy = "repaired"
        validation.validation_status = "repaired"
        repair_summary = validation.repair_summary.strip()
        validation.repair_summary = "; ".join(
            part for part in [repair_summary, f"flagged {len(contradictory_edges)} graph-contradictory edges"] if part
        )
    if inferred_edges and validation.validation_status == "valid" and generation_strategy == "llm":
        validation.validation_status = "repaired"
        generation_strategy = "repaired"
        repair_summary = validation.repair_summary.strip()
        validation.repair_summary = "; ".join(
            part for part in [repair_summary, "flagged inferred edges outside deterministic scaffold"] if part
        )
    elif validation.validation_status == "repaired" and generation_strategy == "llm":
        generation_strategy = "repaired"

    detail_mermaid_source = ""
    detail_raw_mermaid_source = ""
    detail_validation_status = ""
    detail_repair_summary = ""
    detail_diagram_summary = ""
    detail_subsystem_name = ""
    detail_candidate_subsystems: list[str] = []
    detail_evidence: list[EvidenceRef] = []
    if depth == "deep":
        detail_candidate_subsystems = _rank_detail_candidates(deterministic_diagram_json)[:4]
        if detail_candidate_subsystems:
            detail_subsystem_name = detail_candidate_subsystems[0]
            detail_modules = _select_detail_modules(deterministic_diagram_json, detail_subsystem_name)
            detail_evidence = _detail_evidence_from_modules(detail_modules)
            detail_context = json.dumps(
                {
                    "repository_name": repository_name,
                    "subsystem_name": detail_subsystem_name,
                    "modules": detail_modules,
                }
            )
            detail_prompt = build_architecture_component_detail_prompt(
                repository_name,
                audience,
                detail_subsystem_name,
                detail_context,
            )
            check_prompt_budget(
                detail_prompt,
                system=ARCHITECTURE_DIAGRAM_SYSTEM,
                context="architecture_diagram:repository_detail",
            )
            detail_response = require_nonempty(
                await complete_with_optional_model(
                    provider,
                    detail_prompt,
                    system=ARCHITECTURE_DIAGRAM_SYSTEM,
                    temperature=0.0,
                    max_tokens=3072,
                    model=model_override,
                ),
                context="architecture_diagram:repository_detail",
            )
            usage = LLMUsageRecord(
                provider=detail_response.provider_name or usage.provider,
                model=detail_response.model or usage.model,
                input_tokens=usage.input_tokens + detail_response.input_tokens,
                output_tokens=usage.output_tokens + detail_response.output_tokens,
                operation="architecture_diagram",
                entity_name=repository_name,
            )
            try:
                detail_validation = validate_and_repair_mermaid(detail_response.content)
            except ValueError as exc:
                retry_prompt = build_architecture_component_detail_retry_prompt(
                    repository_name,
                    audience,
                    detail_subsystem_name,
                    detail_context,
                    detail_response.content,
                )
                retry_response = require_nonempty(
                    await complete_with_optional_model(
                        provider,
                        retry_prompt,
                        system=ARCHITECTURE_DIAGRAM_SYSTEM,
                        temperature=0.0,
                        max_tokens=3072,
                        model=model_override,
                    ),
                    context="architecture_diagram:repository_detail_retry",
                )
                usage = LLMUsageRecord(
                    provider=retry_response.provider_name or usage.provider,
                    model=retry_response.model or usage.model,
                    input_tokens=usage.input_tokens + retry_response.input_tokens,
                    output_tokens=usage.output_tokens + retry_response.output_tokens,
                    operation="architecture_diagram",
                    entity_name=repository_name,
                )
                try:
                    detail_validation = validate_and_repair_mermaid(retry_response.content)
                    detail_validation.repair_summary = "; ".join(
                        part for part in [detail_validation.repair_summary.strip(), f"retry regenerated invalid Mermaid: {exc}"] if part
                    )
                except ValueError as retry_exc:
                    fallback = _component_detail_fallback_mermaid(detail_subsystem_name, detail_modules)
                    if not fallback:
                        raise retry_exc
                    detail_validation = validate_and_repair_mermaid(fallback)
                    detail_validation.repair_summary = "; ".join(
                        part for part in [detail_validation.repair_summary.strip(), f"fell back to deterministic detail view: {retry_exc}"] if part
                    )
            detail_issues = _detail_quality_issues(detail_validation.mermaid_source)
            if detail_issues:
                fallback = _component_detail_fallback_mermaid(detail_subsystem_name, detail_modules)
                if fallback:
                    detail_validation = validate_and_repair_mermaid(fallback)
                    detail_validation.repair_summary = "; ".join(
                        part for part in [detail_validation.repair_summary.strip(), "fell back to deterministic detail view after quality gate"] if part
                    )
            detail_mermaid_source = detail_validation.mermaid_source
            detail_raw_mermaid_source = detail_validation.raw_mermaid_source
            detail_validation_status = detail_validation.validation_status
            detail_repair_summary = detail_validation.repair_summary
            detail_diagram_summary = _component_detail_summary(detail_subsystem_name, detail_modules)

    result = {
        "mermaid_source": validation.mermaid_source,
        "raw_mermaid_source": validation.raw_mermaid_source,
        "validation_status": validation.validation_status,
        "repair_summary": validation.repair_summary,
        "diagram_summary": _system_view_summary(repository_name, snapshot_json),
        "evidence": _build_evidence(deterministic_diagram_json),
        "supported_edges": supported_edges,
        "inferred_edges": inferred_edges,
        "contradictory_edges": contradictory_edges,
        "generation_strategy": generation_strategy,
        "detail_mermaid_source": detail_mermaid_source,
        "detail_raw_mermaid_source": detail_raw_mermaid_source,
        "detail_validation_status": detail_validation_status,
        "detail_repair_summary": detail_repair_summary,
        "detail_diagram_summary": detail_diagram_summary,
        "detail_subsystem_name": detail_subsystem_name,
        "detail_candidate_subsystems": detail_candidate_subsystems,
        "detail_evidence": detail_evidence,
    }
    return result, usage
