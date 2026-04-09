"""LLM provider protocol and response types."""

from __future__ import annotations

from collections.abc import AsyncIterator
from dataclasses import dataclass
from typing import Protocol


@dataclass
class LLMResponse:
    """Response from an LLM provider."""

    content: str
    model: str
    input_tokens: int = 0
    output_tokens: int = 0
    stop_reason: str = ""
    tokens_per_second: float | None = None
    generation_time_ms: float | None = None
    acceptance_rate: float | None = None
    provider_name: str | None = None


class LLMProvider(Protocol):
    """Protocol for LLM providers."""

    async def complete(
        self,
        prompt: str,
        *,
        system: str = "",
        max_tokens: int = 4096,
        temperature: float = 0.0,
        model: str | None = None,
    ) -> LLMResponse:
        """Generate a completion. If model is provided, it overrides the default."""
        ...

    async def stream(
        self,
        prompt: str,
        *,
        system: str = "",
        max_tokens: int = 4096,
        temperature: float = 0.0,
        model: str | None = None,
    ) -> AsyncIterator[str]:
        """Stream a completion token by token."""
        ...


class LLMEmptyResponseError(RuntimeError):
    """Raised when a provider returns an empty or whitespace-only response."""

    def __init__(self, response: LLMResponse, context: str):
        self.response = response
        self.context = context
        super().__init__(
            "LLM returned empty content "
            f"(context={context}, model={response.model}, input_tokens={response.input_tokens}, "
            f"stop_reason={response.stop_reason})"
        )


def require_nonempty(response: LLMResponse, context: str) -> LLMResponse:
    """Reject empty completions so callers fail explicitly instead of fabricating success."""
    if not response.content or not response.content.strip():
        raise LLMEmptyResponseError(response, context)
    return response


async def complete_with_optional_model(
    provider: LLMProvider,
    prompt: str,
    *,
    system: str = "",
    max_tokens: int = 4096,
    temperature: float = 0.0,
    model: str | None = None,
) -> LLMResponse:
    """Call provider.complete while remaining compatible with legacy test doubles.

    Some local test providers do not accept the optional ``model`` kwarg yet.
    """
    if model is None:
        return await provider.complete(
            prompt,
            system=system,
            max_tokens=max_tokens,
            temperature=temperature,
        )
    return await provider.complete(
        prompt,
        system=system,
        max_tokens=max_tokens,
        temperature=temperature,
        model=model,
    )
