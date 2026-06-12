// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package qa

import (
	"strings"
	"testing"
)

func TestAdjacentTestCandidates_Go(t *testing.T) {
	got := adjacentTestCandidates("internal/qa/pipeline.go")
	mustContain(t, got, "internal/qa/pipeline_test.go")
}

func TestAdjacentTestCandidates_Python(t *testing.T) {
	got := adjacentTestCandidates("workers/foo/bar.py")
	mustContain(t, got, "workers/foo/test_bar.py")
	mustContain(t, got, "workers/foo/bar_test.py")
	mustContain(t, got, "workers/foo/tests/test_bar.py")
}

func TestAdjacentTestCandidates_TypeScript(t *testing.T) {
	got := adjacentTestCandidates("web/src/auth/session.ts")
	mustContain(t, got, "web/src/auth/session.test.ts")
	mustContain(t, got, "web/src/auth/session.spec.ts")
	mustContain(t, got, "web/src/auth/__tests__/session.test.ts")
}

func TestIsTestFile(t *testing.T) {
	for _, p := range []string{
		"internal/qa/pipeline_test.go",
		"workers/tests/test_cli_ask.py",
		"web/src/auth/session.test.ts",
		"web/src/auth/__tests__/session.spec.tsx",
		"src/test/java/com/foo/AuthTest.java",
	} {
		if !isTestFile(p) {
			t.Errorf("expected %q to be recognized as a test file", p)
		}
	}
	for _, p := range []string{
		"internal/qa/pipeline.go",
		"workers/foo/bar.py",
		"web/src/auth/session.ts",
	} {
		if isTestFile(p) {
			t.Errorf("%q should not match test patterns", p)
		}
	}
}

func TestFindTestFunctionsGo(t *testing.T) {
	body := `package foo

import "testing"

func TestAlpha(t *testing.T) {
	if 1 + 1 != 2 {
		t.Errorf("math")
	}
}

func TestBeta(t *testing.T) {
	t.Fatal("no")
}

func helper() {}
`
	frames := findTestFunctions("foo_test.go", body)
	if len(frames) != 2 {
		t.Fatalf("expected 2 test frames, got %d", len(frames))
	}
	if frames[0].TestName != "TestAlpha" {
		t.Errorf("wrong first test name: %s", frames[0].TestName)
	}
	if frames[1].TestName != "TestBeta" {
		t.Errorf("wrong second test name: %s", frames[1].TestName)
	}
	// First test should span lines 5..9 (func TestAlpha through closing brace).
	if frames[0].EndLine-frames[0].StartLine < 3 {
		t.Errorf("end-of-block too tight: %d-%d", frames[0].StartLine, frames[0].EndLine)
	}
}

func TestFindTestFunctionsPython(t *testing.T) {
	body := `import pytest

def test_alpha():
    assert 1 + 1 == 2
    assert "foo" in "foobar"

def test_beta():
    pass

def regular_helper():
    return 1
`
	frames := findTestFunctions("test_foo.py", body)
	if len(frames) != 2 {
		t.Fatalf("expected 2 python test frames, got %d", len(frames))
	}
}

func TestExtractAssertions(t *testing.T) {
	body := `func TestFoo(t *testing.T) {
	if x != y {
		t.Errorf("expected equal")
	}
	assert.Equal(t, x, y)
	expect(x).toBe(y)
}`
	asserts := extractAssertions(body)
	if len(asserts) == 0 {
		t.Fatal("expected at least one assertion extracted")
	}
	joined := strings.Join(asserts, "|")
	// At least one of the recognized patterns should be present.
	if !strings.Contains(joined, "Errorf") && !strings.Contains(joined, "assert") && !strings.Contains(joined, "expect") {
		t.Errorf("no recognized assertion pattern: %v", asserts)
	}
}

func TestBuildTestHandle(t *testing.T) {
	fr := testFrame{
		FilePath:  "internal/qa/pipeline_test.go",
		StartLine: 10,
		EndLine:   25,
		TestName:  "TestOrchestrator",
	}
	got := buildTestHandle(fr)
	want := "test:internal/qa/pipeline_test.go:10-25"
	if got != want {
		t.Errorf("handle = %q, want %q", got, want)
	}
}

