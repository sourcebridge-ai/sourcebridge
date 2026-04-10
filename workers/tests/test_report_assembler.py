# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Tests for the report assembler."""

from workers.reports.assembler import assemble_report, count_words
from workers.reports.evidence_registry import EvidenceRegistry
from workers.reports.section_generator import GeneratedSection


def _make_section(key: str, title: str, category: str, content: str = "") -> GeneratedSection:
    md = content or f"## {title}\n\nThis is the {title} section content."
    return GeneratedSection(
        key=key, title=title, category=category,
        markdown=md, word_count=len(md.split()),
    )


def test_assemble_basic_report():
    """Assembled report has cover page, TOC, and sections."""
    sections = [
        _make_section("exec", "Executive Summary", "Executive"),
        _make_section("testing", "Testing", "Delivery"),
    ]
    evidence = EvidenceRegistry()

    md = assemble_report(
        report_name="Test Report",
        report_type="architecture_baseline",
        audience_title="Technical Leadership",
        repo_names=["Repo A", "Repo B"],
        sections=sections,
        evidence=evidence,
    )

    assert "# Test Report" in md
    assert "Architecture Baseline" in md
    assert "Technical Leadership" in md
    assert "Repo A, Repo B" in md
    assert "## Table of Contents" in md
    assert "Executive Summary" in md
    assert "Testing" in md


def test_assemble_with_evidence():
    """Assembled report includes evidence appendices."""
    sections = [_make_section("sec", "Security", "Security")]
    evidence = EvidenceRegistry()
    evidence.add("security", "No auth on /api/lookup", severity="critical", file_path="src/api.ts")
    evidence.add("security", "XSS via rehype-raw", severity="high")

    md = assemble_report(
        report_name="Security Report",
        report_type="architecture_baseline",
        audience_title="Developer",
        repo_names=["App1"],
        sections=sections,
        evidence=evidence,
    )

    assert "Evidence Registry" in md
    assert "E-SEC-01" in md
    assert "No auth on /api/lookup" in md
    assert "CRITICAL" in md
    assert "src/api.ts" in md


def test_assemble_empty_evidence():
    """No appendices when evidence is empty."""
    sections = [_make_section("test", "Testing", "Delivery")]
    evidence = EvidenceRegistry()

    md = assemble_report(
        report_name="Test",
        report_type="swot",
        audience_title="Developer",
        repo_names=[],
        sections=sections,
        evidence=evidence,
    )

    assert "Evidence Registry" not in md


def test_toc_contains_all_sections():
    """TOC lists every section with category grouping."""
    sections = [
        _make_section("a", "Section A", "Category 1"),
        _make_section("b", "Section B", "Category 1"),
        _make_section("c", "Section C", "Category 2"),
    ]
    evidence = EvidenceRegistry()

    md = assemble_report(
        report_name="TOC Test",
        report_type="environment_eval",
        audience_title="Executive",
        repo_names=["R1"],
        sections=sections,
        evidence=evidence,
    )

    assert "**Category 1**" in md
    assert "**Category 2**" in md
    assert "Section A" in md
    assert "Section B" in md
    assert "Section C" in md


def test_count_words():
    assert count_words("hello world") == 2
    assert count_words("") == 0
    assert count_words("one two three four five") == 5


def test_branding_footer():
    """Branding footer appears on cover page."""
    sections = [_make_section("x", "X", "Y")]
    evidence = EvidenceRegistry()

    md = assemble_report(
        report_name="Branded",
        report_type="swot",
        audience_title="C-Suite",
        repo_names=[],
        sections=sections,
        evidence=evidence,
        branding_footer="Hoegg Software, Co.",
    )

    assert "Hoegg Software, Co." in md
