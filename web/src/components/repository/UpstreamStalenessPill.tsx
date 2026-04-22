"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery } from "urql";
import { Button } from "@/components/ui/button";
import {
  REINDEX_REPOSITORY_MUTATION,
  REPOSITORY_UPSTREAM_STATUS_QUERY,
} from "@/lib/graphql/queries";

type UpstreamStatus =
  | "UP_TO_DATE"
  | "BEHIND"
  | "UNKNOWN"
  | "UNREACHABLE"
  | "UNSUPPORTED";

type UpstreamStatusResult = {
  repository: {
    id: string;
    commitSha: string | null;
    upstreamStatus: {
      status: UpstreamStatus;
      upstreamCommitSha: string | null;
      indexedCommitSha: string | null;
      checkedAt: string;
      errorMessage: string | null;
    } | null;
  } | null;
};

// Default polling interval (ms) while the repo page is visible. Kept
// slightly longer than the server-side 30s cache TTL so we don't
// pound the upstream check during cache-miss windows across viewers.
const POLL_INTERVAL_MS = 60_000;

type UpstreamStalenessPillProps = {
  repositoryId: string;
  /** Override for tests / non-standard cadences. */
  intervalMs?: number;
};

/**
 * Subtle badge rendered near the repository title that surfaces
 * upstream-vs-indexed drift. Renders nothing when the repo is up to
 * date, has no remote (UNSUPPORTED), or we've never managed a check
 * (UNKNOWN). Offers a one-click "Reindex" action when the repo is
 * BEHIND; shows a muted "Upstream unreachable" tooltip on UNREACHABLE
 * without demanding user action.
 *
 * Polls only while the tab is visible. Stops when the tab is hidden
 * so a forgotten tab doesn't burn API-call budget.
 */
export function UpstreamStalenessPill({
  repositoryId,
  intervalMs = POLL_INTERVAL_MS,
}: UpstreamStalenessPillProps) {
  const [result, reexecute] = useQuery<UpstreamStatusResult>({
    query: REPOSITORY_UPSTREAM_STATUS_QUERY,
    variables: { id: repositoryId },
    // Page-visibility-aware polling is set up below.
    pause: false,
  });

  const [, reindex] = useMutation(REINDEX_REPOSITORY_MUTATION);
  const [reindexing, setReindexing] = useState(false);

  // Poll while the tab is visible. When the tab is hidden we pause so
  // forgotten tabs don't run up upstream checks for hours.
  useEffect(() => {
    if (!repositoryId) return;
    let timer: ReturnType<typeof setInterval> | null = null;

    const start = () => {
      if (timer) return;
      timer = setInterval(() => {
        reexecute({ requestPolicy: "network-only" });
      }, intervalMs);
    };
    const stop = () => {
      if (timer) {
        clearInterval(timer);
        timer = null;
      }
    };

    const onVisibility = () => {
      if (document.visibilityState === "visible") {
        // Kick a fresh check immediately when the tab becomes visible
        // so users coming back to the page don't wait a full interval.
        reexecute({ requestPolicy: "network-only" });
        start();
      } else {
        stop();
      }
    };

    // Initial arming — only run the interval if the tab is currently
    // visible. The subscribe covers later tab switches.
    if (document.visibilityState === "visible") start();
    document.addEventListener("visibilitychange", onVisibility);

    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      stop();
    };
  }, [repositoryId, intervalMs, reexecute]);

  const status = result.data?.repository?.upstreamStatus ?? null;

  // Nothing to show for these states.
  if (!status) return null;
  if (status.status === "UP_TO_DATE") return null;
  if (status.status === "UNSUPPORTED") return null;
  if (status.status === "UNKNOWN") return null;

  if (status.status === "UNREACHABLE") {
    return (
      <span
        className="inline-flex items-center gap-1.5 rounded-full border border-[var(--border-default)] bg-[var(--bg-surface)] px-2.5 py-0.5 text-xs text-[var(--text-tertiary)]"
        title={status.errorMessage ?? "Upstream check failed"}
      >
        <span className="h-1.5 w-1.5 rounded-full bg-[var(--text-tertiary)]" aria-hidden />
        Upstream unreachable
      </span>
    );
  }

  const handleReindex = async () => {
    if (reindexing) return;
    setReindexing(true);
    try {
      await reindex({ id: repositoryId });
      // Force a fresh upstream check — once reindex lands, the
      // indexedCommitSha should match upstream and the pill will hide
      // itself on the next poll.
      reexecute({ requestPolicy: "network-only" });
    } finally {
      setReindexing(false);
    }
  };

  const upstream = status.upstreamCommitSha;
  const indexed = status.indexedCommitSha;
  return (
    <span
      className="inline-flex items-center gap-2 rounded-full border border-amber-500/40 bg-amber-500/10 px-2.5 py-0.5 text-xs font-medium text-amber-500"
      title={
        upstream && indexed
          ? `Indexed ${indexed.slice(0, 7)} · Upstream ${upstream.slice(0, 7)}`
          : "Upstream has changes since last index"
      }
    >
      <span className="h-1.5 w-1.5 rounded-full bg-amber-500" aria-hidden />
      Upstream has new commits
      <Button
        variant="ghost"
        size="sm"
        onClick={handleReindex}
        disabled={reindexing}
        className="h-6 px-2 text-xs text-amber-500 hover:bg-amber-500/10"
      >
        {reindexing ? "Reindexing…" : "Reindex"}
      </Button>
    </span>
  );
}
