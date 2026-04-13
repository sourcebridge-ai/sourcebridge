# Troubleshooting

Common issues and how to fix them. If your issue isn't listed here, [open a GitHub issue](https://github.com/sourcebridge/sourcebridge/issues) with the error output.

---

## 1. Docker Not Installed or Not Running

**Symptom:** `demo.sh` fails with "Docker is not installed" or "Docker daemon is not running."

**Fix:**

1. Install Docker Desktop from [docker.com](https://docker.com)
2. Start Docker Desktop
3. Verify it's running:
   ```bash
   docker info
   ```
4. Re-run `./demo.sh`

**macOS note:** If you see a permissions error, make sure Docker Desktop has completed its initial setup and the Docker icon appears in the menu bar.

---

## 2. Ports 3000 or 8080 Already in Use

**Symptom:** `demo.sh` fails with "Port 3000 is already in use" or "Port 8080 is already in use."

**Fix — Option A:** Stop whatever is using the port:

```bash
# Find what's using port 3000
lsof -i :3000

# Kill it (replace PID with the actual process ID)
kill <PID>
```

**Fix — Option B:** Use different ports:

```bash
SOURCEBRIDGE_WEB_PORT=3001 SOURCEBRIDGE_API_PORT=8081 ./demo.sh
```

Then open `http://localhost:3001` instead.

---

## 3. No AI Artifacts (Cliff Notes, Code Tours) Appearing

**Symptom:** The repository is indexed and visible, but cliff notes and other AI-generated content don't appear.

**Cause:** No LLM is available. SourceBridge needs either a local model (Ollama) or a cloud API key to generate AI artifacts.

**Fix — Ollama (local, free):**

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model (qwen3:32b recommended, or use a smaller model)
ollama pull qwen3:32b

# Re-run the demo
./demo.sh
```

**Fix — Cloud API key:**

```bash
# Anthropic (recommended for best quality)
SOURCEBRIDGE_LLM_API_KEY=sk-ant-... ./demo.sh

# OpenAI
SOURCEBRIDGE_LLM_PROVIDER=openai SOURCEBRIDGE_LLM_API_KEY=sk-... ./demo.sh
```

**Check if artifacts are still generating:**
```bash
# View worker logs
docker compose logs -f worker
```

Artifact generation can take 2-10 minutes depending on the repo size and model speed.

---

## 4. Ollama Not Reachable

**Symptom:** Ollama is installed but SourceBridge can't connect to it. Worker logs show connection errors to `host.docker.internal:11434`.

**Fix:**

1. Verify Ollama is running:
   ```bash
   curl http://localhost:11434/api/tags
   ```
   If this fails, start Ollama: `ollama serve`

2. Verify a model is pulled:
   ```bash
   ollama list
   ```
   If empty, pull one: `ollama pull qwen3:32b`

3. **Linux-specific:** `host.docker.internal` may not resolve. Add it to your Docker Compose override:
   ```yaml
   # docker-compose.override.yml
   services:
     worker:
       extra_hosts:
         - "host.docker.internal:host-gateway"
   ```

4. Restart services:
   ```bash
   docker compose down && ./demo.sh
   ```

---

## 5. Services Not Healthy After Startup

**Symptom:** `demo.sh` times out waiting for services, or services keep restarting.

**Diagnose:**

```bash
# Check which services are running
docker compose ps

# Check logs for the failing service
docker compose logs surrealdb
docker compose logs sourcebridge
docker compose logs worker
docker compose logs web
```

**Common causes:**

**SurrealDB won't start:**
- Check disk space: `df -h`
- Remove stale data and retry: `docker compose down -v && ./demo.sh`

**API server won't start:**
- Check for config errors: `docker compose logs sourcebridge | head -50`
- Ensure SurrealDB is healthy first: `curl http://localhost:8000/health`

**Worker crashes on startup:**
- Check for Python dependency issues: `docker compose logs worker | head -50`
- Rebuild: `docker compose build worker && docker compose up -d worker`

**Web UI not loading:**
- The Next.js build takes 15-30 seconds on first start — wait and refresh
- Check logs: `docker compose logs web`

**Out of memory:**
- SourceBridge services need ~2GB RAM total
- Check Docker Desktop memory allocation (Settings > Resources)
- Increase to at least 4GB

---

## General Debugging

### View all logs

```bash
docker compose logs -f
```

### Restart a single service

```bash
docker compose restart sourcebridge
```

### Full reset (removes all data)

```bash
docker compose down -v
./demo.sh
```

### Check API health

```bash
# Basic health
curl http://localhost:8080/healthz

# Detailed readiness (database, worker status)
curl http://localhost:8080/readyz | python3 -m json.tool
```

---

## Still Stuck?

1. Check [GitHub Issues](https://github.com/sourcebridge/sourcebridge/issues) — someone may have hit the same problem
2. [Open a new issue](https://github.com/sourcebridge/sourcebridge/issues/new?template=bug_first_run.md) — include the output of `docker compose logs` and your OS/Docker version
3. [Start a discussion](https://github.com/sourcebridge/sourcebridge/discussions) — for questions or general help
