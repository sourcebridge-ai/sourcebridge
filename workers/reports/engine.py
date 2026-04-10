# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Report generation engine.

Orchestrates the full pipeline: collect → analyze → generate → assemble → render.
This is the main entry point called by the gRPC servicer or REST handler.
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Callable, Awaitable

from workers.common.llm.provider import LLMProvider
from workers.reports.assembler import assemble_report, count_words
from workers.reports.evidence_registry import EvidenceRegistry
from workers.reports.prompts.audience import get_audience
from workers.reports.section_generator import (
    GeneratedSection,
    generate_placeholder_section,
    generate_section,
)

logger = logging.getLogger(__name__)

ProgressCallback = Callable[[float, str, str], Awaitable[None] | None]


async def _noop_progress(progress: float, phase: str, message: str) -> None:
    pass


@dataclass
class ReportConfig:
    """Configuration for a report generation run."""

    report_id: str
    report_name: str
    report_type: str  # "architecture_baseline", "swot", etc.
    audience: str  # "c_suite", "developer", etc.
    repository_ids: list[str] = field(default_factory=list)
    selected_sections: list[str] = field(default_factory=list)
    include_diagrams: bool = False
    output_formats: list[str] = field(default_factory=lambda: ["markdown"])
    loe_mode: str = "human_hours"
    branding_footer: str = ""
    output_dir: str = ""  # where to write files
    model_override: str | None = None


@dataclass
class ReportResult:
    """Output of a report generation run."""

    markdown: str
    section_count: int
    word_count: int
    evidence_count: int
    content_dir: str
    sections: list[GeneratedSection] = field(default_factory=list)


# Section definitions — imported from Go via REST or hardcoded here for standalone use.
# These match internal/reports/sections.go.
SECTION_DEFINITIONS: dict[str, dict[str, Any]] = {}  # populated at runtime from API


