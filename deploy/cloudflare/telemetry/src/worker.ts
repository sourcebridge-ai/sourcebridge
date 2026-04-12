/**
 * SourceBridge Telemetry Worker
 *
 * Receives anonymous telemetry pings from SourceBridge installations,
 * stores them in Cloudflare D1, and forwards to PostHog for analytics.
 *
 * Endpoints:
 *   POST /v1/ping    — receive a telemetry ping
 *   GET  /v1/stats   — public aggregate stats (JSON)
 *   GET  /v1/badge   — shields.io-compatible badge (active installs)
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

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const path = url.pathname;

    // CORS headers for browser access to /v1/stats
    const corsHeaders = {
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type",
    };

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
      if (path === "/v1/badge" && request.method === "GET") {
        return await handleBadge(env, corsHeaders);
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

  // Store in D1 — upsert by installation_id
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
    JSON.stringify(ping.features || []),
    JSON.stringify(ping.counts || {}),
  ).run();

  // Forward to PostHog (non-blocking)
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

    // Fire and forget — don't block the response
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
  // Active installs: seen in the last 7 days
  const active = await env.DB.prepare(
    "SELECT COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-7 days')"
  ).first<{ count: number }>();

  // Total installs ever
  const total = await env.DB.prepare(
    "SELECT COUNT(*) as count FROM pings"
  ).first<{ count: number }>();

  // Version distribution
  const versions = await env.DB.prepare(
    "SELECT version, COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-7 days') GROUP BY version ORDER BY count DESC LIMIT 10"
  ).all();

  // Edition distribution
  const editions = await env.DB.prepare(
    "SELECT edition, COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-7 days') GROUP BY edition ORDER BY count DESC"
  ).all();

  // Platform distribution
  const platforms = await env.DB.prepare(
    "SELECT platform, COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-7 days') GROUP BY platform ORDER BY count DESC LIMIT 10"
  ).all();

  // Total repos across all active installs
  const repoSum = await env.DB.prepare(
    "SELECT COALESCE(SUM(repos), 0) as total FROM pings WHERE last_seen > datetime('now', '-7 days')"
  ).first<{ total: number }>();

  const stats = {
    active_installs_7d: active?.count || 0,
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
      "Cache-Control": "public, max-age=300", // 5 min cache
    },
  });
}

async function handleBadge(env: Env, headers: Record<string, string>): Promise<Response> {
  // shields.io endpoint format
  const active = await env.DB.prepare(
    "SELECT COUNT(*) as count FROM pings WHERE last_seen > datetime('now', '-7 days')"
  ).first<{ count: number }>();

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
