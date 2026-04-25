// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

/**
 * Canonical citation parser for SourceBridge citation handles.
 *
 * Mirrors the format defined in internal/citations/citations.go.
 *
 * On-wire format variants:
 *   - File range:    "path:startLine-endLine"  (e.g. "internal/auth/auth.go:42-55")
 *   - Single line:   "path:line"               (e.g. "foo.go:10")
 *   - Symbol:        "sym_<id>"                (e.g. "sym_abc123")
 *   - Requirement:   any other string           (verbatim external ID)
 */

export type CitationKind = "file_range" | "symbol" | "requirement";

export interface FileRangeCitation {
  kind: "file_range";
  path: string;
  startLine: number;
  endLine: number;
}

export interface SymbolCitation {
  kind: "symbol";
  symbolId: string;
}

export interface RequirementCitation {
  kind: "requirement";
  requirementId: string;
}

export type Citation = FileRangeCitation | SymbolCitation | RequirementCitation;

/**
 * Parse a citation handle string into a typed Citation.
 * Returns null for empty or whitespace-only strings.
 */
export function parseCitation(handle: string): Citation | null {
  const s = handle.trim();
  if (!s) return null;

  // Symbol handle.
  if (s.startsWith("sym_")) {
    const id = s.slice(4);
    if (!id) return null;
    return { kind: "symbol", symbolId: s };
  }

  // Attempt file-range parse: rightmost ":" separates path from line spec.
  const colonIdx = s.lastIndexOf(":");
  if (colonIdx > 0) {
    const tail = s.slice(colonIdx + 1);
    const range = parseLineRange(tail);
    if (range) {
      return {
        kind: "file_range",
        path: s.slice(0, colonIdx),
        startLine: range.start,
        endLine: range.end,
      };
    }
  }

  // Fall through to requirement.
  return { kind: "requirement", requirementId: s };
}

/**
 * Format a Citation back to its canonical string representation.
 * Inverse of parseCitation — Format(Parse(s)) === s for well-formed inputs.
 */
export function formatCitation(citation: Citation): string {
  switch (citation.kind) {
    case "file_range":
      if (citation.startLine === citation.endLine) {
        return `${citation.path}:${citation.startLine}-${citation.endLine}`;
      }
      return `${citation.path}:${citation.startLine}-${citation.endLine}`;
    case "symbol":
      return citation.symbolId;
    case "requirement":
      return citation.requirementId;
  }
}

/**
 * Parse the tail after the last ":" into a line range.
 * Accepts "n" (single line, mapped to n-n) or "n-m" (range).
 * Returns null for non-numeric tails.
 */
function parseLineRange(tail: string): { start: number; end: number } | null {
  const dashIdx = tail.indexOf("-");
  if (dashIdx < 0) {
    const n = parseInt(tail, 10);
    if (isNaN(n) || n <= 0) return null;
    return { start: n, end: n };
  }
  const start = parseInt(tail.slice(0, dashIdx), 10);
  const end = parseInt(tail.slice(dashIdx + 1), 10);
  if (isNaN(start) || isNaN(end) || start <= 0 || end < start) return null;
  return { start, end };
}

/**
 * Convenience: extract just the file path and start line from any
 * citation string, for callers that only need to jump-to-line.
 *
 * - File range → (path, startLine)
 * - Symbol → (undefined, undefined) — callers must look up the symbol
 * - Requirement → (undefined, undefined) — no file location
 */
export function citationToFileLocation(
  handle: string
): { filePath: string; line: number } | null {
  const c = parseCitation(handle);
  if (!c || c.kind !== "file_range") return null;
  return { filePath: c.path, line: c.startLine };
}