async def generate_report(
    provider: LLMProvider,
    config: ReportConfig,
    *,
    repo_data: dict[str, Any] | None = None,
    section_definitions: list[dict[str, Any]] | None = None,
    progress: ProgressCallback = _noop_progress,
) -> ReportResult:
    """Generate a complete report.

    Args:
        provider: LLM provider for section generation.
        config: Report configuration.
        repo_data: Pre-collected repository data (cliff notes, scores, etc.)
                   keyed by repo ID. If None, generates with placeholder data.
        section_definitions: Section metadata from the Go API. If None, uses
                            config.selected_sections as plain keys.
        progress: Callback for progress reporting.

    Returns:
        ReportResult with the assembled Markdown and metadata.
    """
    await _maybe_await(progress(0.05, "collecting", "Preparing report data"))

    # Build portfolio context
    repo_names = []
    if repo_data:
        for rid in config.repository_ids:
            rd = repo_data.get(rid, {})
            repo_names.append(rd.get("name", rid))
    else:
        repo_names = config.repository_ids

    portfolio_context = _build_portfolio_context(config, repo_data, repo_names)

    # Resolve section definitions
    sections_to_generate = _resolve_sections(config, section_definitions)

    # Separate terminal sections (depend on *) from independent ones
    independent = []
    terminal = []
    dependent = []
    for sec in sections_to_generate:
        deps = sec.get("depends_on", [])
        if "*" in deps:
            terminal.append(sec)
        elif deps:
            dependent.append(sec)
        else:
            independent.append(sec)

    evidence = EvidenceRegistry()
    generated: dict[str, GeneratedSection] = {}

    # Phase 1: Generate independent sections (can parallelize)
    total = len(sections_to_generate)
    completed = 0

    await _maybe_await(progress(0.1, "generating", f"Generating {len(independent)} independent sections"))

    for sec in independent:
        section_data = _collect_section_data(sec, config, repo_data)
        result = await generate_section(
            provider,
            section_key=sec["key"],
            section_title=sec["title"],
            section_category=sec.get("category", ""),
            report_type=config.report_type,
            audience=config.audience,
            portfolio_context=portfolio_context,
            section_data=section_data,
            loe_mode=config.loe_mode,
            min_words=sec.get("min_word_count", 150),
            model_override=config.model_override,
        )
        generated[sec["key"]] = result
        completed += 1
        pct = 0.1 + 0.6 * (completed / total)
        await _maybe_await(progress(pct, "generating", f"Generated {result.title} ({completed}/{total})"))

    # Phase 2: Generate dependent sections
    for sec in dependent:
        section_data = _collect_section_data(sec, config, repo_data)
        # Add dependent section data from already-generated sections
        dep_context = ""
        for dep_key in sec.get("depends_on", []):
            if dep_key in generated:
                dep_context += f"\n\n### From {generated[dep_key].title}:\n{generated[dep_key].markdown[:1000]}"
        if dep_context:
            section_data += f"\n\n--- Dependent section context ---\n{dep_context}"

        result = await generate_section(
            provider,
            section_key=sec["key"],
            section_title=sec["title"],
            section_category=sec.get("category", ""),
            report_type=config.report_type,
            audience=config.audience,
            portfolio_context=portfolio_context,
            section_data=section_data,
            loe_mode=config.loe_mode,
            min_words=sec.get("min_word_count", 150),
            model_override=config.model_override,
        )
        generated[sec["key"]] = result
        completed += 1
        pct = 0.1 + 0.6 * (completed / total)
        await _maybe_await(progress(pct, "generating", f"Generated {result.title} ({completed}/{total})"))

    # Phase 3: Generate terminal sections (Executive Summary etc.)
    await _maybe_await(progress(0.75, "synthesizing", "Generating executive synthesis"))
    for sec in terminal:
        # Feed ALL generated sections as context
        all_context = "\n\n".join(
            f"### {g.title}\n{g.markdown[:500]}" for g in generated.values()
        )
        section_data = f"Summary of all report sections:\n{all_context}"

        result = await generate_section(
            provider,
            section_key=sec["key"],
            section_title=sec["title"],
            section_category=sec.get("category", ""),
            report_type=config.report_type,
            audience=config.audience,
            portfolio_context=portfolio_context,
            section_data=section_data,
            loe_mode=config.loe_mode,
            min_words=sec.get("min_word_count", 300),
            extra_instructions="This is the executive synthesis section. Summarize the key findings from all sections above into a coherent narrative.",
            model_override=config.model_override,
        )
        generated[sec["key"]] = result
        completed += 1

    # Phase 4: Assemble
    await _maybe_await(progress(0.85, "assembling", "Assembling document"))

    # Order sections by their definition order
    ordered_sections = []
    for sec in sections_to_generate:
        if sec["key"] in generated:
            ordered_sections.append(generated[sec["key"]])

    aud = get_audience(config.audience)
    markdown = assemble_report(
        report_name=config.report_name,
        report_type=config.report_type,
        audience_title=aud.title,
        repo_names=repo_names,
        sections=ordered_sections,
        evidence=evidence,
        include_diagrams=config.include_diagrams,
        branding_footer=config.branding_footer,
    )

    # Phase 5: Write to disk
    await _maybe_await(progress(0.9, "rendering", "Writing output files"))

    content_dir = config.output_dir
    if not content_dir:
        content_dir = f"/data/reports/{config.report_id}/v1"
    os.makedirs(content_dir, exist_ok=True)

    md_path = os.path.join(content_dir, "report.md")
    with open(md_path, "w") as f:
        f.write(markdown)

    # Write evidence registry
    evidence_path = os.path.join(content_dir, "evidence.json")
    with open(evidence_path, "w") as f:
        json.dump(
            [
                {
                    "evidence_id": item.evidence_id,
                    "category": item.category,
                    "title": item.title,
                    "description": item.description,
                    "severity": item.severity,
                    "file_path": item.file_path,
                }
                for item in evidence.all_items()
            ],
            f,
            indent=2,
        )

    await _maybe_await(progress(1.0, "ready", "Report generation complete"))

    return ReportResult(
        markdown=markdown,
        section_count=len(ordered_sections),
        word_count=count_words(markdown),
        evidence_count=evidence.count(),
        content_dir=content_dir,
        sections=ordered_sections,
    )


def _build_portfolio_context(
    config: ReportConfig,
    repo_data: dict[str, Any] | None,
    repo_names: list[str],
) -> str:
    """Build the portfolio context string fed into every section prompt."""
    lines = [
        f"Portfolio: {len(config.repository_ids)} repositories ({', '.join(repo_names)})",
    ]
    if repo_data:
        # Detect common technologies
        all_languages: dict[str, int] = {}
        total_files = 0
        total_symbols = 0
        for rid in config.repository_ids:
            rd = repo_data.get(rid, {})
            total_files += rd.get("file_count", 0)
            total_symbols += rd.get("symbol_count", 0)
            for lang in rd.get("languages", []):
                if isinstance(lang, str):
                    all_languages[lang] = all_languages.get(lang, 0) + 1
                elif isinstance(lang, dict):
                    name = lang.get("name", "")
                    if name:
                        all_languages[name] = all_languages.get(name, 0) + 1

        if all_languages:
            sorted_langs = sorted(all_languages.items(), key=lambda x: -x[1])
            lines.append(f"Common technologies: {', '.join(f'{k} ({v} repos)' for k, v in sorted_langs[:5])}")
        lines.append(f"Total files: {total_files}, Total symbols: {total_symbols}")

        # Add cliff notes summaries if available
        for rid in config.repository_ids:
            rd = repo_data.get(rid, {})
            cliff_notes = rd.get("cliff_notes", [])
            if cliff_notes:
                lines.append(f"\n{rd.get('name', rid)} analysis:")
                for cn in cliff_notes[:3]:
                    if isinstance(cn, dict):
                        lines.append(f"  - {cn.get('title', '')}: {cn.get('summary', '')}")

    return "\n".join(lines)


