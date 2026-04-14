#!/usr/bin/env python3
"""Run an end-to-end classic vs understanding-first cliff-notes benchmark.

This script assumes a SourceBridge API and worker are already running.
It authenticates, imports the target repository twice, sets different
generation-mode defaults, runs cold classic and understanding-first
cliff-notes generation, and optionally performs a warm understanding-first
rerun after deleting only the final artifact.
"""

from __future__ import annotations

import argparse
import base64
import json
import sys
import time
import urllib.request
from dataclasses import dataclass
from typing import Any


@dataclass
class BenchmarkConfig:
    base_url: str
    password: str
    repo_path: str
    existing_repo_id: str | None
    classic_name: str
    understanding_name: str
    audience: str
    depth: str
    timeout_secs: int
    sql_url: str | None
    surreal_ns: str
    surreal_db: str
    surreal_user: str
    surreal_pass: str


def request_json(
    url: str,
    body: dict[str, Any] | None = None,
    headers: dict[str, str] | None = None,
    method: str | None = None,
) -> dict[str, Any]:
    req = urllib.request.Request(
        url,
        data=None if body is None else json.dumps(body).encode(),
        headers={"Content-Type": "application/json", **(headers or {})},
        method=method,
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.loads(resp.read().decode())


def gql(base_url: str, token: str, query: str, variables: dict[str, Any] | None = None) -> dict[str, Any]:
    payload = request_json(
        f"{base_url.rstrip('/')}/api/v1/graphql",
        {"query": query, "variables": variables or {}},
        {"Authorization": f"Bearer {token}"},
    )
    if "errors" in payload:
        raise RuntimeError(json.dumps(payload["errors"]))
    return payload["data"]


def ensure_auth(base_url: str, password: str) -> str:
    info = request_json(f"{base_url.rstrip('/')}/auth/info", method="GET")
    if not info.get("setup_done"):
        request_json(f"{base_url.rstrip('/')}/auth/setup", {"password": password})
    return request_json(f"{base_url.rstrip('/')}/auth/login", {"password": password})["token"]


def surreal_sql(cfg: BenchmarkConfig, sql: str) -> str:
    if not cfg.sql_url:
        raise RuntimeError("warm rerun requires --sql-url")
    req = urllib.request.Request(cfg.sql_url, data=sql.encode(), method="POST")
    req.add_header("Content-Type", "application/surrealdb")
    req.add_header("Accept", "application/json")
    req.add_header("Surreal-NS", cfg.surreal_ns)
    req.add_header("Surreal-DB", cfg.surreal_db)
    basic = base64.b64encode(f"{cfg.surreal_user}:{cfg.surreal_pass}".encode()).decode()
    req.add_header("Authorization", f"Basic {basic}")
    with urllib.request.urlopen(req, timeout=60) as resp:
        return resp.read().decode()


def wait_for_artifact(base_url: str, token: str, artifact_id: str, timeout_secs: int) -> tuple[dict[str, Any], list[tuple[Any, ...]]]:
    query = """
    query Artifact($id: ID!) {
      knowledgeArtifact(id: $id) {
        id
        status
        progress
        progressPhase
        progressMessage
        errorCode
        errorMessage
        generationMode
        understandingId
        rendererVersion
        sections {
          title
          summary
          content
          evidence {
            sourceType
            filePath
            lineStart
            lineEnd
          }
        }
      }
    }
    """
    deadline = time.time() + timeout_secs
    history: list[tuple[Any, ...]] = []
    while time.time() < deadline:
        artifact = gql(base_url, token, query, {"id": artifact_id})["knowledgeArtifact"]
        if artifact is None:
            history.append(("MISSING", None, None, None))
            time.sleep(2.5)
            continue
        history.append(
            (
                artifact["status"],
                artifact.get("progress"),
                artifact.get("progressPhase"),
                artifact.get("progressMessage"),
            )
        )
        if artifact["status"] in ("READY", "FAILED"):
            return artifact, history
        time.sleep(2.5)
    raise TimeoutError(f"timed out waiting for artifact {artifact_id}")


def add_repo(base_url: str, token: str, name: str, path: str) -> dict[str, Any]:
    query = """
    mutation AddRepository($input: AddRepositoryInput!) {
      addRepository(input: $input) {
        id
        name
        generationModeDefault
      }
    }
    """
    return gql(base_url, token, query, {"input": {"name": name, "path": path}})["addRepository"]


def get_repo(base_url: str, token: str, repo_id: str) -> dict[str, Any]:
    query = """
    query Repo($id: ID!) {
      repository(id: $id) {
        id
        name
        generationModeDefault
      }
    }
    """
    return gql(base_url, token, query, {"id": repo_id})["repository"]


def set_repo_mode(base_url: str, token: str, repo_id: str, mode: str) -> None:
    query = """
    mutation SetMode($input: UpdateRepositoryKnowledgeSettingsInput!) {
      updateRepositoryKnowledgeSettings(input: $input) {
        id
        generationModeDefault
      }
    }
    """
    gql(base_url, token, query, {"input": {"repositoryId": repo_id, "generationModeDefault": mode}})


def generate_cliff_notes(base_url: str, token: str, repo_id: str, audience: str, depth: str, mode: str) -> dict[str, Any]:
    query = """
    mutation Generate($input: GenerateCliffNotesInput!) {
      generateCliffNotes(input: $input) {
        id
        status
        generationMode
        rendererVersion
        repositoryId
      }
    }
    """
    return gql(
        base_url,
        token,
        query,
        {"input": {"repositoryId": repo_id, "audience": audience, "depth": depth, "generationMode": mode}},
    )["generateCliffNotes"]


def summarize_artifact(artifact: dict[str, Any], duration_sec: float, history: list[tuple[Any, ...]]) -> dict[str, Any]:
    sections = artifact.get("sections") or []
    return {
        "artifact_id": artifact["id"],
        "duration_sec": round(duration_sec, 2),
        "status": artifact["status"],
        "generation_mode": artifact.get("generationMode"),
        "understanding_id": artifact.get("understandingId"),
        "renderer_version": artifact.get("rendererVersion"),
        "section_count": len(sections),
        "evidence_count": sum(len(section.get("evidence") or []) for section in sections),
        "section_titles": [section["title"] for section in sections],
        "error_code": artifact.get("errorCode"),
        "error_message": artifact.get("errorMessage"),
        "history_tail": history[-8:],
        "sample_sections": [
            {
                "title": section["title"],
                "summary": (section.get("summary") or "")[:240],
                "content_preview": (section.get("content") or "")[:360],
                "evidence_count": len(section.get("evidence") or []),
            }
            for section in sections[:4]
        ],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="http://127.0.0.1:18084")
    parser.add_argument("--password", required=True)
    parser.add_argument("--repo-path")
    parser.add_argument("--existing-repo-id")
    parser.add_argument("--classic-name", default="bench-classic")
    parser.add_argument("--understanding-name", default="bench-understanding")
    parser.add_argument("--audience", default="DEVELOPER")
    parser.add_argument("--depth", default="MEDIUM")
    parser.add_argument("--timeout-secs", type=int, default=1200)
    parser.add_argument("--sql-url")
    parser.add_argument("--surreal-ns", default="sourcebridge")
    parser.add_argument("--surreal-db", default="")
    parser.add_argument("--surreal-user", default="root")
    parser.add_argument("--surreal-pass", default="CHANGE_ME")
    parser.add_argument("--skip-warm", action="store_true")
    parser.add_argument(
        "--single-mode",
        choices=("CLASSIC", "UNDERSTANDING_FIRST"),
        help="run only one generation mode instead of the full classic/understanding comparison",
    )
    args = parser.parse_args()

    cfg = BenchmarkConfig(
        base_url=args.base_url,
        password=args.password,
        repo_path=args.repo_path or "",
        existing_repo_id=args.existing_repo_id,
        classic_name=args.classic_name,
        understanding_name=args.understanding_name,
        audience=args.audience,
        depth=args.depth,
        timeout_secs=args.timeout_secs,
        sql_url=args.sql_url,
        surreal_ns=args.surreal_ns,
        surreal_db=args.surreal_db,
        surreal_user=args.surreal_user,
        surreal_pass=args.surreal_pass,
    )

    if not cfg.existing_repo_id and not cfg.repo_path:
        raise RuntimeError("provide either --repo-path or --existing-repo-id")

    token = ensure_auth(cfg.base_url, cfg.password)
    if cfg.existing_repo_id:
        existing_repo = get_repo(cfg.base_url, token, cfg.existing_repo_id)
        classic_repo = existing_repo
        understanding_repo = existing_repo
    else:
        classic_repo = add_repo(cfg.base_url, token, cfg.classic_name, cfg.repo_path)
        understanding_repo = add_repo(cfg.base_url, token, cfg.understanding_name, cfg.repo_path)
        set_repo_mode(cfg.base_url, token, classic_repo["id"], "CLASSIC")
        set_repo_mode(cfg.base_url, token, understanding_repo["id"], "UNDERSTANDING_FIRST")

    results: dict[str, Any] = {"classic_repo": classic_repo, "understanding_repo": understanding_repo}

    runs = [
        ("classic_cold", classic_repo["id"], "CLASSIC"),
        ("understanding_cold", understanding_repo["id"], "UNDERSTANDING_FIRST"),
    ]
    if args.single_mode == "CLASSIC":
        runs = [("classic_cold", classic_repo["id"], "CLASSIC")]
    elif args.single_mode == "UNDERSTANDING_FIRST":
        runs = [("understanding_cold", understanding_repo["id"], "UNDERSTANDING_FIRST")]

    for label, repo_id, mode in runs:
        started = time.time()
        artifact = generate_cliff_notes(cfg.base_url, token, repo_id, cfg.audience, cfg.depth, mode)
        final_artifact, history = wait_for_artifact(cfg.base_url, token, artifact["id"], cfg.timeout_secs)
        results[label] = summarize_artifact(final_artifact, time.time() - started, history)

    if not args.skip_warm and args.single_mode in (None, "UNDERSTANDING_FIRST"):
        if not cfg.sql_url or not cfg.surreal_db:
            raise RuntimeError("warm rerun requires --sql-url and --surreal-db")
        cold_id = results["understanding_cold"]["artifact_id"]
        results["warm_delete_sql_result"] = surreal_sql(
            cfg,
            f"DELETE FROM ca_knowledge_artifact WHERE id = type::thing('ca_knowledge_artifact', '{cold_id}');",
        )[:400]
        started = time.time()
        artifact = generate_cliff_notes(
            cfg.base_url,
            token,
            understanding_repo["id"],
            cfg.audience,
            cfg.depth,
            "UNDERSTANDING_FIRST",
        )
        final_artifact, history = wait_for_artifact(cfg.base_url, token, artifact["id"], cfg.timeout_secs)
        results["understanding_warm"] = summarize_artifact(final_artifact, time.time() - started, history)

    print(json.dumps(results, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
