---
sourcebridge:
    page_id: test-repo.arch.internal.auth
    template: architecture
    audience: for-engineers
    dependencies:
        paths:
            - internal/auth/**
        symbols:
            - internal/auth.Middleware
        dependency_scope: direct
    stale_when:
        - signature_change_in:
            - internal/auth.Middleware
---
<!-- sourcebridge:block id="b5dbbf7e53a7a" kind="stale_banner" owner="generated" -->
> ⚠️ **This page may be out of date.** Recent changes to `auth.Middleware` (commit `f3a9b1d`) may affect this content. [Refresh from source](https://app.sourcebridge.ai/repos/test-repo/pages/test-repo.arch.internal.auth/refresh). Next scheduled regen: in 2 hours.
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bd5264bb4621c" kind="heading" owner="generated" -->
# Architecture: internal/auth
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b01b419c77bc4" kind="heading" owner="generated" -->
## Overview
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b56b628a603e4" kind="paragraph" owner="generated" -->
The internal/auth package handles authentication for inbound HTTP requests. (internal/auth/auth.go:1-10)
It validates session tokens and enforces role-based access control on all protected routes.
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bfebdc603a322" kind="heading" owner="generated" -->
## Key types
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b72880f407799" kind="paragraph" owner="generated" -->
| Type | Purpose |
|---|---|
| Middleware | Wraps http.Handler with session verification (internal/auth/auth.go:12-25) |
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b20650a0485b6" kind="heading" owner="generated" -->
## Public API
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b60839db93e77" kind="paragraph" owner="generated" -->
Middleware is the primary entry point. Pass it any http.Handler to enable authentication. (internal/auth/auth.go:12-25)
RequireRole asserts that the authenticated user holds a given role. (internal/auth/auth.go:30-45)
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b096ea048fa2d" kind="heading" owner="generated" -->
## Dependencies
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b9f5b25d3e093" kind="paragraph" owner="generated" -->
- internal/jwt for token parsing (internal/jwt/jwt.go:1-50)
- internal/sessions for session lookup (internal/sessions/sessions.go:1-80)
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bacffad92028f" kind="heading" owner="generated" -->
## Used by
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b01585d9a2050" kind="paragraph" owner="generated" -->
- internal/api/rest (internal/api/rest/rest.go:1-20)
- internal/billing (internal/billing/billing.go:1-20)
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b5a2f99050da4" kind="heading" owner="generated" -->
## Code example
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b28062c4c1e10" kind="code" owner="generated" -->
```go
mux.Handle("/api/", auth.Middleware(apiHandler))
```
<!-- /sourcebridge:block -->
