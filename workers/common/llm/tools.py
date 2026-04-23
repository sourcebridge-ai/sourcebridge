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


async def run_agent_turn_anthropic(
    client: Any,
    model: str,
    messages: list[AgentMessage],
    tools: list[ToolSchema],
    max_tokens: int,
) -> AgentTurnResponse:
    """Run one tool-use turn against Anthropic's messages API.

    Returns the assistant's response as an AgentMessage; either
    `text` or `tool_calls` is populated. Raises on hard API errors.
    """
    api_messages, system_text = _messages_for_anthropic(messages)
    api_tools = [t.to_anthropic() for t in tools]

    import anthropic

    kwargs: dict[str, Any] = {
        "model": model,
        "max_tokens": max_tokens or 2048,
        "messages": cast(list[dict[str, Any]], api_messages),
    }
    if system_text:
        kwargs["system"] = system_text
    if api_tools:
        kwargs["tools"] = api_tools

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

    return AgentTurnResponse(
        capability_supported=True,
        turn=turn,
        input_tokens=resp.usage.input_tokens,
        output_tokens=resp.usage.output_tokens,
        model=model,
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
