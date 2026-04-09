# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Tests for ModelCapabilityRegistry and StrategySelector."""

from __future__ import annotations

from dataclasses import dataclass

from workers.comprehension.capabilities import (
    ModelCapabilities,
    ModelCapabilityRegistry,
)
from workers.comprehension.corpus import CorpusSource
from workers.comprehension.selector import StrategySelector
from workers.comprehension.strategy import CapabilityRequirements


@dataclass
class _FakeStrategy:
    """Minimal ComprehensionStrategy stand-in for selector tests."""

    name: str
    reqs: CapabilityRequirements

    def capability_requirements(self) -> CapabilityRequirements:
        return self.reqs

    async def build_tree(
        self,
        corpus: CorpusSource,
        *,
        progress=None,
    ) -> None:
        raise NotImplementedError


def _hierarchical_like() -> _FakeStrategy:
    return _FakeStrategy(
        name="hierarchical",
        reqs=CapabilityRequirements(
            min_context_tokens=2048,
            min_instruction_following="low",
        ),
    )


def _long_context_like() -> _FakeStrategy:
    return _FakeStrategy(
        name="long_context_direct",
        reqs=CapabilityRequirements(
            min_context_tokens=32_000,
            min_instruction_following="medium",
        ),
    )


def _graphrag_like() -> _FakeStrategy:
    return _FakeStrategy(
        name="graphrag",
        reqs=CapabilityRequirements(
            min_context_tokens=8_192,
            min_instruction_following="high",
            requires_json_mode=True,
        ),
    )


def test_registry_knows_claude_sonnet() -> None:
    reg = ModelCapabilityRegistry()
    caps = reg.lookup("claude-sonnet-4-6")
    assert caps.provider == "anthropic"
    assert caps.effective_context_tokens >= 128_000
    assert caps.instruction_following == "high"


def test_registry_unknown_model_falls_back_to_conservative_profile() -> None:
    reg = ModelCapabilityRegistry()
    caps = reg.lookup("totally-made-up:latest")
    assert caps.source == "fallback"
    assert caps.effective_context_tokens == 4096
    assert caps.instruction_following == "low"


def test_selector_prefers_first_viable() -> None:
    """Cloud model — the chain's first entry (long_context) passes."""
    reg = ModelCapabilityRegistry()
    strategies = {
        "long_context_direct": _long_context_like(),
        "hierarchical": _hierarchical_like(),
    }
    selector = StrategySelector(registry=reg)
    result = selector.select(
        strategies=strategies,
        preference_chain=["long_context_direct", "hierarchical"],
        model_id="claude-sonnet-4-6",
    )
    assert result.strategy_name == "long_context_direct"
    assert result.trace.selected == "long_context_direct"
    assert result.trace.steps[0].status == "selected"


def test_selector_falls_through_to_hierarchical_on_small_model() -> None:
    """Ollama small model — long_context should skip, hierarchical runs."""
    reg = ModelCapabilityRegistry()
    strategies = {
        "long_context_direct": _long_context_like(),
        "hierarchical": _hierarchical_like(),
    }
    selector = StrategySelector(registry=reg)
    result = selector.select(
        strategies=strategies,
        preference_chain=["long_context_direct", "hierarchical"],
        model_id="llama3:latest",  # 4K effective context
    )
    assert result.strategy_name == "hierarchical"
    assert result.trace.steps[0].status == "skipped_capability"
    assert "context" in result.trace.steps[0].reason.lower()
    assert result.trace.steps[1].status == "selected"


def test_selector_reports_all_skipped_when_no_strategy_fits() -> None:
    """Only graphrag in chain, but small model can't satisfy json_mode."""
    reg = ModelCapabilityRegistry()
    strategies = {"graphrag": _graphrag_like()}
    selector = StrategySelector(registry=reg)
    result = selector.select(
        strategies=strategies,
        preference_chain=["graphrag"],
        model_id="llama3:latest",
    )
    assert result.strategy is None
    assert result.strategy_name == ""
    assert result.trace.steps[0].status == "skipped_capability"


def test_selector_records_unknown_strategy_entries() -> None:
    """Chain entries that aren't registered should be recorded, not crash."""
    reg = ModelCapabilityRegistry()
    strategies = {"hierarchical": _hierarchical_like()}
    selector = StrategySelector(registry=reg)
    result = selector.select(
        strategies=strategies,
        preference_chain=["totally_new", "hierarchical"],
        model_id="claude-sonnet-4-6",
    )
    assert result.strategy_name == "hierarchical"
    assert result.trace.steps[0].status == "skipped_unknown"
    assert "not registered" in result.trace.steps[0].reason


def test_selector_trace_summary_is_one_liner() -> None:
    reg = ModelCapabilityRegistry()
    strategies = {"hierarchical": _hierarchical_like()}
    selector = StrategySelector(registry=reg)
    result = selector.select(
        strategies=strategies,
        preference_chain=["hierarchical"],
        model_id="claude-sonnet-4-6",
    )
    summary = result.trace.summary()
    assert "hierarchical" in summary
    assert "claude-sonnet-4-6" in summary
    assert "\n" not in summary


def test_selector_checks_instruction_following_grade() -> None:
    """High-grade requirement against a medium-grade model should skip."""
    reg = ModelCapabilityRegistry(
        overrides={
            "custom:medium": ModelCapabilities(
                model_id="custom:medium",
                declared_context_tokens=32_000,
                effective_context_tokens=32_000,
                instruction_following="medium",
            ),
        }
    )
    strategies = {"graphrag": _graphrag_like()}
    selector = StrategySelector(registry=reg)
    result = selector.select(
        strategies=strategies,
        preference_chain=["graphrag"],
        model_id="custom:medium",
    )
    assert result.strategy is None
    assert "instruction_following" in result.trace.steps[0].reason


def test_selector_checks_json_mode_requirement() -> None:
    """Strategy needs json_mode but model has none."""
    reg = ModelCapabilityRegistry(
        overrides={
            "custom:no-json": ModelCapabilities(
                model_id="custom:no-json",
                declared_context_tokens=32_000,
                effective_context_tokens=32_000,
                instruction_following="high",
                json_mode="none",
            ),
        }
    )
    strategies = {"graphrag": _graphrag_like()}
    selector = StrategySelector(registry=reg)
    result = selector.select(
        strategies=strategies,
        preference_chain=["graphrag"],
        model_id="custom:no-json",
    )
    assert result.strategy is None
    assert "json_mode" in result.trace.steps[0].reason
