# QA parity runner

One arm of the QA parity benchmark. Two arms pair into a report.

## Build

```bash
cd benchmarks/qa/cmd/runner
go build -o ../../runner .
```

## Baseline arm (Python subprocess)

Must run from a host with `uv` + the Python worker installed:

```bash
./runner -arm=baseline \
         -questions=../../questions.yaml \
         -mode=deep \
         -workers-dir=../../../workers \
         -out=../../reports/$(date +%Y-%m-%d)_baseline
```

## Candidate arm (server-side orchestrator)

Must run against a server with `SOURCEBRIDGE_QA_SERVER_SIDE_ENABLED=true`:

```bash
./runner -arm=candidate \
         -questions=../../questions.yaml \
         -mode=deep \
         -server-url=https://sourcebridge.example.com \
         -repository-id=rep-abc \
         -api-token=$SOURCEBRIDGE_API_TOKEN \
         -out=../../reports/$(date +%Y-%m-%d)_candidate
```

## Output

Each arm writes a directory containing:

- `run.jsonl` — one JSON record per question with answer,
  references, diagnostics, usage, and elapsed_ms.
- `environment.yaml` — commit SHA, arm, date, mode, server URL,
  optional notes. Used by `report.py` to caption the paired report
  so a reader can tell which build produced which numbers.

## Next step

Pair two arms with `report.py` (follow-up). Never treat a single-arm
run as proof of parity — the decision rule is defined over paired
differences with a reviewed regression table.
