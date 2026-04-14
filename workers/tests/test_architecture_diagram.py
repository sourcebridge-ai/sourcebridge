# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

from __future__ import annotations

from workers.common.llm.provider import LLMResponse
from workers.knowledge.architecture_diagram import generate_architecture_diagram


class _SequenceProvider:
    def __init__(self, responses: list[str]) -> None:
        self._responses = responses
        self.calls: list[str] = []

    async def complete(
        self,
        prompt: str,
        *,
        system: str = "",
        max_tokens: int = 4096,
        temperature: float = 0.0,
        frequency_penalty: float = 0.0,
        model: str | None = None,
    ) -> LLMResponse:
        self.calls.append(prompt)
        content = self._responses.pop(0)
        return LLMResponse(
            content=content,
            model=model or "test-model",
            input_tokens=100,
            output_tokens=50,
            provider_name="test",
        )

    async def stream(self, *args, **kwargs):  # pragma: no cover
        raise NotImplementedError


async def test_generate_architecture_diagram_retries_invalid_mermaid() -> None:
    provider = _SequenceProvider(
        [
            "I think this system has an API and a database.",
            'flowchart LR\napi["API"] --> db["DB"]',
        ]
    )
    result, usage = await generate_architecture_diagram(
        provider,
        repository_name="Example Repo",
        audience="developer",
        depth="medium",
        snapshot_json='{"repository_name":"Example Repo"}',
        deterministic_diagram_json='{"modules":[{"path":"api","outbound_paths":["db"]}]}',
        model_override="test-model",
    )

    assert len(provider.calls) == 2
    assert result["mermaid_source"].startswith("flowchart LR")
    assert "retry regenerated invalid Mermaid" in result["repair_summary"]
    assert usage.input_tokens == 200
    assert usage.output_tokens == 100
