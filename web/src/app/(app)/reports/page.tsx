"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { PageFrame } from "@/components/ui/page-frame";
import { PageHeader } from "@/components/ui/page-header";
import { Panel } from "@/components/ui/panel";
import { TOKEN_KEY } from "@/lib/token-key";
import { cn } from "@/lib/utils";

interface ReportSummary {
  id: string;
  name: string;
  reportType: string;
  audience: string;
  repositoryIds: string[];
  status: string;
  progress: number;
  progressMessage: string;
  sectionCount: number;
  wordCount: number;
  stale: boolean;
  createdAt: string;
  completedAt?: string;
}

const TYPE_LABELS: Record<string, string> = {
  architecture_baseline: "Architecture Baseline",
  swot: "SWOT Analysis",
  environment_eval: "Environment Evaluation",
  portfolio_health: "Portfolio Health",
  due_diligence: "Due Diligence",
  compliance_gap: "Compliance Gap",
};

const AUDIENCE_LABELS: Record<string, string> = {
  c_suite: "C-Suite",
  executive: "Executive",
  technical_leadership: "Technical Lead",
  developer: "Developer",
  compliance: "Compliance",
  non_technical: "Non-Technical",
};

function statusBadge(status: string) {
  const styles: Record<string, string> = {
    ready: "bg-green-500/20 text-green-400 border-green-500/30",
    generating: "bg-blue-500/20 text-blue-400 border-blue-500/30 animate-pulse",
    collecting: "bg-blue-500/20 text-blue-400 border-blue-500/30 animate-pulse",
    rendering: "bg-blue-500/20 text-blue-400 border-blue-500/30 animate-pulse",
    pending: "bg-amber-500/20 text-amber-400 border-amber-500/30",
    failed: "bg-red-500/20 text-red-400 border-red-500/30",
  };
  return (
    <span className={cn("inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium", styles[status] || styles.pending)}>
      {status}
    </span>
  );
}

export default function ReportsPage() {
  const [reports, setReports] = useState<ReportSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchWithAuth = useCallback(async (path: string) => {
    const token = typeof window !== "undefined" ? localStorage.getItem(TOKEN_KEY) : null;
    return fetch(path, {
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    });
  }, []);

  const loadReports = useCallback(async () => {
    try {
      const res = await fetchWithAuth("/api/v1/reports");
      if (res.ok) {
        const data = await res.json();
        setReports(Array.isArray(data) ? data : []);
      }
    } catch (e) {
      console.error("Failed to load reports:", e);
    } finally {
      setLoading(false);
    }
  }, [fetchWithAuth]);

  useEffect(() => {
    loadReports();
    // Poll for active reports
    const interval = setInterval(loadReports, 5000);
    return () => clearInterval(interval);
  }, [loadReports]);

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete report "${name}"? This cannot be undone.`)) return;
    const token = typeof window !== "undefined" ? localStorage.getItem(TOKEN_KEY) : null;
    await fetch(`/api/v1/reports/${id}`, {
      method: "DELETE",
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    loadReports();
  };

  if (loading) {
    return (
      <PageFrame>
        <PageHeader eyebrow="Reports" title="Loading..." />
      </PageFrame>
    );
  }

  return (
    <PageFrame>
      <PageHeader
        eyebrow="Reports"
        title="Professional Reports"
        description="Generate evidence-backed reports from your codebase analysis. Choose a report type, select repositories, and get a professional document."
        actions={
          <Link href="/reports/new">
            <Button>New Report</Button>
          </Link>
        }
      />

      {reports.length === 0 ? (
        <EmptyState
          title="No reports yet"
          description="Reports turn SourceBridge's analysis into professional documents you can share. Pick a report type to get started."
          actions={
            <Link href="/reports/new">
              <Button>Create your first report</Button>
            </Link>
          }
        />
      ) : (
        <div className="space-y-3">
          {reports.map((r) => (
            <Panel key={r.id} className="flex items-center justify-between gap-4">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <Link
                    href={`/reports/${r.id}`}
                    className="truncate text-sm font-semibold text-[var(--text-primary)] hover:underline"
                  >
                    {r.name}
                  </Link>
                  {statusBadge(r.status)}
                  {r.stale && (
                    <span className="rounded-full bg-amber-500/20 border border-amber-500/30 px-2 py-0.5 text-xs text-amber-400">
                      stale
                    </span>
                  )}
                </div>
                <div className="mt-1 flex items-center gap-3 text-xs text-[var(--text-secondary)]">
                  <span>{TYPE_LABELS[r.reportType] || r.reportType}</span>
                  <span>{AUDIENCE_LABELS[r.audience] || r.audience}</span>
                  <span>{r.repositoryIds?.length || 0} repos</span>
                  {r.status === "ready" && (
                    <>
                      <span>{r.sectionCount} sections</span>
                      <span>{r.wordCount?.toLocaleString()} words</span>
                    </>
                  )}
                  {r.status === "generating" || r.status === "collecting" ? (
                    <span>{r.progressMessage || `${Math.round(r.progress * 100)}%`}</span>
                  ) : null}
                </div>
              </div>
              <div className="flex items-center gap-2">
                {r.status === "ready" && (
                  <Link
                    href={`/reports/${r.id}`}
                    className="text-xs text-[var(--accent-primary)] hover:underline"
                  >
                    View
                  </Link>
                )}
                <button
                  onClick={() => handleDelete(r.id, r.name)}
                  className="text-xs text-[var(--text-muted)] hover:text-red-400"
                >
                  Delete
                </button>
              </div>
            </Panel>
          ))}
        </div>
      )}
    </PageFrame>
  );
}
