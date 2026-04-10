"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "urql";

import { Button } from "@/components/ui/button";
import { PageFrame } from "@/components/ui/page-frame";
import { PageHeader } from "@/components/ui/page-header";
import { Panel } from "@/components/ui/panel";
import { REPOSITORIES_LIGHT_QUERY } from "@/lib/graphql/queries";
import { TOKEN_KEY } from "@/lib/token-key";
import { cn } from "@/lib/utils";

/**
 * New Report Wizard — 5-step conversational flow.
 *
 * Step 1: Report Type (cards)
 * Step 2: Audience (cards with tone samples)
 * Step 3: Select Repositories
 * Step 4: Configure Sections
 * Step 5: Confirm & Generate
 */

interface ReportTypeDef {
  type: string;
  title: string;
  description: string;
  sections: { key: string; title: string; category: string; description: string }[];
}

interface Repository {
  id: string;
  name: string;
  status: string;
  fileCount: number;
}

const AUDIENCE_OPTIONS = [
  { key: "c_suite", title: "C-Suite / Board", sample: "The portfolio carries significant unaddressed risk. Student data is accessible without authorization." },
  { key: "executive", title: "Executive / VP", sample: "The four Supabase-backed applications bypass Row Level Security via service-role keys." },
  { key: "technical_leadership", title: "Technical Leadership", sample: "All Next.js apps use createClient with SUPABASE_SERVICE_KEY instead of the SSR-aware client." },
  { key: "developer", title: "Developer", sample: "Fix app/api/application/lookup/route.ts L5-37: add session check before the RPC call." },
  { key: "compliance", title: "Compliance / Audit", sample: "Finding: No automated testing controls exist. This represents a gap in change management controls per FERPA 34 CFR 99.31." },
  { key: "non_technical", title: "Non-Technical Stakeholder", sample: "Think of it like a building with no locks on some doors. The fix takes days, not months." },
];

const LOE_OPTIONS = [
  { key: "human_hours", title: "Human Hours", desc: "Traditional person-hours/weeks estimates" },
  { key: "ai_assisted", title: "AI-Assisted", desc: "AI agent hours + human review hours" },
];

