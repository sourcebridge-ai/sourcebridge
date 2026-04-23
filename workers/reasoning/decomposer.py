"""Query decomposition + final-answer synthesis for Phase 4 of the
quality-push plan.

Two small LLM calls bracket the agentic loop:

- `decompose_question` — splits a multi-hop question into 3–4
  narrow sub-questions the loop can answer in parallel. Returns
  [] when the question is already atomic.
- `synthesize_from_sub_answers` — after each sub-loop runs, joins
  the sub-answers into a coherent final answer with the original
  question's framing.

Both functions fail gracefully: the Go orchestrator catches
exceptions and falls through to the single-loop path.
"""

from __future__ import annotations

import json
import logging
import re
from dataclasses import dataclass
from typing import Any

from workers.common.llm.provider import LLMProvider, require_nonempty

log = logging.getLogger(__name__)


DEFAULT_DECOMPOSER_MODEL = "claude-haiku-4-5"
DEFAULT_SYNTHESIZER_MODEL = "claude-sonnet-4-5"


@dataclass
class DecompositionResult:
    """Sub-questions produced by the decomposer."""

    sub_questions: list[str]
    input_tokens: int = 0
    output_tokens: int = 0
    model: str = ""


@dataclass
class SynthesisResult:
    """Final synthesis output."""

    answer: str
    input_tokens: int = 0
    output_tokens: int = 0
    model: str = ""
    cache_creation_input_tokens: int = 0
    cache_read_input_tokens: int = 0


_DECOMPOSE_SYSTEM = """You are a question-decomposition planner for a
codebase QA system.

When a question is multi-hop (spans several files/modules or asks
about a concern that threads through the code), produce a JSON
array of 2–4 minimal sub-questions that a specialized agent can
answer independently. Each sub-question should:

- Be answerable from a narrow slice of code (one module / one
  concern).
- Stand on its own without context from the other sub-questions.
- Use concrete nouns from the original question when possible.

When the question is already narrow enough for a single agent to
answer directly, return an empty array [].

Output STRICT JSON. No prose, no markdown, no code fences. The
output must be a JSON array of strings.

Examples:

Q: "How does authentication work across the stack?"
A: ["Where are user credentials validated?", "How are sessions
issued and stored?", "Where is the session token checked on
protected endpoints?", "How does logout invalidate sessions?"]

Q: "Where is the recycle bin handler?"
A: []

Q: "What happens when a request hits /api/v1/ask?"
A: ["What middleware runs on /api/v1/ask?", "Where is the POST
handler for /api/v1/ask defined?", "What orchestrator function
does that handler invoke?"]"""


_SYNTHESIZE_SYSTEM = """You are synthesizing a final answer from
sub-answers produced by several agents working independently.

Write one coherent answer to the original question, drawing on the
sub-answers. Rules:

- Preserve every `[cite:<handle>]` tag from the sub-answers
  verbatim. They're the citation anchors the reference resolver
  uses downstream.
- Prefer the sub-answers over general knowledge. When sub-answers
  disagree, surface both and note the disagreement.
- Keep it tight — no preamble, no "based on my research" framing.
  Answer the question.
- If a sub-answer returned no useful information, say so briefly
  and move on. Don't fabricate.
"""


_JSON_ARRAY_RE = re.compile(r"\[.*\]", re.DOTALL)


def _parse_sub_questions(raw: str, cap: int = 4) -> list[str]:
    """Parse the decomposer output. Tolerates stray prose by
    extracting the outermost JSON array block."""
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        match = _JSON_ARRAY_RE.search(raw)
        if not match:
            log.warning("decomposer_output_not_json", extra={"raw": raw[:200]})
            return []
        parsed = json.loads(match.group(0))
    if not isinstance(parsed, list):
        return []
    out: list[str] = []
    for item in parsed:
        if not isinstance(item, str):
            continue
        q = item.strip()
        if q:
            out.append(q)
        if len(out) >= cap:
            break
    return out


