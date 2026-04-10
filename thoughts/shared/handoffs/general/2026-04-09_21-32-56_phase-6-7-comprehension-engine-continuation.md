---
date: 2026-04-10T01:32:56+0000
researcher: Claude Opus 4.6
git_commit: 279d7e3506b3724206c90b400b0051593f10d825
branch: main
repository: sourcebridge
topic: "Comprehension Engine Phases 6-7 Implementation"
tags: [implementation, comprehension-engine, settings-ui, model-capabilities, incremental-indexing, prompt-caching]
status: complete
last_updated: 2026-04-09
last_updated_by: Claude Opus 4.6
type: implementation_strategy
---

# Handoff: Phase 6-7 Comprehension Engine Continuation

## Task(s)

### Completed (Phases 1-5, all deployed to thor)

| Phase | Commit | Description |
|---|---|---|
| Phase 1 | `892a57b` | Budget guard, dedupe window, scoped timeouts, workflow story fix |
| Phase 2a | `a221603` | LLM job orchestrator foundation (Job model, MemStore, SurrealDB store, bounded queue, dedupe, retry, metrics) |
| Phase 2b | `f4480d6` | Route knowledge mutations through orchestrator |
| Phase 2c | `04a6d59` | Monitor REST endpoints + SSE stream |
| Phase 2d | `c98aa9b` | Monitor page frontend + repo-scoped AI jobs popover |
| Phase 2e | `28e8d8f` | Route synchronous LLM mutations through orchestrator |
| Phase 3 | `adf11d6` | Comprehension engine: CorpusSource, SummaryTree, HierarchicalStrategy, CodeCorpus adapter, CliffNotesRenderer |
| Phase 4a-c | `f9d83e4` | SingleShotStrategy, LongContextDirectStrategy, ModelCapabilityRegistry (13 builtin models), StrategySelector with preference chains |
| Phase 4d-f | `f541e2f` | RequirementsCorpus + DocumentCorpus adapters, plan status addendum |
| Phase 5 | `279d7e3` | Progress phase labels on artifacts, 2s polling, phase/message display in UI |

**All deployed to thor** (`192.168.10.222:30500` registry, namespace `sourcebridge`). All pods running healthy as of 2026-04-09 21:30 UTC.

### Remaining (Phases 6-7)

**Phase 6 — Settings UI + Model Capability Management** (LARGEST remaining phase)
- `ca_strategy_settings` table + Go migration
- `internal/settings/comprehension/` Go package (Get/Set/Effective with inheritance)
- GraphQL mutations: `updateComprehensionSettings`, `resetComprehensionSettings`, `probeModel`, `updateModelCapabilities`
- Python model probe routine (`workers/common/llm/probe.py`)
- Frontend: `/admin/settings/comprehension/page.tsx` with Simple mode (3 cards: model picker + recommended setup, strategy-per-artifact-type matrix, live monitor snapshot) and Advanced mode (overrides, orchestration, models subpage, unsafe overrides)
- Frontend: Models subpage (`/admin/settings/comprehension/models/page.tsx`)
- Live capability badges, undo on every save, first-run onboarding overlay
- See plan Phase 6 section and the "UI/UX principles" section for the full requirements

**Phase 7 — Incremental Indexing + Prompt Caching** (MEDIUM)
- `ca_summary_node` SurrealDB table for persisting the hierarchical summary tree
- Merkle-tree fingerprinting in `workers/comprehension/tree.py` so only changed subtrees are re-summarized on new commits
- Anthropic prompt caching (`cache_control` header) in `workers/common/llm/anthropic.py`
- Gemini implicit cache instrumentation
- Ollama `keep_alive` tuning documentation
- "Rebuild index" admin button + `/api/v1/admin/llm/corpus/:id/invalidate` endpoint

## Critical References

