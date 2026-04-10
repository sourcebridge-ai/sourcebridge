# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Tests for report prompt configuration."""

from workers.reports.prompts.audience import (
    AUDIENCES,
    audience_prompt_block,
    get_audience,
)


def test_all_audiences_defined():
    """All 6 audience presets are defined."""
    assert len(AUDIENCES) == 6
    expected = {"c_suite", "executive", "technical_leadership", "developer", "compliance", "non_technical"}
    assert set(AUDIENCES.keys()) == expected


def test_audience_has_required_fields():
    """Every audience has all required fields populated."""
    for key, aud in AUDIENCES.items():
        assert aud.key == key, f"{key} has mismatched key"
        assert aud.title, f"{key} missing title"
        assert aud.language, f"{key} missing language"
        assert aud.depth, f"{key} missing depth"
        assert aud.recommendations, f"{key} missing recommendations"
        assert aud.metrics, f"{key} missing metrics"
        assert aud.sample, f"{key} missing sample"


def test_get_audience_default():
    """Unknown audience falls back to technical_leadership."""
    aud = get_audience("nonexistent")
    assert aud.key == "technical_leadership"


def test_audience_prompt_block_format():
    """Prompt block contains audience name and all instruction sections."""
    block = audience_prompt_block("c_suite")
    assert "C-Suite / Board" in block
    assert "Language:" in block
    assert "Detail level:" in block
    assert "Recommendations style:" in block
    assert "Metrics style:" in block


def test_c_suite_no_jargon_instruction():
    """C-Suite audience explicitly forbids jargon."""
    aud = get_audience("c_suite")
    assert "jargon" in aud.language.lower()


def test_developer_file_paths_instruction():
    """Developer audience explicitly includes file paths."""
    aud = get_audience("developer")
    assert "file path" in aud.depth.lower()


def test_compliance_formal_instruction():
    """Compliance audience uses formal language."""
    aud = get_audience("compliance")
    assert "formal" in aud.language.lower()
