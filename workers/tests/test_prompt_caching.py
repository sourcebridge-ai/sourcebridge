"""Unit tests for Phase-1 prompt caching on the agentic tool-use path.

Validates the Anthropic payload builder attaches cache_control markers
to the right blocks when prompt caching is enabled, and leaves them
off when disabled. No live API call.
"""

from __future__ import annotations

from workers.common.llm.tools import (
    AgentMessage,
    ToolSchema,
    build_anthropic_kwargs,
)


def _sample_messages() -> list[AgentMessage]:
    return [
        AgentMessage(role="system", text="You are a QA assistant."),
        AgentMessage(role="user", text="what does the orchestrator do?"),
    ]


def _sample_tools() -> list[ToolSchema]:
    return [
        ToolSchema(
            name="search_evidence",
            description="search for evidence",
            input_schema_json='{"type":"object","properties":{"query":{"type":"string"}}}',
        ),
        ToolSchema(
            name="read_file",
            description="read a file",
            input_schema_json='{"type":"object","properties":{"path":{"type":"string"}}}',
        ),
    ]


def test_prompt_caching_attaches_to_system_and_last_tool():
    """cache_control must land on (a) system prompt text block and
    (b) the LAST tool definition. Anthropic caches up to and including
    the last marker, so this produces the "all tools + system" cache
    prefix the Phase 1 plan targets."""
    kwargs = build_anthropic_kwargs(
        model="claude-sonnet-4-5",
        messages=_sample_messages(),
        tools=_sample_tools(),
        max_tokens=2048,
        enable_prompt_caching=True,
    )
    system = kwargs["system"]
    assert isinstance(system, list), "system must be a content-block array when caching is on"
    assert len(system) == 1
    assert system[0]["type"] == "text"
    assert system[0]["cache_control"] == {"type": "ephemeral"}

    tools = kwargs["tools"]
    assert len(tools) == 2
    assert "cache_control" not in tools[0], "first tool must NOT carry cache_control"
    assert tools[1]["cache_control"] == {"type": "ephemeral"}, "last tool must carry cache_control"


def test_prompt_caching_disabled_leaves_payload_plain():
    """With caching off, payload has no cache_control markers and
    `system` is the plain string form."""
    kwargs = build_anthropic_kwargs(
        model="claude-sonnet-4-5",
        messages=_sample_messages(),
        tools=_sample_tools(),
        max_tokens=2048,
        enable_prompt_caching=False,
    )
    assert isinstance(kwargs["system"], str), "system should be a plain string when caching is off"
    for t in kwargs["tools"]:
        assert "cache_control" not in t


def test_prompt_caching_no_tools_still_caches_system():
    """When there are no tools, we still cache the system prompt so
    the fixed question-handling instructions don't re-ingest each
    time."""
    kwargs = build_anthropic_kwargs(
        model="claude-sonnet-4-5",
        messages=_sample_messages(),
        tools=[],
        max_tokens=2048,
        enable_prompt_caching=True,
    )
    system = kwargs["system"]
    assert isinstance(system, list)
    assert system[0]["cache_control"] == {"type": "ephemeral"}
    assert "tools" not in kwargs or kwargs["tools"] == []


def test_prompt_caching_empty_system_omitted():
    """No system prompt → no `system` key in kwargs, caching or not."""
    messages = [AgentMessage(role="user", text="question")]
    kwargs_on = build_anthropic_kwargs(
        model="claude-sonnet-4-5",
        messages=messages,
        tools=_sample_tools(),
        max_tokens=2048,
        enable_prompt_caching=True,
    )
    assert "system" not in kwargs_on
    # But the last tool still carries cache_control so the tools list
    # gets cached on its own.
    assert kwargs_on["tools"][-1]["cache_control"] == {"type": "ephemeral"}
