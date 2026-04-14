# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

from __future__ import annotations

ARCHITECTURE_DIAGRAM_SYSTEM = """\
You are a software architect generating a high-level repository architecture diagram.

Return Mermaid flowchart syntax only. No prose, no explanations, no Markdown fences.

Rules:
1. Keep the diagram high-level and readable.
2. Prefer subgraphs for major repository areas.
3. Treat the deterministic architecture scaffold as the structural source of truth.
4. Do not invent module-to-module edges that contradict the scaffold.
5. If an edge is uncertain, omit it rather than guessing.
6. Avoid naming a subgraph and a node with the same identifier.
7. Output a single Mermaid flowchart diagram only.
"""


def build_architecture_diagram_prompt(
    repository_name: str,
    audience: str,
    depth: str,
    architecture_context_json: str,
    deterministic_diagram_json: str,
) -> str:
    return f"""\
Generate an AI architecture diagram for the repository "{repository_name}".

Target audience: {audience}
Depth: {depth}

Use the repository understanding and deterministic scaffold below to create a
human-readable Mermaid diagram that stays structurally grounded.

Deterministic scaffold JSON:
{deterministic_diagram_json}

Bounded architecture context JSON:
{architecture_context_json}
"""


def build_architecture_diagram_retry_prompt(
    repository_name: str,
    audience: str,
    depth: str,
    deterministic_diagram_json: str,
    invalid_output: str,
) -> str:
    return f"""\
The previous attempt to generate an architecture diagram for "{repository_name}" was invalid Mermaid.

Target audience: {audience}
Depth: {depth}

You must return a single valid Mermaid flowchart only.
Requirements:
- Start with `flowchart LR` or `flowchart TD`
- Define explicit nodes or subgraphs
- Preserve the deterministic scaffold structure
- Omit uncertain edges instead of guessing
- No prose, no Markdown fences, no explanation

Deterministic scaffold JSON:
{deterministic_diagram_json}

Invalid previous output:
{invalid_output}
"""
