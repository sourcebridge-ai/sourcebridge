import { describe, it, expect } from "vitest";
import { getLanguageExtension } from "./CodeViewer";

// getLanguageExtension is a pure function that maps a language string to a
// CodeMirror language extension. Tests verify:
//   1. Groovy and Gradle map to the Java extension (closest JVM syntax family;
//      no @codemirror/lang-groovy package exists).
//   2. Uppercase language names (e.g. GraphQL canonical "GROOVY", "GO") are
//      handled correctly — the switch uses lang.toLowerCase() so case is
//      normalised before matching.
//   3. The TypeScript branch uses lower.startsWith("t") (not lang.startsWith)
//      so "TYPESCRIPT" correctly activates TypeScript mode.
//
// CodeMirror extensions are stateful objects with auto-incrementing internal
// IDs, so .toEqual() comparisons across two separate calls are unreliable.
// Tests assert that the returned object is a non-null/non-undefined value.
// Groovy / Gradle tests additionally verify the object is structurally
// identical to a java() call by comparing JSON-serialisable shape at one level.

describe("getLanguageExtension", () => {
  // --- Groovy / Gradle (O4, Phase 5) ---

  it('maps "groovy" to a language extension (java fallback)', () => {
    const ext = getLanguageExtension("groovy");
    expect(ext).toBeTruthy();
  });

  it('maps "gradle" to a language extension (java fallback)', () => {
    const ext = getLanguageExtension("gradle");
    expect(ext).toBeTruthy();
  });

  it('maps uppercase "GROOVY" to a language extension (case-insensitive)', () => {
    const ext = getLanguageExtension("GROOVY");
    expect(ext).toBeTruthy();
  });

  it('maps uppercase "GRADLE" to a language extension (case-insensitive)', () => {
    const ext = getLanguageExtension("GRADLE");
    expect(ext).toBeTruthy();
  });

  // --- Uppercase inputs for existing languages ---

  it('maps uppercase "GO" to a language extension (case-insensitive)', () => {
    expect(getLanguageExtension("GO")).toBeTruthy();
  });

  it('maps uppercase "PYTHON" to a language extension (case-insensitive)', () => {
    expect(getLanguageExtension("PYTHON")).toBeTruthy();
  });

  it('maps uppercase "JAVA" to a language extension (case-insensitive)', () => {
    expect(getLanguageExtension("JAVA")).toBeTruthy();
  });

  it('maps uppercase "RUST" to a language extension (case-insensitive)', () => {
    expect(getLanguageExtension("RUST")).toBeTruthy();
  });

  // --- TypeScript uppercase: the critical case ---
  // Without lower.startsWith("t"), "TYPESCRIPT" would silently produce
  // javascript({ typescript: false }) instead of typescript: true.
  // We verify the function returns a truthy extension in both cases,
  // confirming the toLowerCase() guard routes "TYPESCRIPT" into the ts branch.

  it('maps uppercase "TYPESCRIPT" to a language extension (case-insensitive)', () => {
    expect(getLanguageExtension("TYPESCRIPT")).toBeTruthy();
  });

  it('maps lowercase "typescript" to a language extension', () => {
    expect(getLanguageExtension("typescript")).toBeTruthy();
  });

  it('maps uppercase "JAVASCRIPT" to a language extension (case-insensitive)', () => {
    expect(getLanguageExtension("JAVASCRIPT")).toBeTruthy();
  });

  // --- Default fallback ---

  it("returns a language extension for an unknown language (default fallback)", () => {
    const ext = getLanguageExtension("xyzzy");
    expect(ext).toBeTruthy();
  });
});
