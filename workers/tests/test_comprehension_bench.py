# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Tests for the OSS-safe comprehension benchmark runner."""

from __future__ import annotations

import json

import pytest

from workers.benchmarks.run_comprehension_bench import _load_manifest, _run_case


def test_benchmark_manifest_loads_cases() -> None:
    cases = _load_manifest()
    assert len(cases) >= 4
    assert {case["artifact_type"] for case in cases} >= {
        "cliff_notes",
        "learning_path",
        "code_tour",
        "workflow_story",
    }


@pytest.mark.asyncio
async def test_benchmark_case_runs_successfully() -> None:
    case = {
        "id": "fixture_cliff_notes_fake",
        "corpus_id": "multi-lang-repo-fixture",
        "artifact_type": "cliff_notes",
        "provider_mode": "fake",
        "repository_name": "multi-lang-repo",
        "audience": "developer",
        "depth": "medium",
        "scope_type": "repository",
        "scope_path": "",
        "expected_checks": [
            "cliff_notes_required_sections",
            "cliff_notes_has_evidence",
        ],
    }

    result = await _run_case(case)

    assert result.success is True
    assert result.metrics["section_count"] >= 7
    assert result.checks["cliff_notes_required_sections"] is True
    assert result.input_tokens > 0


def test_benchmark_result_serializes_to_json() -> None:
    payload = {
        "case_id": "fixture_cliff_notes_fake",
        "corpus_id": "multi-lang-repo-fixture",
        "artifact_type": "cliff_notes",
        "provider_mode": "fake",
        "provider_name": "fake",
        "model_id": "fake-test-model",
        "success": True,
        "duration_ms": 10,
        "input_tokens": 20,
        "output_tokens": 30,
        "error": None,
        "checks": {"cliff_notes_required_sections": True},
        "metrics": {"section_count": 7},
    }
    encoded = json.dumps(payload)
    assert "fixture_cliff_notes_fake" in encoded
