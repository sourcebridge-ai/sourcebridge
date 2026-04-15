/**
 * SourceBridge Telemetry Worker
 *
 * Receives anonymous telemetry pings from SourceBridge installations,
 * stores current install state plus daily snapshots in Cloudflare D1,
 * forwards pings to PostHog, and serves a lightweight public dashboard.
 *
 * Endpoints:
 *   POST /v1/ping             receive a telemetry ping
 *   GET  /v1/stats            public aggregate stats (JSON)
 *   GET  /v1/stats/timeseries public daily trends (JSON)
 *   GET  /v1/badge            shields.io-compatible badge
 *   GET  /dashboard           human-friendly dashboard
 */

export interface Env {
  DB: D1Database;
  POSTHOG_API_KEY: string;
  POSTHOG_HOST: string;
}

interface TelemetryPing {
  installation_id: string;
  version: string;
  edition: string;
  platform: string;
  go_version?: string;
  uptime?: string;
  repos: number;
  users: number;
  features?: string[];
  counts?: Record<string, number>;
  timestamp: string;
}

interface CountRow {
  count: number;
}

interface TotalRow {
  total: number;
}

interface BreakdownRow {
  value: string;
  count: number;
}

interface TimeseriesRow {
  snapshot_date: string;
  active_installs: number;
  total_repos: number;
  avg_repos_per_install: number;
}

const corsHeaders = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
  "Access-Control-Allow-Headers": "Content-Type",
};

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const path = url.pathname;

    if (request.method === "OPTIONS") {
      return new Response(null, { headers: corsHeaders });
    }

    try {
      if (path === "/v1/ping" && request.method === "POST") {
        return await handlePing(request, env, corsHeaders);
      }
      if (path === "/v1/stats" && request.method === "GET") {
        return await handleStats(env, corsHeaders);
      }
      if (path === "/v1/stats/timeseries" && request.method === "GET") {
        return await handleTimeseries(url, env, corsHeaders);
      }
      if (path === "/v1/badge" && request.method === "GET") {
        return await handleBadge(env, corsHeaders);
      }
      if (path === "/dashboard" && request.method === "GET") {
        return await handleDashboard(headersWithNoCors());
      }
      if (path === "/" || path === "/health") {
        return new Response(JSON.stringify({ status: "ok", service: "sourcebridge-telemetry" }), {
          headers: { ...corsHeaders, "Content-Type": "application/json" },
        });
      }
      return new Response("Not found", { status: 404, headers: corsHeaders });
    } catch (e) {
      console.error("Worker error:", e);
      return new Response("Internal error", { status: 500, headers: corsHeaders });
    }
  },
};

async function handlePing(request: Request, env: Env, headers: Record<string, string>): Promise<Response> {
  let ping: TelemetryPing;
  try {
    ping = await request.json();
  } catch {
    return new Response(JSON.stringify({ error: "invalid JSON" }), {
      status: 400,
      headers: { ...headers, "Content-Type": "application/json" },
    });
  }

  if (!ping.installation_id) {
    return new Response(JSON.stringify({ error: "installation_id required" }), {
      status: 400,
      headers: { ...headers, "Content-Type": "application/json" },
    });
  }

  const snapshotDate = normalizeSnapshotDate(ping.timestamp);
  const features = JSON.stringify(ping.features || []);
  const counts = JSON.stringify(ping.counts || {});

  await env.DB.prepare(`
    INSERT INTO pings (installation_id, version, edition, platform, repos, users, features, counts, last_seen, first_seen)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, datetime('now'), datetime('now'))
    ON CONFLICT(installation_id) DO UPDATE SET
      version = ?2,
      edition = ?3,
      platform = ?4,
      repos = ?5,
      users = ?6,
      features = ?7,
      counts = ?8,
      last_seen = datetime('now'),
      ping_count = ping_count + 1
  `).bind(
    ping.installation_id,
    ping.version || "unknown",
    ping.edition || "oss",
    ping.platform || "unknown",
    ping.repos || 0,
    ping.users || 0,
    features,
    counts,
  ).run();

  await env.DB.prepare(`
    INSERT INTO daily_install_snapshots (
      installation_id, snapshot_date, version, edition, platform, repos, users, features, counts
    )
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)
    ON CONFLICT(installation_id, snapshot_date) DO UPDATE SET
      version = excluded.version,
      edition = excluded.edition,
      platform = excluded.platform,
      repos = excluded.repos,
      users = excluded.users,
      features = excluded.features,
      counts = excluded.counts
  `).bind(
    ping.installation_id,
    snapshotDate,
    ping.version || "unknown",
    ping.edition || "oss",
    ping.platform || "unknown",
    ping.repos || 0,
    ping.users || 0,
    features,
    counts,
  ).run();

  if (env.POSTHOG_API_KEY) {
    const posthogEvent = {
      api_key: env.POSTHOG_API_KEY,
      event: "telemetry_ping",
      distinct_id: ping.installation_id,
      properties: {
        version: ping.version,
        edition: ping.edition,
        platform: ping.platform,
        repos: ping.repos,
        users: ping.users,
        features: ping.features,
        ...ping.counts,
      },
      timestamp: ping.timestamp || new Date().toISOString(),
    };

    const posthogUrl = (env.POSTHOG_HOST || "https://us.i.posthog.com") + "/capture/";
    request.ctx?.waitUntil?.(
      fetch(posthogUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(posthogEvent),
      }).catch(() => {})
    );
  }

  return new Response(JSON.stringify({ status: "ok" }), {
    headers: { ...headers, "Content-Type": "application/json" },
  });
}

