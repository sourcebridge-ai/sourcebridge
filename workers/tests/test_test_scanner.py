"""Tests for TestFileScanner — Groovy/Spock + general coverage.

Created in Phase 4 of the Groovy/Grails port (CA-TBD).  Previously the
test-scanner had zero test coverage; this file bootstraps it.
"""

from __future__ import annotations

from common.v1 import types_pb2

from workers.linking.servicer import _language_name as linking_language_name
from workers.reasoning.servicer import _language_name as reasoning_language_name
from workers.requirements.scanners.test_scanner import TEST_FILE_PATTERNS, TestFileScanner

# ---------------------------------------------------------------------------
# is_test_file — Groovy file name patterns
# ---------------------------------------------------------------------------


def test_is_test_file_spock_spec():
    """FooSpec.groovy is recognised as a Groovy test file."""
    scanner = TestFileScanner()
    assert scanner.is_test_file("src/test/groovy/example/FooSpec.groovy", "groovy") is True


def test_is_test_file_test_suffix():
    """FooTest.groovy is recognised as a Groovy test file."""
    scanner = TestFileScanner()
    assert scanner.is_test_file("src/test/groovy/example/FooTest.groovy", "groovy") is True


def test_is_test_file_tests_suffix():
    """FooTests.groovy is recognised as a Groovy test file."""
    scanner = TestFileScanner()
    assert scanner.is_test_file("src/test/groovy/example/FooTests.groovy", "groovy") is True


def test_is_test_file_plain_groovy_not_test():
    """A plain Groovy source file is NOT a test file."""
    scanner = TestFileScanner()
    assert scanner.is_test_file("src/main/groovy/example/FooService.groovy", "groovy") is False


def test_is_test_file_gradle_not_test():
    """A build.gradle file is NOT a test file."""
    scanner = TestFileScanner()
    assert scanner.is_test_file("build.gradle", "groovy") is False


# ---------------------------------------------------------------------------
# TEST_FILE_PATTERNS has groovy key
# ---------------------------------------------------------------------------


def test_test_file_patterns_has_groovy():
    """TEST_FILE_PATTERNS contains a groovy entry."""
    assert "groovy" in TEST_FILE_PATTERNS
    patterns = TEST_FILE_PATTERNS["groovy"]
    assert any("Spec.groovy" in p for p in patterns)
    assert any("Test.groovy" in p for p in patterns)


# ---------------------------------------------------------------------------
# TEST_PATTERNS — Spock quoted-name extraction (group 1)
# ---------------------------------------------------------------------------


def test_spock_quoted_name_group1():
    """Spock 'def "should count"()' extracts the quoted name via group 1."""
    scanner = TestFileScanner()

    content = '''\
class CounterSpec extends Specification {
    def "should count items correctly"() {
        expect:
            counter.count() == 3
    }

    def "should reset to zero"() {
        when:
            counter.reset()
        then:
            counter.count() == 0
    }
}
'''
    candidates = scanner.extract(
        "src/test/groovy/example/CounterSpec.groovy",
        content,
        "groovy",
        all_files=[],
    )

    names = [c.metadata["test_name"] for c in candidates]
    assert "should count items correctly" in names
    assert "should reset to zero" in names


# ---------------------------------------------------------------------------
# TEST_PATTERNS — JUnit-style def testFoo() fallback (group 2)
# ---------------------------------------------------------------------------


def test_junit_style_def_test_group2():
    """JUnit-style 'def testCount()' extracts the method name via group 2."""
    scanner = TestFileScanner()

    content = '''\
class FooTest {
    def testCount() {
        assert service.count() == 5
    }

    void testReset() {
        service.reset()
        assert service.count() == 0
    }
}
'''
    candidates = scanner.extract(
        "src/test/groovy/example/FooTest.groovy",
        content,
        "groovy",
        all_files=[],
    )

    names = [c.metadata["test_name"] for c in candidates]
    assert "testCount" in names
    assert "testReset" in names


# ---------------------------------------------------------------------------
# _resolve_file_under_test — Groovy: src/test/groovy -> src/main/groovy
# ---------------------------------------------------------------------------


def test_resolve_file_under_test_groovy_spec():
    """FooSpec.groovy in test tree resolves to Foo.groovy in main tree."""
    scanner = TestFileScanner()

    test_file = "src/test/groovy/example/FooSpec.groovy"
    repo_files = [
        "src/main/groovy/example/Foo.groovy",
        "src/test/groovy/example/FooSpec.groovy",
        "build.gradle",
    ]

    result = scanner._resolve_file_under_test(test_file, "groovy", repo_files)

    assert result == "src/main/groovy/example/Foo.groovy"


def test_resolve_file_under_test_groovy_test_suffix():
    """FooTest.groovy in test tree resolves to Foo.groovy in main tree."""
    scanner = TestFileScanner()

    test_file = "src/test/groovy/example/FooTest.groovy"
    repo_files = [
        "src/main/groovy/example/Foo.groovy",
        "src/test/groovy/example/FooTest.groovy",
    ]

    result = scanner._resolve_file_under_test(test_file, "groovy", repo_files)

    assert result == "src/main/groovy/example/Foo.groovy"


def test_resolve_file_under_test_groovy_tests_suffix():
    """FooTests.groovy in test tree resolves to Foo.groovy in main tree."""
    scanner = TestFileScanner()

    test_file = "src/test/groovy/example/FooTests.groovy"
    repo_files = [
        "src/main/groovy/example/Foo.groovy",
    ]

    result = scanner._resolve_file_under_test(test_file, "groovy", repo_files)

    assert result == "src/main/groovy/example/Foo.groovy"


def test_resolve_file_under_test_groovy_no_main_fallback():
    """When the main-tree file is absent, falls back to the first computed candidate."""
    scanner = TestFileScanner()

    test_file = "src/test/groovy/example/FooSpec.groovy"
    repo_files: list[str] = []  # nothing present

    result = scanner._resolve_file_under_test(test_file, "groovy", repo_files)

    # Should return the first candidate path (even though it doesn't exist in repo)
    assert result == "src/main/groovy/example/Foo.groovy"


# ---------------------------------------------------------------------------
# Servicer _LANGUAGE_MAP round-trips: LANGUAGE_GROOVY -> "groovy"
# (codex Low — verify both servicers agree with the Go side)
# ---------------------------------------------------------------------------


def test_linking_servicer_language_groovy():
    """linking servicer: LANGUAGE_GROOVY maps to 'groovy'."""
    assert linking_language_name(types_pb2.LANGUAGE_GROOVY) == "groovy"


def test_reasoning_servicer_language_groovy():
    """reasoning servicer: LANGUAGE_GROOVY maps to 'groovy'."""
    assert reasoning_language_name(types_pb2.LANGUAGE_GROOVY) == "groovy"


def test_linking_servicer_unknown_language():
    """linking servicer: an unknown enum value maps to 'unknown'."""
    assert linking_language_name(9999) == "unknown"


def test_reasoning_servicer_unknown_language():
    """reasoning servicer: an unknown enum value maps to 'unknown'."""
    assert reasoning_language_name(9999) == "unknown"