async def decompose_question(
    provider: LLMProvider,
    *,
    question: str,
    question_class: str = "",
    max_sub_questions: int = 4,
    model: str | None = None,
) -> DecompositionResult:
    """Ask Haiku to split `question` into sub-questions. Returns
    empty list when the question is atomic or when parsing fails —
    caller should run the single-loop path in both cases."""
    if max_sub_questions <= 0:
        max_sub_questions = 4
    prompt_parts = [f"Question: {question}"]
    if question_class:
        prompt_parts.append(f"Class hint: {question_class}")
    prompt_parts.append(
        f"Produce at most {max_sub_questions} sub-questions, or [] if atomic."
    )
    prompt = "\n".join(prompt_parts)

    resp = await provider.complete(
        prompt=prompt,
        system=_DECOMPOSE_SYSTEM,
        max_tokens=512,
        temperature=0.0,
        model=model or DEFAULT_DECOMPOSER_MODEL,
    )
    resp = require_nonempty(resp, context="decompose_question")
    sub_questions = _parse_sub_questions(resp.content, cap=max_sub_questions)
    return DecompositionResult(
        sub_questions=sub_questions,
        input_tokens=resp.input_tokens or 0,
        output_tokens=resp.output_tokens or 0,
        model=resp.model or (model or DEFAULT_DECOMPOSER_MODEL),
    )


def _format_sub_answers(sub_answers: list[dict[str, Any]]) -> str:
    """Render sub-answers as a cite-preserving markdown block for
    the synthesizer prompt."""
    parts: list[str] = []
    for i, sa in enumerate(sub_answers, 1):
        q = sa.get("sub_question", "").strip()
        a = sa.get("sub_answer", "").strip()
        handles = sa.get("reference_handles") or []
        parts.append(f"## Sub-question {i}: {q}\n\n{a}")
        if handles:
            handle_list = ", ".join(f"`{h}`" for h in handles[:10])
            parts.append(f"\n(references: {handle_list})")
    return "\n\n".join(parts)


async def synthesize_from_sub_answers(
    provider: LLMProvider,
    *,
    original_question: str,
    sub_answers: list[dict[str, Any]],
    model: str | None = None,
    enable_prompt_caching: bool = True,
) -> SynthesisResult:
    """Run the final synthesis turn. Returns the joined answer."""
    if not sub_answers:
        return SynthesisResult(answer="")

    body = _format_sub_answers(sub_answers)
    prompt = (
        f"Original question: {original_question}\n\n"
        "Sub-answers gathered by independent agents:\n\n"
        f"{body}\n\n"
        "Synthesize the final answer to the original question now."
    )

    # Use the Anthropic client directly so prompt caching and cache
    # accounting work the same way as the agent loop. Fall back to
    # the provider's `complete` when the client isn't Anthropic.
    anthropic_client = getattr(provider, "client", None)
    model_id = model or getattr(provider, "default_model", "") or DEFAULT_SYNTHESIZER_MODEL

    if anthropic_client is not None and "anthropic" in type(provider).__name__.lower():
        kwargs: dict[str, Any] = {
            "model": model_id,
            "max_tokens": 2048,
            "messages": [{"role": "user", "content": prompt}],
        }
        if enable_prompt_caching:
            kwargs["system"] = [
                {
                    "type": "text",
                    "text": _SYNTHESIZE_SYSTEM,
                    "cache_control": {"type": "ephemeral"},
                }
            ]
        else:
            kwargs["system"] = _SYNTHESIZE_SYSTEM
        resp = await anthropic_client.messages.create(**kwargs)  # type: ignore[call-overload]
        # Anthropic response → unified SynthesisResult.
        text = ""
        for block in resp.content:
            if getattr(block, "type", None) == "text":
                text = (text + block.text) if text else block.text
        cache_creation = getattr(resp.usage, "cache_creation_input_tokens", 0) or 0
        cache_read = getattr(resp.usage, "cache_read_input_tokens", 0) or 0
        return SynthesisResult(
            answer=text,
            input_tokens=resp.usage.input_tokens,
            output_tokens=resp.usage.output_tokens,
            model=model_id,
            cache_creation_input_tokens=cache_creation,
            cache_read_input_tokens=cache_read,
        )

    # Non-Anthropic fallback: use the Protocol's complete(). No
    # cache accounting on this path.
    resp = await provider.complete(
        prompt=prompt,
        system=_SYNTHESIZE_SYSTEM,
        max_tokens=2048,
        temperature=0.0,
        model=model_id,
    )
    resp = require_nonempty(resp, context="synthesize_decomposed")
    return SynthesisResult(
        answer=resp.content,
        input_tokens=resp.input_tokens or 0,
        output_tokens=resp.output_tokens or 0,
        model=resp.model or model_id,
    )