async function handleStats(env: Env, headers: Record<string, string>): Promise<Response> {
  const active7d = await env.DB.prepare(
    "SELECT COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-7 days')"
  ).first<CountRow>();

  const active30d = await env.DB.prepare(
    "SELECT COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-30 days')"
  ).first<CountRow>();

  const total = await env.DB.prepare(
    "SELECT COUNT(*) as count FROM pings"
  ).first<CountRow>();

  const versions = await env.DB.prepare(
    "SELECT version as value, COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-7 days') GROUP BY version ORDER BY count DESC LIMIT 10"
  ).all<BreakdownRow>();

  const editions = await env.DB.prepare(
    "SELECT edition as value, COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-7 days') GROUP BY edition ORDER BY count DESC"
  ).all<BreakdownRow>();

  const platforms = await env.DB.prepare(
    "SELECT platform as value, COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-7 days') GROUP BY platform ORDER BY count DESC LIMIT 10"
  ).all<BreakdownRow>();

  const repoSum = await env.DB.prepare(
    "SELECT COALESCE(SUM(repos), 0) as total FROM pings WHERE last_seen > datetime('now', '-7 days')"
  ).first<TotalRow>();

  const stats = {
    active_installs_7d: active7d?.count || 0,
    active_installs_30d: active30d?.count || 0,
    total_installs: total?.count || 0,
    total_repos: repoSum?.total || 0,
    versions: versions.results || [],
    editions: editions.results || [],
    platforms: platforms.results || [],
    generated_at: new Date().toISOString(),
  };

  return new Response(JSON.stringify(stats, null, 2), {
    headers: {
      ...headers,
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=300",
    },
  });
}

async function handleTimeseries(url: URL, env: Env, headers: Record<string, string>): Promise<Response> {
  const windowDays = parseWindowDays(url.searchParams.get("window"));

  const series = await env.DB.prepare(`
    SELECT
      snapshot_date,
      COUNT(*) as active_installs,
      COALESCE(SUM(repos), 0) as total_repos,
      ROUND(COALESCE(AVG(repos), 0), 2) as avg_repos_per_install
    FROM daily_install_snapshots
    WHERE snapshot_date >= date('now', ?1)
    GROUP BY snapshot_date
    ORDER BY snapshot_date ASC
  `).bind(`-${windowDays} days`).all<TimeseriesRow>();

  const response = {
    window_days: windowDays,
    points: series.results || [],
    generated_at: new Date().toISOString(),
  };

  return new Response(JSON.stringify(response, null, 2), {
    headers: {
      ...headers,
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=300",
    },
  });
}

async function handleBadge(env: Env, headers: Record<string, string>): Promise<Response> {
  const active = await env.DB.prepare(
    "SELECT COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-7 days')"
  ).first<CountRow>();

  const count = active?.count || 0;
  const badge = {
    schemaVersion: 1,
    label: "active installs",
    message: count.toString(),
    color: count > 0 ? "brightgreen" : "lightgrey",
  };

  return new Response(JSON.stringify(badge), {
    headers: {
      ...headers,
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=300",
    },
  });
}

async function handleDashboard(headers: Record<string, string>): Promise<Response> {
  return new Response(renderDashboardHTML(), {
    headers: {
      ...headers,
      "Content-Type": "text/html; charset=utf-8",
      "Cache-Control": "public, max-age=300",
    },
  });
}

function normalizeSnapshotDate(timestamp?: string): string {
  if (!timestamp) {
    return new Date().toISOString().slice(0, 10);
  }
  const parsed = new Date(timestamp);
  if (Number.isNaN(parsed.getTime())) {
    return new Date().toISOString().slice(0, 10);
  }
  return parsed.toISOString().slice(0, 10);
}