export default function NewReportWizard() {
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [reportTypes, setReportTypes] = useState<ReportTypeDef[]>([]);

  // Wizard state
  const [selectedType, setSelectedType] = useState("");
  const [selectedAudience, setSelectedAudience] = useState("technical_leadership");
  const [selectedRepos, setSelectedRepos] = useState<string[]>([]);
  const [selectedSections, setSelectedSections] = useState<string[]>([]);
  const [allSections, setAllSections] = useState<{ key: string; title: string; category: string }[]>([]);
  const [reportName, setReportName] = useState("");
  const [loeMode, setLoeMode] = useState("human_hours");
  const [includeDiagrams, setIncludeDiagrams] = useState(false);
  const [outputFormats] = useState<string[]>(["markdown"]);
  const [creating, setCreating] = useState(false);

  const [reposResult] = useQuery({ query: REPOSITORIES_LIGHT_QUERY });
  const repos: Repository[] = reposResult.data?.repositories || [];

  const fetchWithAuth = useCallback(async (path: string, options?: RequestInit) => {
    const token = typeof window !== "undefined" ? localStorage.getItem(TOKEN_KEY) : null;
    return fetch(path, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...options?.headers,
      },
    });
  }, []);

  // Load report types
  useEffect(() => {
    fetchWithAuth("/api/v1/reports/types").then(async (res) => {
      if (res.ok) {
        const data = await res.json();
        setReportTypes(Array.isArray(data) ? data : []);
      }
    });
  }, [fetchWithAuth]);

  // Load default sections when type or audience changes
  useEffect(() => {
    if (!selectedType) return;
    fetchWithAuth(`/api/v1/reports/sections?report_type=${selectedType}&audience=${selectedAudience}`).then(async (res) => {
      if (res.ok) {
        const data = await res.json();
        setSelectedSections(data.defaultSections || []);
        setAllSections(data.allSections || []);
      }
    });
  }, [selectedType, selectedAudience, fetchWithAuth]);

  const selectedTypeDef = reportTypes.find((t) => t.type === selectedType);

  const handleCreate = async () => {
    setCreating(true);
    try {
      const res = await fetchWithAuth("/api/v1/reports", {
        method: "POST",
        body: JSON.stringify({
          name: reportName || `${selectedTypeDef?.title || "Report"} — ${new Date().toLocaleDateString()}`,
          reportType: selectedType,
          audience: selectedAudience,
          repositoryIds: selectedRepos,
          selectedSections,
          includeDiagrams,
          outputFormats,
          loeMode,
        }),
      });
      if (res.ok) {
        const created = await res.json();
        router.push(`/reports/${created.id}`);
      }
    } finally {
      setCreating(false);
    }
  };

  // Group sections by category
  const sectionsByCategory: Record<string, typeof allSections> = {};
  for (const sec of allSections) {
    sectionsByCategory[sec.category] = sectionsByCategory[sec.category] || [];
    sectionsByCategory[sec.category].push(sec);
  }

  return (
    <PageFrame>
      <PageHeader
        eyebrow={`New Report — Step ${step} of 5`}
        title={
          step === 1
            ? "What kind of report?"
            : step === 2
              ? "Who is this for?"
              : step === 3
                ? "Which repositories?"
                : step === 4
                  ? "What should it cover?"
                  : "Ready to generate"
        }
      />

      {/* Step 1: Report Type */}
      {step === 1 && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {reportTypes.map((rt) => (
            <Panel
              key={rt.type}
              className={cn(
                "cursor-pointer transition-all hover:border-[var(--accent-primary)]",
                selectedType === rt.type && "border-[var(--accent-primary)] bg-[var(--accent-primary)]/5"
              )}
              onClick={() => {
                setSelectedType(rt.type);
                setReportName(`${rt.title} — ${new Date().toLocaleDateString()}`);
              }}
            >
              <h3 className="text-base font-semibold text-[var(--text-primary)]">{rt.title}</h3>
              <p className="mt-1 text-sm text-[var(--text-secondary)]">{rt.description}</p>
              <p className="mt-2 text-xs text-[var(--text-muted)]">{rt.sections.length} sections available</p>
            </Panel>
          ))}
        </div>
      )}

      {/* Step 2: Audience */}
      {step === 2 && (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {AUDIENCE_OPTIONS.map((aud) => (
            <Panel
              key={aud.key}
              className={cn(
                "cursor-pointer transition-all hover:border-[var(--accent-primary)]",
                selectedAudience === aud.key && "border-[var(--accent-primary)] bg-[var(--accent-primary)]/5"
              )}
              onClick={() => setSelectedAudience(aud.key)}
            >
              <h3 className="text-sm font-semibold text-[var(--text-primary)]">{aud.title}</h3>
              <p className="mt-2 text-xs italic text-[var(--text-secondary)]">&ldquo;{aud.sample}&rdquo;</p>
            </Panel>
          ))}
        </div>
      )}

      {/* Step 3: Repositories */}
      {step === 3 && (
        <Panel>
          <div className="mb-3 flex items-center justify-between">
            <p className="text-sm text-[var(--text-secondary)]">
              {selectedRepos.length} of {repos.length} selected
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setSelectedRepos(repos.map((r) => r.id))}
                className="text-xs text-[var(--accent-primary)] hover:underline"
              >
                Select all
              </button>
              <button
                onClick={() => setSelectedRepos([])}
                className="text-xs text-[var(--text-muted)] hover:underline"
              >
                Deselect all
              </button>
            </div>
          </div>
          <div className="space-y-1">
            {repos.map((repo) => (
              <label
                key={repo.id}
                className="flex cursor-pointer items-center gap-3 rounded-[var(--control-radius)] px-3 py-2 hover:bg-[var(--bg-hover)]"
              >
                <input
                  type="checkbox"
                  checked={selectedRepos.includes(repo.id)}
                  onChange={(e) => {
                    if (e.target.checked) {
                      setSelectedRepos((prev) => [...prev, repo.id]);
                    } else {
                      setSelectedRepos((prev) => prev.filter((id) => id !== repo.id));
                    }
                  }}
                  className="rounded border-[var(--border-default)]"
                />
                <span className="flex-1 text-sm text-[var(--text-primary)]">{repo.name}</span>
                <span className="text-xs text-[var(--text-muted)]">{repo.fileCount} files</span>
              </label>
            ))}
          </div>
        </Panel>
      )}

      {/* Step 4: Sections */}
      {step === 4 && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm text-[var(--text-secondary)]">
              {selectedSections.length} sections selected
            </p>
            <button
              onClick={() => {
                if (selectedSections.length === allSections.length) {
                  setSelectedSections([]);
                } else {
                  setSelectedSections(allSections.map((s) => s.key));
                }
              }}
              className="text-xs text-[var(--accent-primary)] hover:underline"
            >
              {selectedSections.length === allSections.length ? "Deselect all" : "Select all"}
            </button>
          </div>
          {Object.entries(sectionsByCategory).map(([category, sections]) => (
            <Panel key={category} padding="sm">
              <div className="mb-2 flex items-center justify-between">
                <h4 className="text-xs font-semibold uppercase tracking-wider text-[var(--text-secondary)]">
                  {category}
                </h4>
                <button
                  onClick={() => {
                    const catKeys = sections.map((s) => s.key);
                    const allSelected = catKeys.every((k) => selectedSections.includes(k));
                    if (allSelected) {
                      setSelectedSections((prev) => prev.filter((k) => !catKeys.includes(k)));
                    } else {
                      setSelectedSections((prev) => [...new Set([...prev, ...catKeys])]);
                    }
                  }}
                  className="text-[10px] text-[var(--accent-primary)] hover:underline"
                >
                  Toggle
                </button>
              </div>
              <div className="space-y-0.5">
                {sections.map((sec) => (
                  <label
                    key={sec.key}
                    className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-[var(--bg-hover)]"
                  >
                    <input
                      type="checkbox"
                      checked={selectedSections.includes(sec.key)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setSelectedSections((prev) => [...prev, sec.key]);
                        } else {
                          setSelectedSections((prev) => prev.filter((k) => k !== sec.key));
                        }
                      }}
                      className="rounded border-[var(--border-default)]"
                    />
                    <span className="text-sm text-[var(--text-primary)]">{sec.title}</span>
                  </label>
                ))}
              </div>
            </Panel>
          ))}

          <Panel padding="sm">
            <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--text-secondary)]">
              Options
            </h4>
            <div className="space-y-2">
              <label className="flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                <input
                  type="checkbox"
                  checked={includeDiagrams}
                  onChange={(e) => setIncludeDiagrams(e.target.checked)}
                  className="rounded border-[var(--border-default)]"
                />
                Include diagram placeholders
              </label>
              <div className="flex items-center gap-3">
                <span className="text-sm text-[var(--text-secondary)]">LOE mode:</span>
                {LOE_OPTIONS.map((opt) => (
                  <label key={opt.key} className="flex items-center gap-1 text-sm text-[var(--text-secondary)]">
                    <input
                      type="radio"
                      name="loe"
                      checked={loeMode === opt.key}
                      onChange={() => setLoeMode(opt.key)}
                    />
                    {opt.title}
                  </label>
                ))}
              </div>
            </div>
          </Panel>
        </div>
      )}

      {/* Step 5: Confirm & Generate */}
      {step === 5 && (
        <Panel>
          <div className="space-y-4">
            <div>
              <label className="mb-1.5 block text-sm font-medium text-[var(--text-secondary)]">Report name</label>
              <input
                value={reportName}
                onChange={(e) => setReportName(e.target.value)}
                className="w-full rounded-[var(--control-radius)] border border-[var(--border-default)] bg-[var(--bg-base)] px-3 py-2 text-sm text-[var(--text-primary)]"
              />
            </div>

            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-[var(--text-secondary)]">Report type:</span>
                <span className="ml-2 font-medium text-[var(--text-primary)]">{selectedTypeDef?.title}</span>
              </div>
              <div>
                <span className="text-[var(--text-secondary)]">Audience:</span>
                <span className="ml-2 font-medium text-[var(--text-primary)]">
                  {AUDIENCE_OPTIONS.find((a) => a.key === selectedAudience)?.title}
                </span>
              </div>
              <div>
                <span className="text-[var(--text-secondary)]">Repositories:</span>
                <span className="ml-2 font-medium text-[var(--text-primary)]">{selectedRepos.length} selected</span>
              </div>
              <div>
                <span className="text-[var(--text-secondary)]">Sections:</span>
                <span className="ml-2 font-medium text-[var(--text-primary)]">{selectedSections.length} selected</span>
              </div>
            </div>

            <Button onClick={handleCreate} disabled={creating}>
              {creating ? "Creating..." : "Generate Report"}
            </Button>
          </div>
        </Panel>
      )}

      {/* Navigation */}
      <div className="mt-6 flex items-center justify-between">
        <Button
          variant="ghost"
          onClick={() => setStep((s) => Math.max(1, s - 1))}
          disabled={step === 1}
        >
          Back
        </Button>
        <div className="flex gap-1">
          {[1, 2, 3, 4, 5].map((s) => (
            <div
              key={s}
              className={cn(
                "h-2 w-8 rounded-full transition-colors",
                s <= step ? "bg-[var(--accent-primary)]" : "bg-[var(--border-default)]"
              )}
            />
          ))}
        </div>
        {step < 5 ? (
          <Button
            onClick={() => setStep((s) => Math.min(5, s + 1))}
            disabled={
              (step === 1 && !selectedType) ||
              (step === 3 && selectedRepos.length === 0) ||
              (step === 4 && selectedSections.length === 0)
            }
          >
            Next
          </Button>
        ) : (
          <div />
        )}
      </div>
    </PageFrame>
  );
}
