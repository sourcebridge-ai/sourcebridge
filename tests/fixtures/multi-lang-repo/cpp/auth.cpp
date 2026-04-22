// C++ fixture for the tree-sitter indexer.
//
// Exercises: preprocessor include, class with methods, struct, free
// function, and a qualified method definition.

#include <string>
#include "session_store.h"

namespace app::auth {

struct Credentials {
    std::string username;
    std::string secret;
};

class TokenVerifier {
public:
    TokenVerifier(const std::string& realm);
    bool verify(const std::string& token, const std::string& secret) const;
    std::string issue(const std::string& subject, const std::string& secret) const;

private:
    std::string realm_;
};

TokenVerifier::TokenVerifier(const std::string& realm) : realm_(realm) {}

bool TokenVerifier::verify(const std::string& token, const std::string& secret) const {
    return token == issue(secret, realm_);
}

std::string TokenVerifier::issue(const std::string& subject, const std::string& secret) const {
    return realm_ + ":" + subject + ":" + secret;
}

// Free function: default verifier for the "app" realm.
TokenVerifier default_verifier() {
    return TokenVerifier("app");
}

}  // namespace app::auth