function parseWindowDays(window: string | null): number {
  if (!window) {
    return 30;
  }
  const trimmed = window.trim().toLowerCase();
  const match = /^(\d+)\s*d$/.exec(trimmed);
  if (!match) {
    return 30;
  }
  const days = Number.parseInt(match[1], 10);
  if (!Number.isFinite(days)) {
    return 30;
  }
  return Math.max(7, Math.min(days, 365));
}

function headersWithNoCors(): Record<string, string> {
  return {};
}

function renderDashboardHTML(): string {
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>SourceBridge Telemetry</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.3/dist/chart.umd.min.js"></script>
    <style>
      :root {
        --bg: #09111f;
        --panel: #0f1a2b;
        --panel-2: #132238;
        --border: rgba(148, 163, 184, 0.18);
        --text: #e5eefb;
        --muted: #94a3b8;
        --accent: #6ee7b7;
        --accent-2: #60a5fa;
        --accent-3: #f59e0b;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        color: var(--text);
        background:
          radial-gradient(circle at top left, rgba(96, 165, 250, 0.15), transparent 35%),
          radial-gradient(circle at top right, rgba(110, 231, 183, 0.12), transparent 30%),
          linear-gradient(180deg, #08101d, #0b1220 55%, #0a1020);
      }
      .wrap {
        width: min(1200px, calc(100vw - 32px));
        margin: 0 auto;
        padding: 32px 0 48px;
      }
      .hero {
        display: flex;
        justify-content: space-between;
        gap: 24px;
        align-items: end;
        margin-bottom: 24px;
      }
      .hero h1 {
        margin: 0;
        font-size: clamp(28px, 4vw, 44px);
        line-height: 1;
      }
      .hero p {
        margin: 12px 0 0;
        color: var(--muted);
        max-width: 720px;
      }
      .meta {
        color: var(--muted);
        font-size: 13px;
        white-space: nowrap;
      }
      .grid {
        display: grid;
        grid-template-columns: repeat(12, minmax(0, 1fr));
        gap: 16px;
      }
      .card {
        background: linear-gradient(180deg, rgba(19, 34, 56, 0.92), rgba(15, 26, 43, 0.92));
        border: 1px solid var(--border);
        border-radius: 18px;
        padding: 18px;
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.18);
      }
      .kpi { grid-column: span 3; min-height: 118px; }
      .chart-wide { grid-column: span 8; }
      .chart-narrow { grid-column: span 4; }
      .chart-half { grid-column: span 6; }
      .label {
        color: var(--muted);
        font-size: 13px;
        text-transform: uppercase;
        letter-spacing: 0.08em;
      }
      .value {
        margin-top: 10px;
        font-size: clamp(28px, 4vw, 42px);
        font-weight: 700;
      }
      .subtle {
        margin-top: 8px;
        color: var(--muted);
        font-size: 13px;
      }
      .section-title {
        margin: 0 0 14px;
        font-size: 16px;
      }
      .toolbar {
        display: flex;
        gap: 8px;
        align-items: center;
        margin-bottom: 12px;
      }
      .pill {
        border: 1px solid var(--border);
        background: rgba(15, 26, 43, 0.9);
        color: var(--text);
        border-radius: 999px;
        padding: 8px 12px;
        font-size: 13px;
        cursor: pointer;
      }
      .pill.active {
        background: var(--accent-2);
        color: #08101d;
        border-color: transparent;
      }
      .empty {
        color: var(--muted);
        font-size: 14px;
        padding: 24px 0;
      }
      canvas {
        width: 100% !important;
        height: 320px !important;
      }
      @media (max-width: 960px) {
        .kpi, .chart-wide, .chart-narrow, .chart-half { grid-column: span 12; }
        .hero { flex-direction: column; align-items: start; }
        .meta { white-space: normal; }
      }
    </style>
  </head>
  <body>
    <div class="wrap">
      <div class="hero">
        <div>
          <h1>SourceBridge Telemetry</h1>
          <p>Anonymous aggregate install telemetry from the existing Cloudflare ingestion path. No repository names, installation IDs, or source code are exposed here.</p>
        </div>
        <div class="meta" id="generatedAt">Loading…</div>
      </div>

      <div class="grid">
        <div class="card kpi">
          <div class="label">Active Installs 7d</div>
          <div class="value" id="kpi-active-7d">—</div>
          <div class="subtle">Seen within the last 7 days</div>
        </div>
        <div class="card kpi">
          <div class="label">Active Installs 30d</div>
          <div class="value" id="kpi-active-30d">—</div>
          <div class="subtle">Seen within the last 30 days</div>
        </div>
        <div class="card kpi">
          <div class="label">Total Installs</div>
          <div class="value" id="kpi-total-installs">—</div>
          <div class="subtle">Unique installations ever observed</div>
        </div>
        <div class="card kpi">
          <div class="label">Total Repos</div>
          <div class="value" id="kpi-total-repos">—</div>
          <div class="subtle">Across active installs in the last 7 days</div>
        </div>

        <div class="card chart-wide">
          <div class="toolbar">
            <h2 class="section-title" style="margin:0; flex:1;">Install Trend</h2>
            <button class="pill active" data-window="30d">30d</button>
            <button class="pill" data-window="90d">90d</button>
            <button class="pill" data-window="180d">180d</button>
          </div>
          <canvas id="installsChart"></canvas>
        </div>

        <div class="card chart-narrow">
          <h2 class="section-title">Repo Trend</h2>
          <canvas id="reposChart"></canvas>
        </div>

        <div class="card chart-half">
          <h2 class="section-title">Version Distribution</h2>
          <canvas id="versionsChart"></canvas>
        </div>

        <div class="card chart-half">
          <h2 class="section-title">Edition Distribution</h2>
          <canvas id="editionsChart"></canvas>
        </div>

        <div class="card chart-half">
          <h2 class="section-title">Platform Distribution</h2>
          <canvas id="platformsChart"></canvas>
        </div>
      </div>
    </div>

    <script>
      const state = {
        window: "30d",
        charts: {},
      };

      function setText(id, value) {
        const el = document.getElementById(id);
        if (el) el.textContent = value;
      }

      function formatNumber(value) {
        return new Intl.NumberFormat().format(value || 0);
      }

      function destroyChart(key) {
        if (state.charts[key]) {
          state.charts[key].destroy();
          delete state.charts[key];
        }
      }

      function renderLineChart(key, canvasId, labels, data, label, color) {
        destroyChart(key);
        state.charts[key] = new Chart(document.getElementById(canvasId), {
          type: "line",
          data: {
            labels,
            datasets: [{
              label,
              data,
              borderColor: color,
              backgroundColor: color + "33",
              fill: true,
              tension: 0.3,
              pointRadius: 2,
            }]
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
              x: { ticks: { color: "#94a3b8" }, grid: { color: "rgba(148,163,184,0.08)" } },
              y: { ticks: { color: "#94a3b8" }, grid: { color: "rgba(148,163,184,0.08)" } },
            },
          }
        });
      }

      function renderBarChart(key, canvasId, rows, label, color) {
        destroyChart(key);
        state.charts[key] = new Chart(document.getElementById(canvasId), {
          type: "bar",
          data: {
            labels: rows.map((row) => row.value),
            datasets: [{
              label,
              data: rows.map((row) => row.count),
              backgroundColor: color,
              borderRadius: 8,
            }]
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: { legend: { display: false } },
            scales: {
              x: { ticks: { color: "#94a3b8" }, grid: { display: false } },
              y: { ticks: { color: "#94a3b8" }, grid: { color: "rgba(148,163,184,0.08)" } },
            },
          }
        });
      }

      async function loadSummary() {
        const res = await fetch("/v1/stats");
        const stats = await res.json();
        setText("kpi-active-7d", formatNumber(stats.active_installs_7d));
        setText("kpi-active-30d", formatNumber(stats.active_installs_30d));
        setText("kpi-total-installs", formatNumber(stats.total_installs));
        setText("kpi-total-repos", formatNumber(stats.total_repos));
        setText("generatedAt", "Updated " + new Date(stats.generated_at).toLocaleString());
        renderBarChart("versions", "versionsChart", stats.versions || [], "Installs", "#60a5fa");
        renderBarChart("editions", "editionsChart", stats.editions || [], "Installs", "#6ee7b7");
        renderBarChart("platforms", "platformsChart", stats.platforms || [], "Installs", "#f59e0b");
      }

      async function loadTimeseries() {
        const res = await fetch("/v1/stats/timeseries?window=" + encodeURIComponent(state.window));
        const stats = await res.json();
        const points = stats.points || [];
        const labels = points.map((point) => point.snapshot_date);
        renderLineChart("installs", "installsChart", labels, points.map((point) => point.active_installs), "Active installs", "#6ee7b7");
        renderLineChart("repos", "reposChart", labels, points.map((point) => point.total_repos), "Total repos", "#60a5fa");
      }

      async function loadDashboard() {
        await Promise.all([loadSummary(), loadTimeseries()]);
      }

      document.querySelectorAll("[data-window]").forEach((button) => {
        button.addEventListener("click", async () => {
          state.window = button.dataset.window;
          document.querySelectorAll("[data-window]").forEach((b) => b.classList.remove("active"));
          button.classList.add("active");
          await loadTimeseries();
        });
      });

      loadDashboard().catch((error) => {
        console.error(error);
        setText("generatedAt", "Failed to load telemetry dashboard");
      });
    </script>
  </body>
</html>`;
}