1. **Master plan**: `thoughts/shared/plans/2026-04-09-comprehension-engine-and-llm-orchestration.md` — contains the full architecture, UI/UX principles (12 rules), phase details, success criteria, data model, and the implementation status addendum at the bottom showing what's shipped.
2. **Superseded plan (still has valid root-cause analysis)**: `thoughts/shared/plans/2026-04-09-cliff-notes-generation-ux-reliability.md`
3. **Deploy reference**: `.claude/projects/-Users-jaystuart-dev-sourcebridge/memory/reference_deploy_flow.md` — documents the local registry push + rollout restart flow for thor.

## Recent changes

- `internal/api/graphql/schema.graphqls:665-666` — added `progressPhase` and `progressMessage` to KnowledgeArtifact type
- `internal/api/graphql/models_gen.go:369-370` — added ProgressPhase/ProgressMessage fields to generated model
- `internal/knowledge/models.go:237-238` — added ProgressPhase/ProgressMessage to Artifact struct
- `internal/knowledge/store.go:22` — new `UpdateKnowledgeArtifactProgressWithPhase` interface method
- `internal/knowledge/memstore.go:173-193` — in-memory implementation
- `internal/db/knowledge_store.go:37-38,63-64,382-403` — SurrealDB DTO + implementation
- `internal/db/migrations/016_artifact_progress_phase.surql` — schema migration
- `internal/api/graphql/schema.resolvers.go` — all 8 progress calls upgraded to WithPhase variant
- `web/src/app/(app)/repositories/[id]/page.tsx:343,992` — poll interval 5000→2000
- `web/src/app/(app)/repositories/[id]/page.tsx:1877-1886` — progress bar now shows phase + percentage
- `web/src/lib/graphql/queries.ts:586-587` — query fetches progressPhase + progressMessage

## Learnings

1. **Thor deploy flow**: Images push to `192.168.10.222:30500` (NodePort registry). API cross-compilation (CGO + tree-sitter ARM→AMD64) takes 5-10 minutes; worker builds in ~30s; web in ~2-3 min. Rollout restart is `kubectl -n sourcebridge rollout restart deploy/sourcebridge-api deploy/sourcebridge-worker deploy/sourcebridge-web`.

2. **Dedupe has two layers**: Phase 1's staleness-window check on `ca_knowledge_artifact` (60s window in `schema.resolvers.go:isInFlightGeneration`) AND the orchestrator's in-process + DB-level dedupe on `ca_llm_job`. They're compatible but both must be understood to avoid confusion.

3. **Strategy preference chain**: `SOURCEBRIDGE_CLIFF_NOTES_STRATEGY` env var is now a comma-separated list (default `hierarchical,single_shot`). The selector walks the chain with capability gating + runtime fallback (SnapshotTooLargeError from long_context triggers fallthrough to hierarchical).

4. **gqlgen code generation**: The `models_gen.go` file is auto-generated from `schema.graphqls`. I manually added the `ProgressPhase`/`ProgressMessage` fields rather than running `go generate` because the generation pipeline wasn't set up in my env. Future changes to the schema should run `make proto` or `go generate ./...` to regenerate.

5. **2 pre-existing test failures** in the worker: `test_batch_link_unimplemented` and `test_answer_question_prefers_context_code` — both pre-date this work (verified via stash-compare). Not related to any comprehension engine changes.

6. **Python comprehension package location**: `workers/comprehension/` with sub-packages `adapters/`, `prompts/`. Strategies live at the top level (`hierarchical.py`, `single_shot.py`, `long_context.py`). The selector is `selector.py`, capabilities are `capabilities.py`. All use the `CorpusSource` protocol from `corpus.py`.

7. **RAPTOR and GraphRAG deferred**: RAPTOR needs scikit-learn (~50MB dep), GraphRAG needs networkx + python-leidenalg (C compilation issues on ARM). Both are behind the StrategySelector preference chain so enabling them later is additive.

## Artifacts

