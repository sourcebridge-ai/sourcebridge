"""Tool-use extension over the LLM provider layer.

Provider-neutral types for representing a tool-using conversation turn
and a small function that dispatches through Anthropic's tool_use
messages API today. Other providers return capability_supported=False
until their adapters land.

See thoughts/shared/plans/2026-04-23-agentic-retrieval-for-deep-qa.md
for the architectural context.
"""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass, field
from typing import Any, cast

logger = logging.getLogger(__name__)


@dataclass
class ToolSchema:
    """One tool definition the model may call."""

    name: str
    description: str
    input_schema_json: str  # JSON Schema as a string

    def to_anthropic(self) -> dict[str, Any]:
        return {
            "name": self.name,
            "description": self.description,
            "input_schema": json.loads(self.input_schema_json),
        }


@dataclass
class ToolCall:
    call_id: str
    name: str
    args_json: str


@dataclass
class ToolResult:
    call_id: str
    ok: bool
    data_json: str = ""
    error: str = ""
    hint: str = ""


@dataclass
class AgentMessage:
    """One turn in the conversation history.

    Exactly one of text / tool_calls / tool_results is populated at a
    time (plus role). The orchestrator accumulates these and replays
    the full history on each call.
    """

    role: str  # "system" | "user" | "assistant" | "tool_result"
    text: str = ""
    tool_calls: list[ToolCall] = field(default_factory=list)
    tool_results: list[ToolResult] = field(default_factory=list)


@dataclass
class AgentTurnResponse:
    """Worker response after running one turn against the provider."""

    capability_supported: bool
    # Assistant-role message returned by the model. Either text is
    # populated (final turn) or tool_calls is populated (tool-use
    # turn), not both.
    turn: AgentMessage
    # Token accounting.
    input_tokens: int = 0
    output_tokens: int = 0
    model: str = ""
    # Prompt-cache accounting. When prompt caching is active on the
    # provider, Anthropic reports how many tokens were served from
    # the cache vs had to be re-ingested. A high cache_read rate is
    # the point of Phase 5 caching.
    cache_creation_input_tokens: int = 0
    cache_read_input_tokens: int = 0
    # Natural-language reason when the worker short-circuited the turn
    # (e.g. a safety gate). Empty on the happy path.
    termination_hint: str = ""


def _messages_for_anthropic(messages: list[AgentMessage]) -> tuple[list[dict[str, Any]], str]:
    """Translate provider-neutral AgentMessages into Anthropic's
    messages API shape. Returns (messages_list, system_text).

    Anthropic's messages API takes a `system` param separate from the
    messages list, so we split system-role entries out.
    """
    system_parts: list[str] = []
    out: list[dict[str, Any]] = []
    for m in messages:
        if m.role == "system":
            if m.text:
                system_parts.append(m.text)
            continue
        if m.role == "user":
            out.append({"role": "user", "content": m.text})
            continue
        if m.role == "assistant":
            content: list[dict[str, Any]] = []
            if m.text:
                content.append({"type": "text", "text": m.text})
            for tc in m.tool_calls:
                try:
                    args = json.loads(tc.args_json) if tc.args_json else {}
                except json.JSONDecodeError:
                    args = {}
                content.append(
                    {
                        "type": "tool_use",
                        "id": tc.call_id,
                        "name": tc.name,
                        "input": args,
                    }
                )
            out.append({"role": "assistant", "content": content})
            continue
        if m.role == "tool_result":
            # Anthropic expects tool_result blocks inside a user-role
            # message, one block per call.
            blocks: list[dict[str, Any]] = []
            for tr in m.tool_results:
                if tr.ok:
                    payload = tr.data_json or "{}"
                else:
                    payload = json.dumps(
                        {"ok": False, "error": tr.error, "hint": tr.hint}
                    )
                blocks.append(
                    {
                        "type": "tool_result",
                        "tool_use_id": tr.call_id,
                        "content": payload,
                        "is_error": not tr.ok,
                    }
                )
            out.append({"role": "user", "content": blocks})
            continue
    return out, "\n\n".join(system_parts)


