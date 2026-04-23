# Authenticates users against the internal session store.
#
# Two independent public methods are defined here so the indexer has
# multiple top-level symbols to extract, plus a class with an
# instance method to exercise the method query.

require "digest"
require_relative "session_store"

module Auth
  DEFAULT_REALM = "app"

  class TokenVerifier
    def initialize(realm: DEFAULT_REALM)
      @realm = realm
    end

    # Verify the HMAC signature on a signed token.
    def verify(token, secret)
      expected = Digest::SHA256.hexdigest("#{@realm}:#{secret}:#{token}")
      token == expected
    end

    def issue(subject, secret)
      Digest::SHA256.hexdigest("#{@realm}:#{secret}:#{subject}")
    end
  end

  # Module-level helper not tied to the class.
  def self.realm_for(request)
    request[:realm] || DEFAULT_REALM
  end
end