**Plans:**
- `thoughts/shared/plans/2026-04-09-comprehension-engine-and-llm-orchestration.md` — master plan with status addendum
- `thoughts/shared/plans/2026-04-09-cliff-notes-generation-ux-reliability.md` — superseded, root-cause analysis still valid

**Go packages:**
- `internal/llm/job.go` — Job, JobStatus, Subsystem, EnqueueRequest, Runtime, JobEvent types
- `internal/llm/store.go` — JobStore interface
- `internal/llm/memstore.go` — in-memory JobStore + tests
- `internal/llm/orchestrator/orchestrator.go` — Enqueue, EnqueueSync, GetJob, ListActive, ListRecent, Metrics, Subscribe, Shutdown
- `internal/llm/orchestrator/runtime.go` — debounced progress, ClassifyError
- `internal/llm/orchestrator/retry.go` — RetryPolicy, IsRetryable
- `internal/llm/orchestrator/dedupe.go` — in-process registry
- `internal/llm/orchestrator/metrics.go` — p50/p95 ring buffer
- `internal/db/llm_job_store.go` — SurrealDB JobStore implementation
- `internal/db/migrations/015_llm_jobs.surql` — ca_llm_job table
- `internal/db/migrations/016_artifact_progress_phase.surql` — progress_phase/message fields
- `internal/api/graphql/knowledge_job.go` — knowledgeJobTargetKey + enqueueKnowledgeJob helper
- `internal/api/graphql/llm_sync.go` — runSyncLLMJob + noopRuntime for synchronous mutations
- `internal/api/rest/admin_llm_monitor.go` — Monitor REST handlers + SSE

**Python packages:**
- `workers/comprehension/corpus.py` — CorpusSource, CorpusUnit, UnitKind, walk helpers
- `workers/comprehension/tree.py` — SummaryNode, SummaryTree
- `workers/comprehension/strategy.py` — ComprehensionStrategy, CapabilityRequirements
- `workers/comprehension/hierarchical.py` — HierarchicalStrategy (4-level tree, parallel leaves)
- `workers/comprehension/single_shot.py` — SingleShotStrategy (legacy wrapper)
- `workers/comprehension/long_context.py` — LongContextDirectStrategy (budget-gated)
- `workers/comprehension/capabilities.py` — ModelCapabilityRegistry (13 builtin models)
- `workers/comprehension/selector.py` — StrategySelector with SelectionTrace
- `workers/comprehension/renderers.py` — CliffNotesRenderer
- `workers/comprehension/adapters/code.py` — CodeCorpus
- `workers/comprehension/adapters/requirements.py` — RequirementsCorpus
- `workers/comprehension/adapters/document.py` — DocumentCorpus
- `workers/comprehension/prompts/hierarchical.py` — leaf/file/package/root prompt templates

**Frontend:**
- `web/src/app/(app)/admin/llm/page.tsx` — Monitor page (health banner, now-running cards, recent table, detail drawer)
- `web/src/components/llm/repo-jobs-popover.tsx` — repo-scoped AI jobs popover

**Tests (total new: ~105 across all phases):**
- `workers/tests/test_comprehension_protocols.py` (10)
- `workers/tests/test_hierarchical_strategy.py` (7)
- `workers/tests/test_code_corpus.py` (9)
- `workers/tests/test_cliff_notes_renderer.py` (5)
- `workers/tests/test_hierarchical_servicer.py` (9)
- `workers/tests/test_single_shot_strategy.py` (7)
- `workers/tests/test_selector.py` (9)
- `workers/tests/test_corpus_adapters.py` (14)
- `internal/llm/memstore_test.go` (9)
- `internal/llm/orchestrator/orchestrator_test.go` (12)
- `internal/api/rest/admin_llm_monitor_test.go` (14)

## Action Items & Next Steps

### Phase 7 (recommended next — medium scope, high production impact)