def build_anthropic_kwargs(
    model: str,
    messages: list[AgentMessage],
    tools: list[ToolSchema],
    max_tokens: int,
    enable_prompt_caching: bool,
) -> dict[str, Any]:
    """Build the kwargs dict for `client.messages.create(**kwargs)`.

    Extracted from `run_agent_turn_anthropic` so unit tests can
    assert the outgoing payload shape without mocking the Anthropic
    SDK. `enable_prompt_caching` controls whether `cache_control`
    markers are attached to the system prompt and tool list.

    Anthropic caches up to and including the LAST block that carries
    a `cache_control` marker. Placing the marker on:
      - the last tool schema, AND
      - the system prompt text block
    means the (system prompt + all N tools) prefix is cached for 5
    minutes. Second and later turns in the same loop see this prefix
    as `cache_read_input_tokens` in `usage`, typically 85–95% of
    the input-token volume for long-context agentic conversations.
    """
    api_messages, system_text = _messages_for_anthropic(messages)
    api_tools = [t.to_anthropic() for t in tools]

    kwargs: dict[str, Any] = {
        "model": model,
        "max_tokens": max_tokens or 2048,
        "messages": cast(list[dict[str, Any]], api_messages),
    }
    if system_text:
        if enable_prompt_caching:
            # Content-block form lets us attach cache_control. Anthropic
            # accepts the array form for `system` the same way it does
            # for message content.
            kwargs["system"] = [
                {
                    "type": "text",
                    "text": system_text,
                    "cache_control": {"type": "ephemeral"},
                }
            ]
        else:
            kwargs["system"] = system_text
    if api_tools:
        if enable_prompt_caching and api_tools:
            # Cache up to and including the last tool. Earlier tools
            # are included implicitly.
            api_tools = list(api_tools)
            api_tools[-1] = {
                **api_tools[-1],
                "cache_control": {"type": "ephemeral"},
            }
        kwargs["tools"] = api_tools
    return kwargs


async def run_agent_turn_anthropic(
    client: Any,
    model: str,
    messages: list[AgentMessage],
    tools: list[ToolSchema],
    max_tokens: int,
    enable_prompt_caching: bool = True,
) -> AgentTurnResponse:
    """Run one tool-use turn against Anthropic's messages API.

    Returns the assistant's response as an AgentMessage; either
    `text` or `tool_calls` is populated. Raises on hard API errors.

    `enable_prompt_caching` (default True) attaches ephemeral
    cache_control markers to the system prompt and the last tool
    schema. For Anthropic models 3.5+ this cuts input-token cost
    ~60-80% on multi-turn loops. Pass False to disable on providers
    that don't support cache markers (currently none in production,
    but kept for safety).
    """
    import anthropic  # noqa: F401 — verifies SDK is installed

    kwargs = build_anthropic_kwargs(
        model=model,
        messages=messages,
        tools=tools,
        max_tokens=max_tokens,
        enable_prompt_caching=enable_prompt_caching,
    )

    resp = await client.messages.create(**kwargs)  # type: ignore[call-overload]

    turn = AgentMessage(role="assistant")
    for block in resp.content:
        btype = getattr(block, "type", None)
        if btype == "text":
            turn.text = (turn.text + block.text) if turn.text else block.text
        elif btype == "tool_use":
            args_json = json.dumps(block.input) if block.input is not None else "{}"
            turn.tool_calls.append(
                ToolCall(
                    call_id=block.id,
                    name=block.name,
                    args_json=args_json,
                )
            )

    # Cache accounting. Older SDK versions may not expose these
    # attributes, so default to 0 on AttributeError.
    cache_creation = getattr(resp.usage, "cache_creation_input_tokens", 0) or 0
    cache_read = getattr(resp.usage, "cache_read_input_tokens", 0) or 0

    return AgentTurnResponse(
        capability_supported=True,
        turn=turn,
        input_tokens=resp.usage.input_tokens,
        output_tokens=resp.usage.output_tokens,
        model=model,
        cache_creation_input_tokens=cache_creation,
        cache_read_input_tokens=cache_read,
    )


def provider_supports_tool_use(provider_name: str, model: str) -> bool:
    """Static capability check for the agentic loop.

    Return True only for providers/models that support structured
    tool use end-to-end with a stable response shape. Other providers
    fall back to the single-shot path on the orchestrator side.
    """
    p = (provider_name or "").strip().lower()
    m = (model or "").strip().lower()
    if p == "anthropic":
        # Claude 3.5 Sonnet and later all support tool use. Older
        # models did too but the response shapes were less stable.
        if "claude-3-5" in m or "claude-sonnet-4" in m or "claude-opus-4" in m:
            return True
        # Conservative default for unknown Anthropic model strings —
        # the API supports tools on all current models.
        return True
    # Add other provider support here as adapters land.
    return False


def provider_supports_prompt_caching(provider_name: str) -> bool:
    return (provider_name or "").strip().lower() == "anthropic"
