# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Evidence registry for reports.

Tracks evidence items collected during report generation, assigns
unique IDs (e.g. E-SEC-01), and groups them into appendices. Every
claim in the report body references evidence via inline markers.
"""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class EvidenceItem:
    """A single piece of evidence backing a report claim."""

    evidence_id: str = ""
    category: str = ""  # "security", "engineering", "operations", etc.
    title: str = ""
    description: str = ""
    source_type: str = ""  # "owasp_scan", "code_analysis", "dependency_audit", etc.
    source_repo_id: str = ""
    file_path: str = ""
    line_start: int = 0
    line_end: int = 0
    code_snippet: str = ""
    raw_data: str = ""
    severity: str = "info"  # "critical", "high", "medium", "low", "info"


_CATEGORY_PREFIXES = {
    "security": "SEC",
    "architecture": "ARC",
    "engineering": "ENG",
    "operations": "OPS",
    "access": "ACC",
    "data": "DAT",
    "delivery": "DEL",
    "governance": "GOV",
    "compliance": "CMP",
    "integration": "INT",
    "users": "USR",
    "review": "REV",
}


class EvidenceRegistry:
    """Collects evidence items and assigns sequential IDs.

    Usage:
        registry = EvidenceRegistry()
        eid = registry.add("security", "No auth on endpoint", ...)
        # eid == "E-SEC-01"
        # Use [E-SEC-01] as an inline marker in the report body
    """

    def __init__(self) -> None:
        self._items: list[EvidenceItem] = []
        self._counters: dict[str, int] = {}

    def add(
        self,
        category: str,
        title: str,
        description: str = "",
        source_type: str = "",
        source_repo_id: str = "",
        file_path: str = "",
        line_start: int = 0,
        line_end: int = 0,
        code_snippet: str = "",
        raw_data: str = "",
        severity: str = "info",
    ) -> str:
        """Add an evidence item and return its unique ID (e.g. E-SEC-01)."""
        prefix = _CATEGORY_PREFIXES.get(category.lower(), category[:3].upper())
        count = self._counters.get(prefix, 0) + 1
        self._counters[prefix] = count
        eid = f"E-{prefix}-{count:02d}"

        self._items.append(EvidenceItem(
            evidence_id=eid,
            category=category,
            title=title,
            description=description,
            source_type=source_type,
            source_repo_id=source_repo_id,
            file_path=file_path,
            line_start=line_start,
            line_end=line_end,
            code_snippet=code_snippet,
            raw_data=raw_data,
            severity=severity,
        ))
        return eid

    def all_items(self) -> list[EvidenceItem]:
        """Return all collected evidence items."""
        return list(self._items)

    def items_by_category(self) -> dict[str, list[EvidenceItem]]:
        """Group evidence items by category for appendix generation."""
        groups: dict[str, list[EvidenceItem]] = {}
        for item in self._items:
            groups.setdefault(item.category, []).append(item)
        return groups

    def count(self) -> int:
        return len(self._items)

    def marker(self, evidence_id: str) -> str:
        """Return the inline marker string for an evidence ID."""
        return f"**[{evidence_id}]**"
