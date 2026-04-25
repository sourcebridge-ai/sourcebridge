// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

import { parseCitation, formatCitation, citationToFileLocation } from "../citations";

describe("parseCitation", () => {
  it("returns null for empty string", () => {
    expect(parseCitation("")).toBeNull();
  });

  it("returns null for whitespace-only string", () => {
    expect(parseCitation("   ")).toBeNull();
  });

  it("parses a file-range citation", () => {
    const c = parseCitation("internal/auth/auth.go:42-55");
    expect(c).toEqual({
      kind: "file_range",
      path: "internal/auth/auth.go",
      startLine: 42,
      endLine: 55,
    });
  });

  it("parses a single-line file citation (maps to start===end)", () => {
    const c = parseCitation("foo.go:10");
    expect(c).toEqual({
      kind: "file_range",
      path: "foo.go",
      startLine: 10,
      endLine: 10,
    });
  });

  it("parses a symbol citation", () => {
    const c = parseCitation("sym_abc123");
    expect(c).toEqual({ kind: "symbol", symbolId: "sym_abc123" });
  });

  it("returns null for bare sym_ prefix with no id", () => {
    expect(parseCitation("sym_")).toBeNull();
  });

  it("parses a requirement citation (no structure)", () => {
    const c = parseCitation("REQ-001");
    expect(c).toEqual({ kind: "requirement", requirementId: "REQ-001" });
  });

  it("handles a deeply nested path", () => {
    const c = parseCitation("internal/auth/v2/auth.go:100-200");
    expect(c).toEqual({
      kind: "file_range",
      path: "internal/auth/v2/auth.go",
      startLine: 100,
      endLine: 200,
    });
  });

  it("falls back to requirement for colon-containing non-range strings", () => {
    const c = parseCitation("https://example.com/path");
    expect(c).not.toBeNull();
    expect(c!.kind).toBe("requirement");
  });
});

describe("formatCitation (round-trip)", () => {
  const cases = [
    "internal/auth/auth.go:42-55",
    "foo.go:10-10",
    "sym_abc123",
    "REQ-001",
  ];

  for (const s of cases) {
    it(`round-trips: ${s}`, () => {
      const c = parseCitation(s);
      expect(c).not.toBeNull();
      expect(formatCitation(c!)).toBe(s);
    });
  }
});

describe("citationToFileLocation", () => {
  it("returns file path and startLine for a range citation", () => {
    const loc = citationToFileLocation("internal/auth/auth.go:42-55");
    expect(loc).toEqual({ filePath: "internal/auth/auth.go", line: 42 });
  });

  it("returns file path and line for a single-line citation", () => {
    const loc = citationToFileLocation("foo.go:10");
    expect(loc).toEqual({ filePath: "foo.go", line: 10 });
  });

  it("returns null for symbol citations", () => {
    expect(citationToFileLocation("sym_abc123")).toBeNull();
  });

  it("returns null for requirement citations", () => {
    expect(citationToFileLocation("REQ-001")).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(citationToFileLocation("")).toBeNull();
  });
});
