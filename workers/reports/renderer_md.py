# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Markdown renderer — writes the assembled report to a .md file.

This is the simplest renderer. The engine.py already produces Markdown
via the assembler; this module just handles any final formatting.
"""

from __future__ import annotations


def post_process_markdown(markdown: str) -> str:
    """Apply final formatting passes to the assembled Markdown."""
    # Normalize line endings
    markdown = markdown.replace("\r\n", "\n")

    # Ensure single blank line between sections (no triple+ newlines)
    while "\n\n\n" in markdown:
        markdown = markdown.replace("\n\n\n", "\n\n")

    # Ensure file ends with a single newline
    markdown = markdown.rstrip() + "\n"

    return markdown