1. **Create `ca_summary_node` SurrealDB migration** (`internal/db/migrations/017_summary_nodes.surql`) matching the SummaryNode shape in `workers/comprehension/tree.py`.
2. **Add persistence to `SummaryTree`** — write nodes to SurrealDB after `build_tree`, check revision fingerprints before re-building on subsequent calls. The Go-side API server owns the DB; the Python worker would need either a gRPC method to persist nodes or direct SurrealDB access (new pattern — needs design decision).
3. **Merkle fingerprinting**: add a `content_hash` field to CorpusUnit that changes when the leaf content changes. SummaryNode inherits it; on rebuild, only re-summarize nodes whose descendants have new hashes.
4. **Anthropic prompt caching**: in `workers/common/llm/anthropic.py`, add `cache_control: {"type": "ephemeral"}` to the system message for stable prefixes (repo skeleton). Saves ~85% on input tokens across multi-artifact runs.
5. **"Rebuild index" button**: new REST endpoint `POST /api/v1/admin/llm/corpus/:id/invalidate` that deletes cached summary nodes and forces a full rebuild. Surface as a button on the admin settings page.

### Phase 6 (largest remaining — settings UI)

1. Read the plan's Phase 6 section and the UI/UX principles carefully — the "Simple mode = 3 cards" layout is critical. The plan has very specific success criteria including "under 60 seconds for a new operator to configure."
2. Start with the Go migration for `ca_strategy_settings` + REST CRUD endpoints.
3. Then build the frontend settings page iteratively: Simple mode first (model picker + recommended setup card), then Advanced mode (strategy overrides, orchestration knobs).
4. Model probe routine is a Python worker addition — needs a new gRPC method or REST call from Go → Python.

### Deployment

After each phase, build and deploy:
```bash
cd /Users/jaystuart/dev/sourcebridge
docker build -f deploy/docker/Dockerfile.sourcebridge --platform linux/amd64 -t 192.168.10.222:30500/sourcebridge-api:latest .
docker build -f deploy/docker/Dockerfile.worker --platform linux/amd64 -t 192.168.10.222:30500/sourcebridge-worker:latest .
docker build -f deploy/docker/Dockerfile.web --platform linux/amd64 -t 192.168.10.222:30500/sourcebridge-web:latest .
docker push 192.168.10.222:30500/sourcebridge-api:latest
docker push 192.168.10.222:30500/sourcebridge-worker:latest
docker push 192.168.10.222:30500/sourcebridge-web:latest
kubectl -n sourcebridge rollout restart deploy/sourcebridge-api deploy/sourcebridge-worker deploy/sourcebridge-web
```

### Activate hierarchical strategy on thor

The hierarchical path is now the default (preference chain: `hierarchical,single_shot`). To verify it's working, generate cliff notes for the MACU Helpdesk repo and check the worker logs for `cliff_notes_hierarchical_started` / `cliff_notes_hierarchical_tree_built` / `cliff_notes_hierarchical_completed` events. If issues arise, override to single_shot only:
```bash
kubectl -n sourcebridge set env deploy/sourcebridge-worker SOURCEBRIDGE_CLIFF_NOTES_STRATEGY=single_shot
```

## Other Notes

- The plan file uses Phase A-F labels while commit messages use Phase 1-7 numbering. The status addendum at the bottom of the plan reconciles both.
- The `graph.Store` (in-memory) and `db.SurrealStore` both implement `knowledge.KnowledgeStore`. In external mode (production), SurrealStore is used; in embedded mode (dev/test), MemStore.
- The `config.Comprehension.MaxConcurrency` setting is wired through `config.toml` → `rest.Server` → `orchestrator.Config`. Default is 3 concurrent LLM jobs.
- The Monitor page frontend polls every 2 seconds when visible, 10 seconds when backgrounded. SSE is wired on the backend (`/api/v1/admin/llm/stream`) and ready for the frontend to cut over from polling when desired.
- Button variants on this codebase's `<Button>` component are: `primary`, `secondary`, `ghost` — NOT `outline`. Previous code tried `outline` and got TS errors.
