"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { authFetch } from "@/lib/auth-fetch";
import { cn } from "@/lib/utils";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ClusterRow {
  id: string;
  label: string;
  member_count: number;
  representative_symbols: string[];
  cross_cluster_calls?: Record<string, number>;
  partial: boolean;
}

interface ClustersPayload {
  repo_id: string;
  status: "ready" | "partial" | "pending" | "unavailable" | string;
  clusters: ClusterRow[] | null;
  retrieved_at?: string;
}

type SortDir = "ascending" | "descending";

interface Props {
  repoId: string;
  /** Increment to trigger a data refresh (e.g. after the improve-labels job completes). */
  refreshKey?: number;
}

const POLL_STALE_INTERVAL_MS = 5000;

// ---------------------------------------------------------------------------
// ClusterTable
// ---------------------------------------------------------------------------

/**
 * ClusterTable — renders the subsystem cluster list for a repository.
 *
 * Columns (in order):
 *   1. Cluster label — inline-editable; saving fires a single-cluster rename.
 *   2. Member count — sortable, default descending.
 *   3. Top symbols — first 3 representative_symbols as <code> chips.
 *   4. Calls into — top-3 cross-cluster targets sorted by edge count.
 *
 * Stale banner: shown only when status is "pending" or "partial". When stale
 * AND empty, the stale banner alone is shown (it already implies data is being
 * recomputed, which subsumes the empty state).
 * Accessible sortable header: <button> inside <th> with aria-sort.
 */
