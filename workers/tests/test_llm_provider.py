# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Tests for shared LLM provider helpers."""

from workers.common.llm.provider import LLMEmptyResponseError, LLMResponse, require_nonempty


def test_require_nonempty_returns_response_when_content_present() -> None:
    response = LLMResponse(content="hello", model="fake")
    assert require_nonempty(response, context="test") is response


def test_require_nonempty_raises_for_blank_content() -> None:
    response = LLMResponse(content="   ", model="fake", input_tokens=12, stop_reason="stop")

    try:
        require_nonempty(response, context="cliff_notes:repository")
    except LLMEmptyResponseError as exc:
        assert exc.context == "cliff_notes:repository"
        assert exc.response is response
        assert "LLM returned empty content" in str(exc)
    else:
        raise AssertionError("expected LLMEmptyResponseError")
