# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Report assembler.

Combines generated sections into a final Markdown document with:
- Cover page
- Table of contents
- Ordered sections by category
- Evidence appendices
"""

from __future__ import annotations

from datetime import datetime, timezone

from workers.reports.evidence_registry import EvidenceRegistry
from workers.reports.section_generator import GeneratedSection


def assemble_report(
    *,
    report_name: str,
    report_type: str,
    audience_title: str,
    repo_names: list[str],
    sections: list[GeneratedSection],
    evidence: EvidenceRegistry,
    include_diagrams: bool = False,
    branding_footer: str = "",
) -> str:
    """Assemble sections into a complete Markdown report."""
    parts: list[str] = []

    # Cover page
    parts.append(_build_cover_page(
        report_name=report_name,
        report_type=report_type,
        audience_title=audience_title,
        repo_names=repo_names,
        branding_footer=branding_footer,
    ))

    # Table of contents
    parts.append(_build_toc(sections))

    # Sections in order
    for section in sections:
        parts.append(section.markdown)
        parts.append("")  # blank line between sections

    # Evidence appendices
    evidence_md = _build_evidence_appendices(evidence)
    if evidence_md:
        parts.append(evidence_md)

    return "\n\n".join(parts)


def _build_cover_page(
    report_name: str,
    report_type: str,
    audience_title: str,
    repo_names: list[str],
    branding_footer: str,
) -> str:
    type_title = report_type.replace("_", " ").title()
    date_str = datetime.now(timezone.utc).strftime("%B %d, %Y")
    repos_str = ", ".join(repo_names) if repo_names else "No repositories selected"

    lines = [
        f"# {report_name}",
        "",
        f"**Report Type:** {type_title}",
        f"**Audience:** {audience_title}",
        f"**Repositories:** {repos_str}",
        f"**Generated:** {date_str}",
    ]
    if branding_footer:
        lines.append(f"**Prepared by:** {branding_footer}")
    lines.append("")
    lines.append("---")
    return "\n".join(lines)


def _build_toc(sections: list[GeneratedSection]) -> str:
    lines = ["## Table of Contents", ""]
    current_category = ""
    for i, section in enumerate(sections, 1):
        if section.category != current_category:
            current_category = section.category
            lines.append(f"**{current_category}**")
        # Generate anchor-friendly slug
        slug = section.title.lower().replace(" ", "-").replace("(", "").replace(")", "")
        slug = "".join(c for c in slug if c.isalnum() or c == "-")
        lines.append(f"  {i}. [{section.title}](#{slug})")
    lines.append("")
    lines.append("---")
    return "\n".join(lines)


def _build_evidence_appendices(evidence: EvidenceRegistry) -> str:
    if evidence.count() == 0:
        return ""

    lines = ["---", "", "# Appendices — Evidence Registry", ""]
    by_category = evidence.items_by_category()

    appendix_letter = ord("A")
    for category, items in sorted(by_category.items()):
        letter = chr(appendix_letter)
        appendix_letter += 1
        lines.append(f"## Appendix {letter}: {category.title()} Evidence")
        lines.append("")

        for item in items:
            lines.append(f"### {item.evidence_id}: {item.title}")
            if item.description:
                lines.append(f"{item.description}")
            lines.append("")
            if item.severity and item.severity != "info":
                lines.append(f"**Severity:** {item.severity.upper()}")
            if item.source_repo_id:
                lines.append(f"**Repository:** {item.source_repo_id}")
            if item.file_path:
                loc = f"**File:** `{item.file_path}`"
                if item.line_start:
                    loc += f" (lines {item.line_start}"
                    if item.line_end:
                        loc += f"-{item.line_end}"
                    loc += ")"
                lines.append(loc)
            if item.code_snippet:
                lines.append(f"\n```\n{item.code_snippet}\n```")
            lines.append("")

    return "\n".join(lines)


def count_words(markdown: str) -> int:
    """Count words in a Markdown string, excluding formatting."""
    return len(markdown.split())