export function ClusterTable({ repoId, refreshKey }: Props) {
  const [payload, setPayload] = useState<ClustersPayload | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>("descending");
  // Track optimistic label overrides: cluster id → pending label string.
  const [pendingLabels, setPendingLabels] = useState<Record<string, string>>({});
  const pollRef = useRef<number | null>(null);

  const fetchClusters = useCallback(async () => {
    try {
      const res = await authFetch(
        `/api/v1/repositories/${encodeURIComponent(repoId)}/clusters`,
      );
      if (!res.ok) {
        setError(
          res.status === 503
            ? "Couldn't load subsystems. The server isn't responding — try again in a moment."
            : `Couldn't load subsystems (${res.status}).`,
        );
        return;
      }
      const data = (await res.json()) as ClustersPayload;
      setPayload(data);
      setError(null);
    } catch {
      setError(
        "Couldn't load subsystems. The server isn't responding — try again in a moment.",
      );
    } finally {
      setLoading(false);
    }
  }, [repoId]);

  // Initial load + refresh when refreshKey changes.
  useEffect(() => {
    setLoading(true);
    void fetchClusters();
  }, [fetchClusters, refreshKey]);

  // Poll while status is stale/pending.
  useEffect(() => {
    const isStale =
      payload?.status === "pending" || payload?.status === "partial";
    if (!isStale) {
      if (pollRef.current !== null) {
        window.clearInterval(pollRef.current);
        pollRef.current = null;
      }
      return;
    }
    pollRef.current = window.setInterval(() => {
      void fetchClusters();
    }, POLL_STALE_INTERVAL_MS);
    return () => {
      if (pollRef.current !== null) {
        window.clearInterval(pollRef.current);
        pollRef.current = null;
      }
    };
  }, [payload?.status, fetchClusters]);

  // Optimistic label update: called by InlineEditLabel after a successful save.
  const handleLabelSaved = useCallback((clusterId: string, newLabel: string) => {
    setPendingLabels((prev) => ({ ...prev, [clusterId]: newLabel }));
  }, []);

  // Revert optimistic label on save failure.
  const handleLabelReverted = useCallback((clusterId: string) => {
    setPendingLabels((prev) => {
      const next = { ...prev };
      delete next[clusterId];
      return next;
    });
  }, []);

  // ---------------------------------------------------------------------------
  // Sort
  // ---------------------------------------------------------------------------

  const rows = payload?.clusters ?? [];
  const sorted = [...rows].sort((a, b) =>
    sortDir === "descending"
      ? b.member_count - a.member_count
      : a.member_count - b.member_count,
  );

  function toggleSort() {
    setSortDir((d) => (d === "descending" ? "ascending" : "descending"));
  }

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  if (loading) {
    return (
      <div className="py-8 text-center text-sm text-[var(--text-secondary)]">
        Loading subsystems…
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-[var(--radius-sm)] border border-[var(--border-subtle)] bg-[var(--bg-surface)] px-4 py-3">
        <p className="text-sm text-[var(--text-primary)]">Couldn&apos;t load subsystems</p>
        <p className="mt-1 text-xs text-[var(--text-tertiary)]">{error}</p>
      </div>
    );
  }

  const isStale =
    payload?.status === "pending" || payload?.status === "partial";
  const isEmpty = rows.length === 0;

  return (
    <div className="flex flex-col gap-3">
      {/* Stale banner — shown alone when stale; subsumes empty state. */}
      {isStale && (
        <div
          role="status"
          className="rounded-[var(--radius-sm)] border border-[var(--border-subtle)] bg-[var(--bg-surface)] px-4 py-2 text-sm text-[var(--text-secondary)]"
        >
          Recomputing subsystems…{payload?.retrieved_at ? ` last ran ${minutesAgo(payload.retrieved_at)}` : ""}
        </div>
      )}

      {/* Empty state — only shown when not stale. */}
      {!isStale && isEmpty && (
        <div className="rounded-[var(--radius-sm)] border border-[var(--border-subtle)] bg-[var(--bg-surface)] px-4 py-6 text-center text-sm text-[var(--text-secondary)]">
          No subsystems yet — indexing may still be running.
        </div>
      )}

      {/* Table — only shown when there are rows. */}
      {!isEmpty && (
        <div className="overflow-x-auto rounded-[var(--radius-sm)] border border-[var(--border-subtle)]">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border-subtle)] bg-[var(--bg-surface)]">
                <th
                  scope="col"
                  aria-sort="none"
                  className="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wide text-[var(--text-tertiary)]"
                >
                  Cluster label
                </th>
                <th
                  scope="col"
                  aria-sort={sortDir}
                  className="px-4 text-right text-xs font-medium uppercase tracking-wide text-[var(--text-tertiary)]"
                >
                  {/* Button carries the full touch-target padding so ≥44px is met. */}
                  <button
                    onClick={toggleSort}
                    className="inline-flex w-full items-center justify-end gap-1 px-0 py-2.5 rounded focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--focus-ring)]"
                    style={{ minHeight: "44px" }}
                    aria-label={`Sort by member count ${sortDir === "descending" ? "ascending" : "descending"}`}
                  >
                    Members
                    <span aria-hidden="true">{sortDir === "descending" ? "↓" : "↑"}</span>
                  </button>
                </th>
                <th
                  scope="col"
                  aria-sort="none"
                  className="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wide text-[var(--text-tertiary)]"
                >
                  Top symbols
                </th>
                <th
                  scope="col"
                  aria-sort="none"
                  className="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wide text-[var(--text-tertiary)]"
                >
                  Calls into
                </th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((row, idx) => (
                <ClusterRowItem
                  key={row.id}
                  row={row}
                  striped={idx % 2 === 1}
                  repoId={repoId}
                  pendingLabel={pendingLabels[row.id]}
                  onLabelSaved={handleLabelSaved}
                  onLabelReverted={handleLabelReverted}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// ClusterRowItem
// ---------------------------------------------------------------------------

interface ClusterRowItemProps {
  row: ClusterRow;
  striped: boolean;
  repoId: string;
  pendingLabel: string | undefined;
  onLabelSaved: (clusterId: string, newLabel: string) => void;
  onLabelReverted: (clusterId: string) => void;
}

function ClusterRowItem({ row, striped, repoId, pendingLabel, onLabelSaved, onLabelReverted }: ClusterRowItemProps) {
  const callsInto = buildCallsInto(row.cross_cluster_calls);
  const topSymbols = row.representative_symbols.slice(0, 3);
  const displayLabel = pendingLabel ?? row.label;

  return (
    <tr
      className={cn(
        "border-b border-[var(--border-subtle)] last:border-0",
        striped ? "bg-[var(--bg-surface)]" : "bg-[var(--bg-base)]",
      )}
    >
      {/* Cluster label — inline-editable */}
      <td className="px-4 py-2.5 font-medium text-[var(--text-primary)]">
        <InlineEditLabel
          clusterId={row.id}
          repoId={repoId}
          currentLabel={displayLabel}
          onSaved={(newLabel) => onLabelSaved(row.id, newLabel)}
          onReverted={() => onLabelReverted(row.id)}
        />
        {row.partial && (
          <span
            title="Clustering converged partially"
            className="ml-1.5 text-xs text-[var(--text-tertiary)]"
          >
            (partial)
          </span>
        )}
      </td>

      {/* Member count */}
      <td className="px-4 py-2.5 text-right tabular-nums text-[var(--text-secondary)]">
        {row.member_count}
      </td>

      {/* Top symbols */}
      <td className="px-4 py-2.5">
        <div className="flex flex-wrap gap-1">
          {topSymbols.length > 0 ? (
            topSymbols.map((sym) => (
              <code
                key={sym}
                title={sym}
                className="max-w-[200px] truncate rounded bg-[var(--bg-hover)] px-1.5 py-0.5 text-xs text-[var(--text-primary)]"
              >
                {sym}
              </code>
            ))
          ) : (
            <span className="text-xs text-[var(--text-tertiary)]">—</span>
          )}
        </div>
      </td>

      {/* Calls into */}
      <td className="px-4 py-2.5 text-[var(--text-secondary)]">
        {callsInto.length > 0 ? (
          <span className="text-xs">{callsInto.join(" · ")}</span>
        ) : (
          <span className="text-xs text-[var(--text-tertiary)]">—</span>
        )}
      </td>
    </tr>
  );
}

// ---------------------------------------------------------------------------
// InlineEditLabel
// ---------------------------------------------------------------------------

interface InlineEditLabelProps {
  clusterId: string;
  repoId: string;
  currentLabel: string;
  onSaved: (newLabel: string) => void;
  onReverted: () => void;
}

function InlineEditLabel({ clusterId, repoId, currentLabel, onSaved, onReverted }: InlineEditLabelProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(currentLabel);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Sync draft when currentLabel changes externally (e.g. table refresh).
  useEffect(() => {
    if (!editing) {
      setDraft(currentLabel);
    }
  }, [currentLabel, editing]);

  function enterEdit() {
    setDraft(currentLabel);
    setSaveError(null);
    setEditing(true);
    // Focus the input after the next render.
    setTimeout(() => inputRef.current?.select(), 0);
  }

  function cancelEdit() {
    setEditing(false);
    setSaveError(null);
  }

  async function commitEdit() {
    const trimmed = draft.trim();
    if (!trimmed || trimmed === currentLabel) {
      cancelEdit();
      return;
    }
    setSaving(true);
    setSaveError(null);

    // Optimistic: surface immediately via onSaved before the request completes.
    onSaved(trimmed);
    setEditing(false);

    try {
      const res = await authFetch(
        `/api/v1/repositories/${encodeURIComponent(repoId)}/clusters/relabel`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ cluster_ids: [clusterId], label: trimmed }),
        },
      );
      if (!res.ok) {
        // Roll back optimistic update.
        onReverted();
        setSaveError("Couldn't save label. Try again.");
        setEditing(true);
        setTimeout(() => inputRef.current?.select(), 0);
      }
    } catch {
      // Roll back optimistic update.
      onReverted();
      setSaveError("Couldn't save label. Try again.");
      setEditing(true);
      setTimeout(() => inputRef.current?.select(), 0);
    } finally {
      setSaving(false);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault();
      void commitEdit();
    } else if (e.key === "Escape") {
      e.preventDefault();
      cancelEdit();
    }
  }

  if (editing) {
    return (
      <span className="inline-flex items-center gap-1.5">
        <input
          ref={inputRef}
          type="text"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={() => void commitEdit()}
          disabled={saving}
          aria-label={`Rename cluster "${currentLabel}"`}
          className={cn(
            "rounded border border-[var(--border-strong)] bg-[var(--bg-base)] px-1.5 py-0.5 text-sm text-[var(--text-primary)] focus:outline-2 focus:outline-[var(--focus-ring)]",
            saving && "opacity-60",
          )}
          autoFocus
        />
        {saving && (
          <span
            className="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-[var(--text-secondary)] border-t-transparent"
            aria-label="Saving…"
          />
        )}
        {saveError && (
          <span
            role="alert"
            className="text-xs text-[var(--text-danger,red)]"
            title={saveError}
          >
            {saveError}
          </span>
        )}
      </span>
    );
  }

  return (
    <span className="group inline-flex items-center gap-1">
      <button
        type="button"
        onClick={enterEdit}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === "F2") enterEdit();
        }}
        title="Click to rename"
        aria-label={`Rename cluster "${currentLabel}"`}
        className="rounded text-left underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--focus-ring)]"
      >
        {currentLabel}
      </button>
      {/* Pencil affordance — visible on row hover */}
      <span
        aria-hidden="true"
        className="text-[var(--text-tertiary)] opacity-0 transition-opacity group-hover:opacity-100"
      >
        ✎
      </span>
      {saveError && (
        <span
          role="alert"
          className="text-xs text-[var(--text-danger,red)]"
          title={saveError}
        >
          {saveError}
        </span>
      )}
    </span>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Returns a human-readable "N minutes ago" / "N hours ago" string from an ISO
 * timestamp string. Falls back to "a moment ago" for very recent or invalid values.
 */
function minutesAgo(isoTimestamp: string): string {
  const d = new Date(isoTimestamp);
  if (Number.isNaN(d.getTime())) return "a moment ago";
  const diffMs = Date.now() - d.getTime();
  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) return "less than a minute ago";
  if (minutes === 1) return "1 minute ago";
  if (minutes < 60) return `${minutes} minutes ago`;
  const hours = Math.floor(minutes / 60);
  return hours === 1 ? "1 hour ago" : `${hours} hours ago`;
}

/**
 * Returns top-3 cluster labels sorted by edge count descending.
 */
function buildCallsInto(
  crossCalls: Record<string, number> | undefined,
): string[] {
  if (!crossCalls) return [];
  return Object.entries(crossCalls)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 3)
    .map(([label]) => label);
}
