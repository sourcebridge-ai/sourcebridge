# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Tests for the shared knowledge-artifact parse helpers."""

from __future__ import annotations

from workers.knowledge.parse_utils import (
    coerce_int,
    count_specific_identifiers,
    count_unique_file_paths,
    meets_confidence_floor,
)


def test_coerce_int_handles_null_and_strings():
    assert coerce_int(None) == 0
    assert coerce_int("42") == 42
    assert coerce_int("  7 ") == 7
    assert coerce_int("not-a-number") == 0
    assert coerce_int("not-a-number", default=-1) == -1
    assert coerce_int(3.9) == 3
    assert coerce_int(True) == 0
    assert coerce_int({"x": 1}) == 0


def test_count_unique_file_paths_ignores_blanks():
    paths = [" internal/foo.go ", "internal/foo.go", "", None, "workers/bar.py"]
    assert count_unique_file_paths(p for p in paths if p is not None) == 2


def test_count_specific_identifiers_finds_backticked_names():
    content = "The `FooService` calls `BarController.load` via `_private_helper`."
    # Only the plain identifiers inside single backticks count; qualified
    # names like `BarController.load` don't match the regex.
    assert count_specific_identifiers(content) == 2


def test_meets_confidence_floor_positive_path():
    assert meets_confidence_floor(
        current_confidence="low",
        unique_file_paths={"internal/foo.go", "internal/bar.go", "workers/baz.py"},
        content="Mentions `FooService` and `BarController` explicitly.",
        min_files=3,
        min_identifiers=2,
    ) is True


def test_meets_confidence_floor_already_high_returns_false():
    assert meets_confidence_floor(
        current_confidence="high",
        unique_file_paths={"a.go", "b.go", "c.go"},
        content="`X` and `Y` and `Z`.",
    ) is False


def test_meets_confidence_floor_requires_both_thresholds():
    # enough files but too few identifiers
    assert meets_confidence_floor(
        current_confidence="low",
        unique_file_paths={"a.go", "b.go", "c.go"},
        content="Only `Foo` here.",
    ) is False
    # enough identifiers but too few files
    assert meets_confidence_floor(
        current_confidence="low",
        unique_file_paths={"a.go"},
        content="`Foo` and `Bar` and `Baz`.",
    ) is False
