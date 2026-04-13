"use client";

interface ActivityEnvelope<TJob> {
  active?: TJob[];
  recent?: TJob[];
  active_jobs?: TJob[];
  recent_jobs?: TJob[];
}

export function normalizeActivityResponse<TJob, TBody extends ActivityEnvelope<TJob>>(
  body: TBody
): TBody & { active: TJob[]; recent: TJob[] } {
  const active = Array.isArray(body.active)
    ? body.active
    : Array.isArray(body.active_jobs)
      ? body.active_jobs
      : [];
  const recent = Array.isArray(body.recent)
    ? body.recent
    : Array.isArray(body.recent_jobs)
      ? body.recent_jobs
      : [];
  return {
    ...body,
    active,
    recent,
  };
}
