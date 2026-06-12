// Spock spec fixture. The grammar emits ERROR nodes for `def "should …"()`
// blocks (documented in NODES.md), so Spock method names are NOT
// extractable via tree-sitter. Test-file detection for Spock specs falls
// back to the filename pattern + adjacent-test heuristic.
//
// This fixture exists primarily to confirm:
//   1. The `*Spec.groovy` filename pattern is recognized as a test file.
//   2. The enclosing FooSpec class IS captured (class_definition parses
//      cleanly even though the def-string method bodies do not).

import spock.lang.Specification

class FooSpec extends Specification {

    def "should return the realm verbatim"() {
        given:
        def verifier = new HmacTokenVerifier("acme", "secret")

        when:
        def result = verifier.issue("user-1")

        then:
        result == "acme.user-1"
    }

    def "should reject malformed tokens"() {
        expect:
        !new HmacTokenVerifier("acme", "secret").verify("not-a-token")
    }
}