def _resolve_sections(
    config: ReportConfig,
    section_definitions: list[dict[str, Any]] | None,
) -> list[dict[str, Any]]:
    """Resolve section definitions from the API or config."""
    if section_definitions:
        return [s for s in section_definitions if s.get("key") in config.selected_sections]

    # Fallback: generate minimal definitions from keys
    return [
        {
            "key": key,
            "title": key.replace("_", " ").title(),
            "category": "General",
            "min_word_count": 150,
            "depends_on": ["*"] if key in ("executive_summary", "overall_assessment", "dd_executive_summary") else [],
        }
        for key in config.selected_sections
    ]


def _collect_section_data(
    section_def: dict[str, Any],
    config: ReportConfig,
    repo_data: dict[str, Any] | None,
) -> str:
    """Collect the relevant data for a section from repo_data."""
    if not repo_data:
        return "(No repository data available)"

    lines = []
    data_sources = section_def.get("data_sources", [])

    for rid in config.repository_ids:
        rd = repo_data.get(rid, {})
        repo_name = rd.get("name", rid)
        lines.append(f"\n--- {repo_name} ---")

        if "repo_metadata" in data_sources:
            lines.append(f"Files: {rd.get('file_count', 'unknown')}")
            lines.append(f"Symbols: {rd.get('symbol_count', 'unknown')}")
            languages = rd.get("languages", [])
            if languages:
                lines.append(f"Languages: {', '.join(str(l) for l in languages[:5])}")

        if "understanding_scores" in data_sources:
            scores = rd.get("understanding_score", {})
            if scores:
                lines.append(f"Understanding score: {scores.get('overall', 'N/A')}")
                lines.append(f"  Traceability: {scores.get('traceabilityCoverage', 'N/A')}%")
                lines.append(f"  Documentation: {scores.get('documentationCoverage', 'N/A')}%")
                lines.append(f"  Test: {scores.get('testCoverage', 'N/A')}%")

        if "cliff_notes" in data_sources:
            cliff = rd.get("cliff_notes", [])
            for cn in cliff:
                if isinstance(cn, dict):
                    lines.append(f"[{cn.get('title', '')}] {cn.get('content', '')[:300]}")

        if "test_detection" in data_sources:
            test_info = rd.get("test_detection", {})
            if test_info:
                lines.append(f"Test files: {test_info.get('test_file_count', 0)}")
                lines.append(f"Test frameworks: {', '.join(test_info.get('frameworks', []))}")

        if "dependency_audit" in data_sources:
            dep_info = rd.get("dependency_audit", {})
            if dep_info:
                lines.append(f"Vulnerabilities: {dep_info.get('vulnerability_count', 0)}")
                lines.append(f"Outdated packages: {dep_info.get('outdated_count', 0)}")

        if "auth_detection" in data_sources:
            auth_info = rd.get("auth_detection", {})
            if auth_info:
                lines.append(f"Auth patterns: {', '.join(auth_info.get('patterns', []))}")

        if "cicd_detection" in data_sources:
            cicd_info = rd.get("cicd_detection", {})
            if cicd_info:
                lines.append(f"CI/CD: {', '.join(cicd_info.get('tools', []))}")
                lines.append(f"Dockerized: {cicd_info.get('has_dockerfile', False)}")

        if "git_analysis" in data_sources:
            git_info = rd.get("git_analysis", {})
            if git_info:
                lines.append(f"Contributors: {git_info.get('contributor_count', 0)}")
                lines.append(f"Uses PRs: {git_info.get('uses_pull_requests', False)}")

        if "secret_scanner" in data_sources:
            secret_info = rd.get("secret_scanner", {})
            if secret_info:
                lines.append(f"Secret patterns detected: {secret_info.get('finding_count', 0)}")

    return "\n".join(lines)


async def _maybe_await(value: object) -> None:
    if asyncio.iscoroutine(value):
        await value
