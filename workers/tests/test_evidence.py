# SPDX-License-Identifier: AGPL-3.0-or-later

from workers.knowledge.evidence import (
    extract_section_evidence_refs,
    is_valid_evidence_path,
)
from workers.knowledge.types import EvidenceRef


def test_is_valid_evidence_path_rejects_placeholder_values() -> None:
    assert not is_valid_evidence_path("")
    assert not is_valid_evidence_path("None")
    assert not is_valid_evidence_path("repository")
    assert not is_valid_evidence_path("src/")


def test_is_valid_evidence_path_accepts_real_files() -> None:
    assert is_valid_evidence_path("README.md")
    assert is_valid_evidence_path("src/services/auth-service.ts")


def test_extract_section_evidence_refs_filters_invalid_paths() -> None:
    refs = extract_section_evidence_refs(
        [
            EvidenceRef(source_type="file", file_path="repository", line_start=1, line_end=2),
            EvidenceRef(source_type="file", file_path="src/services/auth-service.ts", line_start=10, line_end=20),
        ]
    )
    assert len(refs) == 1
    assert refs[0].path == "src/services/auth-service.ts"
