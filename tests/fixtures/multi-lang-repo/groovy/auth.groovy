// Mirror of the token-verifier fixture shape used by the other languages
// (see ruby/auth.rb, php/Auth.php, cpp/auth.cpp). Same symbol set so the
// parser test can assert parity with the house style.

import foo.util.Base64Support
import static foo.Time.now

/**
 * Verifies HMAC-signed tokens.
 */
class HmacTokenVerifier {
    String realm
    String secret

    HmacTokenVerifier(String realm, String secret) {
        this.realm = realm
        this.secret = secret
    }

    boolean verify(String token) {
        def parts = token.split('\\.')
        return parts.size() == 2
    }

    String issue(String subject) {
        return "${realm}.${subject}"
    }
}

class TokenVerifier {
    // Marker class — in a real codebase this would be an interface,
    // but we also want to exercise the class classification path for a
    // plain class body with no methods.
}

// Top-level function — `def` at source level. Used to assert bare-def
// extraction separate from class methods.
def default_verifier() {
    return new HmacTokenVerifier("default", "secret")
}

// Closure assigned to a top-level variable. Not expected to register as
// a function symbol — tree-sitter emits this as a declaration, which is
// consistent with how field-level closures are handled in other
// languages. Exercise the call extraction when invoked below.
def onInit = { realm, secret ->
    println "init $realm"
}

onInit("default", "secret")
