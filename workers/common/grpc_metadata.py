"""Utilities for reading SourceBridge gRPC metadata."""

from __future__ import annotations

from dataclasses import dataclass

import grpc


def resolve_model_override(context: grpc.aio.ServicerContext) -> str | None:
    """Read model override from gRPC metadata, return None to use default.

    The Go API attaches 'x-sb-model' metadata when advanced mode is on.
    """
    if not hasattr(context, "invocation_metadata"):
        return None
    for key, value in context.invocation_metadata():
        if key == "x-sb-model":
            return value or None
    return None


@dataclass
class RuntimeLLMOverride:
    provider: str = ""
    base_url: str = ""
    api_key: str = ""
    model: str = ""
    draft_model: str = ""
    operation: str = ""

    def is_empty(self) -> bool:
        return not any(
            (
                self.provider,
                self.base_url,
                self.api_key,
                self.model,
                self.draft_model,
                self.operation,
            )
        )


def resolve_llm_override(context: grpc.aio.ServicerContext) -> RuntimeLLMOverride | None:
    """Read the API-provided effective LLM config from gRPC metadata."""
    if not hasattr(context, "invocation_metadata"):
        return None
    override = RuntimeLLMOverride()
    for key, value in context.invocation_metadata():
        if key == "x-sb-llm-provider":
            override.provider = value or ""
        elif key == "x-sb-llm-base-url":
            override.base_url = value or ""
        elif key == "x-sb-llm-api-key":
            override.api_key = value or ""
        elif key == "x-sb-model":
            override.model = value or ""
        elif key == "x-sb-llm-draft-model":
            override.draft_model = value or ""
        elif key == "x-sb-operation":
            override.operation = value or ""
    if override.is_empty():
        return None
    return override
