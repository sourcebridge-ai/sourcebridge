---
sourcebridge:
    page_id: test-repo.arch.internal.billing
    template: architecture
    audience: for-engineers
    dependencies:
        paths:
            - internal/billing/**
        symbols:
            - internal/billing.Charge
        dependency_scope: direct
    stale_when:
        - signature_change_in:
            - internal/billing.Charge
---
<!-- sourcebridge:block id="b43bc62604c34" kind="heading" owner="generated" -->
# Architecture: internal/billing
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b36735f506b38" kind="heading" owner="generated" -->
## Overview
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b929b2e38244d" kind="paragraph" owner="generated" -->
The internal/auth package handles authentication for inbound HTTP requests. (internal/auth/auth.go:1-10)
It validates session tokens and enforces role-based access control on all protected routes.
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b10d4dbba5e74" kind="heading" owner="generated" -->
## Key types
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b9281b564949a" kind="paragraph" owner="generated" -->
| Type | Purpose |
|---|---|
| Middleware | Wraps http.Handler with session verification (internal/auth/auth.go:12-25) |
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b0652e60921ba" kind="heading" owner="generated" -->
## Public API
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b3b59296b2d40" kind="paragraph" owner="generated" -->
Middleware is the primary entry point. Pass it any http.Handler to enable authentication. (internal/auth/auth.go:12-25)
RequireRole asserts that the authenticated user holds a given role. (internal/auth/auth.go:30-45)
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b91e6dc23928b" kind="heading" owner="generated" -->
## Dependencies
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b2a20b539178d" kind="paragraph" owner="generated" -->
- internal/jwt for token parsing (internal/jwt/jwt.go:1-50)
- internal/sessions for session lookup (internal/sessions/sessions.go:1-80)
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b0b0916100aae" kind="heading" owner="generated" -->
## Used by
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b7a53e32da842" kind="paragraph" owner="generated" -->
- internal/api/rest (internal/api/rest/rest.go:1-20)
- internal/billing (internal/billing/billing.go:1-20)
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b826794783d7f" kind="heading" owner="generated" -->
## Code example
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="be577e1fbd776" kind="code" owner="generated" -->
```go
mux.Handle("/api/", auth.Middleware(apiHandler))
```
<!-- /sourcebridge:block -->
