---
name: First-Run Issue
about: Something went wrong during your first time running SourceBridge
title: "[First Run] "
labels: bug, first-run
assignees: ""
---

## What happened?

Describe what you were trying to do and what went wrong.

## How did you run SourceBridge?

- [ ] `./demo.sh`
- [ ] `docker compose up`
- [ ] `./setup.sh`
- [ ] Homebrew (`brew install`)
- [ ] Helm / Kubernetes
- [ ] Other: ___

## How far did you get?

- [ ] Docker started
- [ ] Services became healthy
- [ ] Web UI loaded at localhost:3000
- [ ] Sample repo appeared in the UI
- [ ] AI artifacts (cliff notes, etc.) generated
- [ ] Didn't get this far — stuck at: ___

## LLM Setup

- [ ] No LLM (zero-config demo)
- [ ] Ollama (local)
- [ ] Anthropic API key
- [ ] OpenAI API key
- [ ] Other: ___

## Error Output

Paste the error or relevant terminal output:

```
Paste here
```

## Docker Logs

If applicable, paste the output of:
```bash
docker compose logs --tail=50
```

```
Paste here
```

## Environment

- OS: [e.g., macOS 15.3, Ubuntu 24.04, Windows 11]
- Docker version: [run `docker --version`]
- Docker Compose version: [run `docker compose version`]
- Available RAM: [e.g., 8GB]
