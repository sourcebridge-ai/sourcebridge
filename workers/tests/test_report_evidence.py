# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Tests for the evidence registry."""

from workers.reports.evidence_registry import EvidenceRegistry


def test_evidence_id_generation():
    """Evidence IDs follow the E-{PREFIX}-{NN} pattern."""
    reg = EvidenceRegistry()
    eid1 = reg.add("security", "No auth")
    eid2 = reg.add("security", "XSS")
    eid3 = reg.add("engineering", "No tests")

    assert eid1 == "E-SEC-01"
    assert eid2 == "E-SEC-02"
    assert eid3 == "E-ENG-01"


def test_evidence_count():
    reg = EvidenceRegistry()
    assert reg.count() == 0
    reg.add("security", "Finding 1")
    reg.add("security", "Finding 2")
    assert reg.count() == 2


def test_evidence_items_by_category():
    reg = EvidenceRegistry()
    reg.add("security", "A", severity="critical")
    reg.add("security", "B", severity="high")
    reg.add("engineering", "C", severity="medium")

    by_cat = reg.items_by_category()
    assert len(by_cat["security"]) == 2
    assert len(by_cat["engineering"]) == 1
    assert by_cat["security"][0].severity == "critical"


def test_evidence_marker():
    reg = EvidenceRegistry()
    eid = reg.add("security", "Test")
    marker = reg.marker(eid)
    assert marker == "**[E-SEC-01]**"


def test_evidence_all_items():
    reg = EvidenceRegistry()
    reg.add("security", "A", file_path="src/auth.ts", line_start=10, line_end=20)
    items = reg.all_items()
    assert len(items) == 1
    assert items[0].file_path == "src/auth.ts"
    assert items[0].line_start == 10


def test_evidence_empty_category():
    """Unknown categories get a 3-letter prefix."""
    reg = EvidenceRegistry()
    eid = reg.add("custom_category", "Something")
    assert eid.startswith("E-CUS-")