func mustContain(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, h := range haystack {
		if h == needle {
			return
		}
	}
	t.Errorf("missing %q in %v", needle, haystack)
}

// ---------------------------------------------------------------------------
// Groovy / Spock tests (Phase 3 — M-NEW, M13, M14/M15, M16)
// ---------------------------------------------------------------------------

// TestIsTestFile_GroovySpock confirms that Groovy Spock + JUnit test file
// naming conventions are recognized by isTestFile. This matters because
// augmentFromSearch (test_finder.go) gates search hits through isTestFile —
// without M-NEW patterns, Groovy test files are invisible to that path.
func TestIsTestFile_GroovySpock(t *testing.T) {
	positives := []string{
		"src/test/groovy/example/FooSpec.groovy",
		"FooSpec.groovy",
		"FooTest.groovy",
		"FooTests.groovy",
	}
	for _, p := range positives {
		if !isTestFile(p) {
			t.Errorf("expected %q to be recognized as a test file", p)
		}
	}
	negatives := []string{
		"Foo.groovy",
		"build.gradle",
		"src/main/groovy/example/Foo.groovy",
	}
	for _, p := range negatives {
		if isTestFile(p) {
			t.Errorf("%q should NOT match test patterns", p)
		}
	}
}

// TestAdjacentTestCandidates_Groovy covers the src/main/groovy mirror rule
// and the grails-app/ no-op guard (strings.Replace is a no-op when
// "src/main/groovy" is absent, so the source path is NOT added as a candidate).
func TestAdjacentTestCandidates_Groovy(t *testing.T) {
	// src/main/groovy path — mirror rule fires.
	got := adjacentTestCandidates("src/main/groovy/example/Foo.groovy")
	mustContain(t, got, "src/test/groovy/example/Foo.groovy")
	mustContain(t, got, "src/main/groovy/example/FooSpec.groovy")
	mustContain(t, got, "src/main/groovy/example/FooTest.groovy")

	// grails-app/ path — strings.Replace is a no-op; the source path itself
	// must NOT appear as a candidate (it's not a test file).
	gotGrails := adjacentTestCandidates("grails-app/services/example/FooService.groovy")
	for _, c := range gotGrails {
		if c == "grails-app/services/example/FooService.groovy" {
			t.Errorf("source path %q must not appear as a test candidate", c)
		}
	}
	// Spec and Test siblings should still be present.
	mustContain(t, gotGrails, "grails-app/services/example/FooServiceSpec.groovy")
	mustContain(t, gotGrails, "grails-app/services/example/FooServiceTest.groovy")
}

// TestFindTestFunctionsGroovy exercises both Spock (def "name"()) via group 1
// and JUnit-style (def testX()) via group 2, and the brace-counting endOfBlock
// path for .groovy. Both should produce correctly-named frames.
func TestFindTestFunctionsGroovy(t *testing.T) {
	body := `import spock.lang.Specification

class FooSpec extends Specification {
    def "should add numbers"() {
        expect:
        1 + 1 == 2
    }

    def testLegacy() {
        assert true
    }

    def helper() {}
}
`
	frames := findTestFunctions("FooSpec.groovy", body)
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d: %v", len(frames), frames)
	}

	// Spock quoted name — group 1.
	if frames[0].TestName != "should add numbers" {
		t.Errorf("frame[0].TestName = %q, want %q", frames[0].TestName, "should add numbers")
	}

	// JUnit-style identifier — group 2.
	if frames[1].TestName != "testLegacy" {
		t.Errorf("frame[1].TestName = %q, want %q", frames[1].TestName, "testLegacy")
	}

	// endOfBlock brace-counting: the Spock method should span > 1 line.
	if frames[0].EndLine-frames[0].StartLine < 2 {
		t.Errorf("endOfBlock too tight for Spock method: start=%d end=%d", frames[0].StartLine, frames[0].EndLine)
	}
}
