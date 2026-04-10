# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Integration tests for the report engine using FakeLLMProvider."""

import os
import tempfile

import pytest

from workers.common.llm.provider import LLMResponse
from workers.reports.engine import ReportConfig, generate_report


class FakeReportLLMProvider:
    """Deterministic LLM provider for report tests."""

    def __init__(self):
        self.call_count = 0

    async def complete(self, prompt, *, system="", max_tokens=4096, temperature=0.0, model=None):
        self.call_count += 1
        # Generate a plausible section based on the prompt
        title = "Section"
        if '"' in prompt:
            # Extract section title from prompt
            start = prompt.index('"') + 1
            end = prompt.index('"', start)
            title = prompt[start:end]

        content = (
            f"## {title}\n\n"
            f"This is the generated content for {title}. "
            f"The portfolio consists of multiple repositories with varying "
            f"levels of test coverage and documentation. The applications "
            f"use a mix of Next.js and Python frameworks. Security scanning "
            f"revealed several areas requiring attention, including "
            f"authentication patterns and dependency management. "
            f"The deployment architecture varies across applications, "
            f"with some using managed cloud services and others using "
            f"traditional server hosting. Overall, the portfolio shows "
            f"partial standardization with room for improvement."
        )
        return LLMResponse(
            content=content,
            model=model or "fake-model",
            input_tokens=500,
            output_tokens=100,
        )

    async def stream(self, prompt, *, system="", max_tokens=4096, temperature=0.0, model=None):
        raise NotImplementedError


@pytest.mark.asyncio
async def test_generate_report_basic():
    """Full pipeline runs without crashes for a basic config."""
    provider = FakeReportLLMProvider()
    with tempfile.TemporaryDirectory() as tmpdir:
        config = ReportConfig(
            report_id="test-001",
            report_name="Test Architecture Baseline",
            report_type="architecture_baseline",
            audience="technical_leadership",
            repository_ids=["repo-1"],
            selected_sections=["applications_inventory", "testing", "deployment_architecture"],
            output_dir=tmpdir,
        )

        result = await generate_report(
            provider,
            config,
            repo_data={
                "repo-1": {
                    "name": "MACU Helpdesk",
                    "file_count": 21,
                    "symbol_count": 45,
                    "languages": ["TypeScript", "JavaScript"],
                },
            },
        )

        assert result.section_count == 3
        assert result.word_count > 50
        assert result.content_dir == tmpdir
        assert os.path.exists(os.path.join(tmpdir, "report.md"))
        assert os.path.exists(os.path.join(tmpdir, "evidence.json"))

        # Read the generated markdown
        with open(os.path.join(tmpdir, "report.md")) as f:
            md = f.read()
        assert "Test Architecture Baseline" in md
        assert "Table of Contents" in md


@pytest.mark.asyncio
async def test_generate_report_multi_repo():
    """Multi-repo report includes all repo names in context."""
    provider = FakeReportLLMProvider()
    with tempfile.TemporaryDirectory() as tmpdir:
        config = ReportConfig(
            report_id="test-002",
            report_name="Multi-Repo Review",
            report_type="swot",
            audience="c_suite",
            repository_ids=["repo-1", "repo-2", "repo-3"],
            selected_sections=["strengths", "weaknesses"],
            output_dir=tmpdir,
        )

        result = await generate_report(
            provider,
            config,
            repo_data={
                "repo-1": {"name": "App A"},
                "repo-2": {"name": "App B"},
                "repo-3": {"name": "App C"},
            },
        )

        assert result.section_count == 2
        with open(os.path.join(tmpdir, "report.md")) as f:
            md = f.read()
        assert "App A" in md
        assert "App B" in md
        assert "App C" in md


@pytest.mark.asyncio
async def test_generate_report_terminal_sections_last():
    """Executive summary (depends on *) is generated last."""
    call_order = []

    class OrderTrackingProvider:
        async def complete(self, prompt, *, system="", max_tokens=4096, temperature=0.0, model=None):
            if '"' in prompt:
                start = prompt.index('"') + 1
                end = prompt.index('"', start)
                call_order.append(prompt[start:end])
            return LLMResponse(content="## Section\n\nContent here.", model="fake", input_tokens=100, output_tokens=50)

        async def stream(self, *a, **kw):
            raise NotImplementedError

    with tempfile.TemporaryDirectory() as tmpdir:
        config = ReportConfig(
            report_id="test-003",
            report_name="Order Test",
            report_type="architecture_baseline",
            audience="developer",
            repository_ids=["r1"],
            selected_sections=["executive_summary", "testing", "deployment_architecture"],
            output_dir=tmpdir,
        )

        # Provide section definitions with dependencies
        section_defs = [
            {"key": "executive_summary", "title": "Executive Summary", "category": "Executive", "depends_on": ["*"], "min_word_count": 100},
            {"key": "testing", "title": "Testing", "category": "Delivery", "depends_on": [], "min_word_count": 100},
            {"key": "deployment_architecture", "title": "Deployment Architecture", "category": "Operations", "depends_on": [], "min_word_count": 100},
        ]

        await generate_report(
            OrderTrackingProvider(),
            config,
            section_definitions=section_defs,
        )

        # Executive Summary should be last
        assert call_order[-1] == "Executive Summary"
        assert "Testing" in call_order[:-1]
        assert "Deployment Architecture" in call_order[:-1]


@pytest.mark.asyncio
async def test_generate_report_no_data():
    """Report generates even with no repo data (uses placeholders in context)."""
    provider = FakeReportLLMProvider()
    with tempfile.TemporaryDirectory() as tmpdir:
        config = ReportConfig(
            report_id="test-004",
            report_name="No Data Report",
            report_type="environment_eval",
            audience="non_technical",
            repository_ids=["r1"],
            selected_sections=["tech_stack"],
            output_dir=tmpdir,
        )

        result = await generate_report(provider, config)
        assert result.section_count == 1
        assert result.word_count > 0


@pytest.mark.asyncio
async def test_progress_callback():
    """Progress callback is called with increasing values."""
    progress_calls = []

    async def track_progress(p, phase, msg):
        progress_calls.append((p, phase, msg))

    provider = FakeReportLLMProvider()
    with tempfile.TemporaryDirectory() as tmpdir:
        config = ReportConfig(
            report_id="test-005",
            report_name="Progress Test",
            report_type="swot",
            audience="developer",
            repository_ids=["r1"],
            selected_sections=["strengths", "weaknesses"],
            output_dir=tmpdir,
        )

        await generate_report(provider, config, progress=track_progress)

        assert len(progress_calls) >= 4  # at least: collecting, generating x2, assembling, rendering, ready
        # Progress should be monotonically non-decreasing
        for i in range(1, len(progress_calls)):
            assert progress_calls[i][0] >= progress_calls[i - 1][0], (
                f"Progress decreased: {progress_calls[i - 1][0]} -> {progress_calls[i][0]}"
            )
        # Last call should be 1.0
        assert progress_calls[-1][0] == 1.0
