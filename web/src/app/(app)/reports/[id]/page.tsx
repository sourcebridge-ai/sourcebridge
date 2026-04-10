"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";

import { PageFrame } from "@/components/ui/page-frame";
import { PageHeader } from "@/components/ui/page-header";
import { Panel } from "@/components/ui/panel";
import { TOKEN_KEY } from "@/lib/token-key";

interface ReportDetail {
  id: string;
  name: string;
  reportType: string;
  audience: string;
  repositoryIds: string[];
  status: string;
  progress: number;
  progressPhase: string;
  progressMessage: string;
  sectionCount: number;
  wordCount: number;
  evidenceCount: number;
  stale: boolean;
  version: number;
  contentDir: string;
  createdAt: string;
  completedAt?: string;
}

export default function ReportDetailPage() {
  const params = useParams();
  const reportId = params.id as string;

  const [report, setReport] = useState<ReportDetail | null>(null);
  const [markdown, setMarkdown] = useState("");
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

  const loadReport = useCallback(async () => {
    try {
      const res = await fetchWithAuth(`/api/v1/reports/${reportId}`);
      if (res.ok) {
        const data = await res.json();
        setReport(data);

        if (data.status === "ready" && data.contentDir) {
          const mdRes = await fetchWithAuth(`/api/v1/reports/${reportId}/markdown`);
          if (mdRes.ok) {
            const mdData = await mdRes.json();
            setMarkdown(mdData.markdown || "");
          }
        }
      }
    } catch (e) {
      console.error("Failed to load report:", e);
    } finally {
      setLoading(false);
    }
  }, [reportId, fetchWithAuth]);

  useEffect(() => {
    loadReport();
  }, [loadReport]);

  // Poll while generating
  useEffect(() => {
    if (!report || report.status === "ready" || report.status === "failed") return;
    const interval = setInterval(loadReport, 3000);
    return () => clearInterval(interval);
  }, [report, loadReport]);

  if (loading) {
    return (
      <PageFrame>
        <PageHeader eyebrow="Report" title="Loading..." />
      </PageFrame>
    );
  }

  if (!report) {
    return (
      <PageFrame>
        <PageHeader eyebrow="Report" title="Not Found" />
        <p className="text-sm text-[var(--text-secondary)]">This report does not exist.</p>
      </PageFrame>
    );
  }

  const isActive = report.status === "generating" || report.status === "collecting" || report.status === "rendering" || report.status === "pending";

  return (
    <PageFrame>
      <PageHeader
        eyebrow="Report"
        title={report.name}
        description={`${report.reportType.replace(/_/g, " ")} | v${report.version}`}
        actions={
          <div className="flex items-center gap-2">
            <Link href="/reports" className="text-sm text-[var(--text-secondary)] hover:underline">
              ← All reports
            </Link>
            {report.status === "ready" && (
              <>
                <a
                  href={`/api/v1/reports/${report.id}/download/markdown`}
                  className="inline-flex items-center rounded-[var(--control-radius)] border border-[var(--border-default)] bg-[var(--bg-base)] px-3 py-1.5 text-xs font-medium text-[var(--text-primary)] hover:bg-[var(--bg-hover)]"
                >
                  Markdown
                </a>
                <a
                  href={`/api/v1/reports/${report.id}/download/pdf`}
                  className="inline-flex items-center rounded-[var(--control-radius)] border border-[var(--border-default)] bg-[var(--bg-base)] px-3 py-1.5 text-xs font-medium text-[var(--text-primary)] hover:bg-[var(--bg-hover)]"
                >
                  PDF
                </a>
                <a
                  href={`/api/v1/reports/${report.id}/download/docx`}
                  className="inline-flex items-center rounded-[var(--control-radius)] border border-[var(--border-default)] bg-[var(--bg-base)] px-3 py-1.5 text-xs font-medium text-[var(--text-primary)] hover:bg-[var(--bg-hover)]"
                >
                  Word
                </a>
              </>
            )}
          </div>
        }
      />

      {/* Status / Progress */}
      {isActive && (
        <Panel>
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-[var(--text-primary)]">
                {report.progressMessage || report.progressPhase || "Preparing..."}
              </span>
              <span className="font-mono text-[var(--text-secondary)]">
                {Math.round(report.progress * 100)}%
              </span>
            </div>
            <progress
              className="h-2 w-full overflow-hidden rounded-full [&::-webkit-progress-bar]:bg-[var(--bg-hover)] [&::-webkit-progress-value]:bg-[var(--accent-primary)]"
              max={100}
              value={Math.max(report.progress * 100, 3)}
            />
          </div>
        </Panel>
      )}

      {/* Failed */}
      {report.status === "failed" && (
        <Panel>
          <p className="text-sm text-red-400">
            Report generation failed: {report.progressMessage || "Unknown error"}
          </p>
        </Panel>
      )}

      {/* Stats */}
      {report.status === "ready" && (
        <div className="flex flex-wrap gap-4 text-xs text-[var(--text-secondary)]">
          <span>{report.sectionCount} sections</span>
          <span>{report.wordCount?.toLocaleString()} words</span>
          <span>{report.evidenceCount} evidence items</span>
          {report.stale && <span className="text-amber-400">Stale — underlying repos have changed</span>}
        </div>
      )}

      {/* Markdown Preview */}
      {report.status === "ready" && markdown && (
        <Panel className="mt-4">
          <div
            className="prose prose-sm max-w-none text-[var(--text-primary)] prose-headings:text-[var(--text-primary)] prose-p:text-[var(--text-secondary)] prose-strong:text-[var(--text-primary)] prose-code:text-[var(--accent-primary)]"
            dangerouslySetInnerHTML={{ __html: markdownToHtml(markdown) }}
          />
        </Panel>
      )}
    </PageFrame>
  );
}

/** Minimal Markdown to HTML for preview. A proper renderer would use react-markdown. */
function markdownToHtml(md: string): string {
  const html = md
    // Headings
    .replace(/^### (.+)$/gm, "<h3>$1</h3>")
    .replace(/^## (.+)$/gm, "<h2>$1</h2>")
    .replace(/^# (.+)$/gm, "<h1>$1</h1>")
    // Bold
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    // Italic
    .replace(/\*(.+?)\*/g, "<em>$1</em>")
    // Code
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    // Blockquotes
    .replace(/^> (.+)$/gm, "<blockquote>$1</blockquote>")
    // Horizontal rules
    .replace(/^---$/gm, "<hr>")
    // Line breaks
    .replace(/\n\n/g, "</p><p>")
    .replace(/\n/g, "<br>");

  return `<p>${html}</p>`;
}
