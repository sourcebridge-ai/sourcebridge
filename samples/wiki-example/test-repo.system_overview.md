---
sourcebridge:
    page_id: test-repo.system_overview
    template: system_overview
    audience: for-product
    dependencies:
        paths:
            - '**'
        downstream_packages:
            - internal/auth
            - internal/billing
        dependency_scope: transitive
---
<!-- sourcebridge:block id="bab62250da483" kind="heading" owner="generated" -->
# System Overview
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bba58967bcea2" kind="heading" owner="generated" -->
## Overview
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b156c46eb1ef7" kind="paragraph" owner="generated" -->
The internal/auth package handles authentication for inbound HTTP requests. (internal/auth/auth.go:1-10)
It validates session tokens and enforces role-based access control on all protected routes.
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b721ffba2847d" kind="heading" owner="generated" -->
## Key types
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bd60d1e1d4931" kind="paragraph" owner="generated" -->
| Type | Purpose |
|---|---|
| Middleware | Wraps http.Handler with session verification (internal/auth/auth.go:12-25) |
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bb6fbc3c6c3ab" kind="heading" owner="generated" -->
## Public API
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b9331091a3fd0" kind="paragraph" owner="generated" -->
Middleware is the primary entry point. Pass it any http.Handler to enable authentication. (internal/auth/auth.go:12-25)
RequireRole asserts that the authenticated user holds a given role. (internal/auth/auth.go:30-45)
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b8c8c06bb4273" kind="heading" owner="generated" -->
## Dependencies
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b4df744340d1d" kind="paragraph" owner="generated" -->
- internal/jwt for token parsing (internal/jwt/jwt.go:1-50)
- internal/sessions for session lookup (internal/sessions/sessions.go:1-80)
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b1ded7566ef44" kind="heading" owner="generated" -->
## Used by
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bc5a72719617c" kind="paragraph" owner="generated" -->
- internal/api/rest (internal/api/rest/rest.go:1-20)
- internal/billing (internal/billing/billing.go:1-20)
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b38fe2f259e54" kind="heading" owner="generated" -->
## Code example
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bd5144c6a838d" kind="paragraph" owner="generated" -->
```go
mux.Handle("/api/", auth.Middleware(apiHandler))
```
<!-- /sourcebridge:block -->
